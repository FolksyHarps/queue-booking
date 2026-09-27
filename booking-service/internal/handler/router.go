package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/booking-service/internal/client"
	"github.com/queue-booking/booking-service/internal/middleware"
	"github.com/queue-booking/booking-service/internal/service"
)

func NewRouter(bookings *service.BookingService, auth *client.AuthClient, internalKey string) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	h := NewBookingHandler(bookings)
	requireAuth := middleware.RequireAuth(auth)

	api := r.Group("/api")
	api.GET("/bookings", requireAuth, h.List)
	api.GET("/bookings/:id", middleware.RequireAuthOrInternal(auth, internalKey), h.Get)
	api.POST("/bookings", requireAuth, h.Create)
	api.PUT("/bookings/:id/cancel", requireAuth, h.Cancel)
	api.PUT("/bookings/:id/reschedule", requireAuth, h.Reschedule)
	api.PUT("/bookings/:id/status", requireAuth, middleware.RequireRole("provider", "admin"), h.UpdateStatus)
	api.GET("/queue/:serviceId", h.Queue)
	return r
}
