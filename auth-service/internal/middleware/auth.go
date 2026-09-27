package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/service"
)

const (
	ctxUser = "user"
	ctxJTI  = "jti"
	ctxExp  = "exp"
)

// RequireAuth validates the Bearer token and stores the current user in the context.
func RequireAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		u, jti, exp, err := auth.Authenticate(token)
		if err != nil {
			msg := "invalid or expired token"
			if errors.Is(err, service.ErrAccountSuspended) {
				msg = "account suspended"
			} else if !errors.Is(err, service.ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}
		c.Set(ctxUser, u)
		c.Set(ctxJTI, jti)
		c.Set(ctxExp, exp)
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

func CurrentUser(c *gin.Context) *model.User {
	return c.MustGet(ctxUser).(*model.User)
}

func TokenInfo(c *gin.Context) (jti string, exp time.Time) {
	return c.GetString(ctxJTI), c.MustGet(ctxExp).(time.Time)
}
