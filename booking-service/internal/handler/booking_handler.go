package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/booking-service/internal/middleware"
	"github.com/queue-booking/booking-service/internal/model"
	"github.com/queue-booking/booking-service/internal/service"
)

type BookingHandler struct {
	bookings *service.BookingService
}

func NewBookingHandler(bookings *service.BookingService) *BookingHandler {
	return &BookingHandler{bookings: bookings}
}

type listQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=pending in_progress completed cancelled no_show"`
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
}

// GET /api/bookings
func (h *BookingHandler) List(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	list, err := h.bookings.List(middleware.CurrentUser(c), q.Status, q.Page, q.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, b := range list {
		items = append(items, gin.H{"id": b.ID, "service_id": b.ServiceID, "date": b.Date, "time": b.Time, "status": b.Status})
	}
	c.JSON(http.StatusOK, gin.H{"bookings": items})
}

// GET /api/bookings/:id
func (h *BookingHandler) Get(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	b, err := h.bookings.Get(middleware.CurrentUser(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": b.ID, "service_id": b.ServiceID, "customer_id": b.CustomerID,
		"date": b.Date, "time": b.Time, "status": b.Status,
	})
}

type createRequest struct {
	ServiceID uint   `json:"service_id" binding:"required"`
	Date      string `json:"date" binding:"required"`
	Time      string `json:"time" binding:"required"`
}

// POST /api/bookings
func (h *BookingHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := h.bookings.Create(middleware.CurrentUser(c), req.ServiceID, req.Date, req.Time)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, bookingJSON(b))
}

// PUT /api/bookings/:id/cancel
func (h *BookingHandler) Cancel(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	b, err := h.bookings.Cancel(middleware.CurrentUser(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": b.ID, "status": b.Status})
}

type rescheduleRequest struct {
	NewDate string `json:"new_date" binding:"required"`
	NewTime string `json:"new_time" binding:"required"`
}

// PUT /api/bookings/:id/reschedule
func (h *BookingHandler) Reschedule(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req rescheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := h.bookings.Reschedule(middleware.CurrentUser(c), id, req.NewDate, req.NewTime)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": b.ID, "date": b.Date, "time": b.Time, "status": b.Status})
}

type statusRequest struct {
	Status string `json:"status" binding:"required,oneof=in_progress completed no_show"`
}

// PUT /api/bookings/:id/status (provider/admin)
func (h *BookingHandler) UpdateStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := h.bookings.UpdateStatus(middleware.CurrentUser(c), id, req.Status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": b.ID, "status": b.Status})
}

// GET /api/queue/:serviceId
func (h *BookingHandler) Queue(c *gin.Context) {
	id, ok := pathID(c, "serviceId")
	if !ok {
		return
	}
	q, err := h.bookings.Queue(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, q)
}

func bookingJSON(b *model.Booking) gin.H {
	return gin.H{"id": b.ID, "service_id": b.ServiceID, "date": b.Date, "time": b.Time, "status": b.Status}
}

func pathID(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return uint(id), true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound), errors.Is(err, service.ErrServiceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrSlotUnavailable), errors.Is(err, service.ErrBadState):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		log.Printf("internal error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
