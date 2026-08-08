package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jobright/api/pkg/response"
)

const UserIDKey = "userID"

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// NormalizeOrigin strips quotes/whitespace/trailing slash so env typos still match.
func NormalizeOrigin(o string) string {
	o = strings.TrimSpace(o)
	o = strings.Trim(o, `"'`)
	return strings.TrimRight(o, "/")
}

func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	allowAll := len(origins) == 0
	for _, o := range origins {
		o = NormalizeOrigin(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[o] = struct{}{}
	}
	if len(allowed) == 0 {
		allowAll = true
	}

	return func(c *gin.Context) {
		origin := NormalizeOrigin(c.GetHeader("Origin"))
		if origin != "" && (allowAll || originAllowed(allowed, origin)) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			// Credentials unused by the web app (Bearer token), but safe when echoing a concrete origin.
			c.Header("Access-Control-Allow-Credentials", "true")
		} else if origin == "" && allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func originAllowed(allowed map[string]struct{}, origin string) bool {
	if _, ok := allowed[origin]; ok {
		return true
	}
	// Allow Vercel preview URLs when any *.vercel.app entry is configured.
	if _, ok := allowed["*.vercel.app"]; ok && strings.HasSuffix(origin, ".vercel.app") && strings.HasPrefix(origin, "https://") {
		return true
	}
	return false
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Unauthorized(c, "missing bearer token")
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			response.Unauthorized(c, "invalid token")
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func UserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(UserIDKey)
	id, _ := v.(uuid.UUID)
	return id
}
