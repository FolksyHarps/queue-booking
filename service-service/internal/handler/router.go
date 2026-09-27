package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/service-service/internal/client"
	"github.com/queue-booking/service-service/internal/middleware"
	"github.com/queue-booking/service-service/internal/service"
)

func NewRouter(catalog *service.CatalogService, auth *client.AuthClient) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	h := NewServiceHandler(catalog)
	requireAuth := middleware.RequireAuth(auth)

	api := r.Group("/api/services")
	api.GET("", h.List)
	api.GET("/:id", h.Get)
	api.POST("", requireAuth, middleware.RequireRole("provider", "admin"), h.Create)
	api.PUT("/:id", requireAuth, h.Update)
	api.DELETE("/:id", requireAuth, h.Delete)
	api.POST("/:id/availability", requireAuth, h.AddAvailability)
	api.GET("/:id/availability", h.GetAvailability)
	return r
}
