package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/auth-service/internal/middleware"
	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/service"
)

func NewRouter(auth *service.AuthService, users *service.UserService) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	ah := NewAuthHandler(auth)
	uh := NewUserHandler(users)
	requireAuth := middleware.RequireAuth(auth)
	adminOnly := middleware.RequireRole(model.RoleAdmin)

	api := r.Group("/api")
	api.POST("/auth/register", ah.Register)
	api.POST("/auth/login", ah.Login)
	api.POST("/auth/logout", requireAuth, ah.Logout)

	api.GET("/users/me", requireAuth, uh.GetMe)
	api.PUT("/users/me", requireAuth, uh.UpdateMe)
	api.GET("/users", requireAuth, adminOnly, uh.List)
	api.PUT("/users/:id/status", requireAuth, adminOnly, uh.SetStatus)
	return r
}
