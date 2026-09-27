package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/notification-service/internal/middleware"
	"github.com/queue-booking/notification-service/internal/service"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(n *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: n}
}

// GET /api/notifications?read=true|false
func (h *NotificationHandler) List(c *gin.Context) {
	var isRead *bool
	if v := c.Query("read"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read must be true or false"})
			return
		}
		isRead = &b
	}
	list, err := h.notifications.List(middleware.CurrentUser(c).ID, isRead)
	if err != nil {
		writeError(c, err)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, n := range list {
		items = append(items, gin.H{
			"id": n.ID, "message": n.Message, "read": n.IsRead,
			"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items})
}

type sendRequest struct {
	BookingID uint   `json:"booking_id" binding:"required"`
	Channel   string `json:"channel" binding:"required,oneof=email in-app"`
}

// POST /api/notifications/send (internal, X-Internal-Key)
func (h *NotificationHandler) Send(c *gin.Context) {
	var req sendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.notifications.Send(req.BookingID, req.Channel)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": n.ID, "booking_id": n.BookingID, "channel": n.Channel, "status": "sent"})
}

// PUT /api/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	n, err := h.notifications.MarkRead(uint(id), middleware.CurrentUser(c).ID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": n.ID, "read": n.IsRead})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("internal error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
