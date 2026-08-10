package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RateLimiter is a simple sliding-window limiter keyed by string (usually user ID).
type RateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	limit   int
	window  time.Duration
	cleanup time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		hits:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) allow(key string) (ok bool, retryAfter time.Duration, remaining int) {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.After(rl.cleanup) {
		rl.pruneLocked(cutoff)
		rl.cleanup = now.Add(rl.window)
	}

	prev := rl.hits[key]
	recent := make([]time.Time, 0, len(prev)+1)
	for _, t := range prev {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		oldest := recent[0]
		retryAfter = oldest.Add(rl.window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		rl.hits[key] = recent
		return false, retryAfter, 0
	}

	recent = append(recent, now)
	rl.hits[key] = recent
	return true, 0, rl.limit - len(recent)
}

func (rl *RateLimiter) pruneLocked(cutoff time.Time) {
	for k, times := range rl.hits {
		kept := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(rl.hits, k)
		} else {
			rl.hits[k] = kept
		}
	}
}

// RateLimit returns Gin middleware. Key is authenticated user ID when present,
// otherwise client IP. Sets standard rate-limit response headers.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	rl := NewRateLimiter(limit, window)
	return func(c *gin.Context) {
		key := rateLimitKey(c)
		ok, retryAfter, remaining := rl.allow(key)
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Window", fmt.Sprintf("%ds", int(rl.window.Seconds())))
		if !ok {
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded — slow down AI requests",
				"retry_after": secs,
				"limit":       rl.limit,
				"window_sec":  int(rl.window.Seconds()),
			})
			return
		}
		c.Next()
	}
}

func rateLimitKey(c *gin.Context) string {
	if v, ok := c.Get(UserIDKey); ok {
		switch id := v.(type) {
		case uuid.UUID:
			return "user:" + id.String()
		case string:
			if id != "" {
				return "user:" + id
			}
		}
	}
	return "ip:" + c.ClientIP()
}
