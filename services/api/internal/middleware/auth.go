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

func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok || len(origins) == 0 {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if len(origins) > 0 {
				c.Header("Access-Control-Allow-Origin", origins[0])
			}
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
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
