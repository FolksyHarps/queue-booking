package middleware

import (
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/booking-service/internal/client"
)

const (
	ctxUser = "user"
	// RoleSystem marks internal service-to-service calls (X-Internal-Key).
	RoleSystem = "system"
)

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

// RequireAuthOrInternal also accepts a valid X-Internal-Key header, used by
// notification-service to read booking details.
func RequireAuthOrInternal(auth *client.AuthClient, internalKey string) gin.HandlerFunc {
	userAuth := RequireAuth(auth)
	return func(c *gin.Context) {
		key := c.GetHeader("X-Internal-Key")
		if key != "" && internalKey != "" && subtle.ConstantTimeCompare([]byte(key), []byte(internalKey)) == 1 {
			c.Set(ctxUser, &client.CurrentUser{Role: RoleSystem})
			c.Next()
			return
		}
		userAuth(c)
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
