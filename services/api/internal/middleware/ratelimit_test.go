package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsThenBlocks(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		ok, _, rem := rl.allow("user:1")
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
		if rem != 3-(i+1) {
			t.Fatalf("remaining want %d got %d", 3-(i+1), rem)
		}
	}
	ok, retry, rem := rl.allow("user:1")
	if ok {
		t.Fatal("4th request should be blocked")
	}
	if rem != 0 {
		t.Fatalf("remaining want 0 got %d", rem)
	}
	if retry < time.Second {
		t.Fatalf("retry-after too small: %v", retry)
	}
	// Different key is independent.
	ok, _, _ = rl.allow("user:2")
	if !ok {
		t.Fatal("other user should be allowed")
	}
}
