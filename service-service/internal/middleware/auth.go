package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/service-service/internal/client"
)

const ctxUser = "user"

// RequireAuth resolves the Bearer token via auth-service.
func RequireAuth(auth *client.AuthClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		u, err := auth.Me(h)
		if errors.Is(err, client.ErrUnauthorized) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if err != nil {
			log.Printf("auth check failed: %v", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
			return
		}
		c.Set(ctxUser, u)
		c.Next()
	}
}

// RequireRole must run after RequireAuth.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		for _, r := range roles {
			if u.Role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}

func CurrentUser(c *gin.Context) *client.CurrentUser {
	return c.MustGet(ctxUser).(*client.CurrentUser)
}
