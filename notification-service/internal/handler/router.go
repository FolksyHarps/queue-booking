package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/notification-service/internal/client"
	"github.com/queue-booking/notification-service/internal/middleware"
	"github.com/queue-booking/notification-service/internal/service"
)

func NewRouter(n *service.NotificationService, r *service.ReportService, auth *client.AuthClient, internalKey string) *gin.Engine {
	e := gin.Default()
	e.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	nh := NewNotificationHandler(n)
	rh := NewReportHandler(r)
	requireAuth := middleware.RequireAuth(auth)
	adminOnly := middleware.RequireRole("admin")

	api := e.Group("/api")
	api.GET("/notifications", requireAuth, nh.List)
	api.POST("/notifications/send", middleware.RequireInternalKey(internalKey), nh.Send)
	api.PUT("/notifications/:id/read", requireAuth, nh.MarkRead)

	api.GET("/reports/summary", requireAuth, adminOnly, rh.Summary)
	api.GET("/reports/cancellation-rate", requireAuth, adminOnly, rh.CancellationRate)
	api.GET("/reports/export", requireAuth, adminOnly, rh.Export)
	return e
}
