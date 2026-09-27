package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/service-service/internal/middleware"
	"github.com/queue-booking/service-service/internal/model"
	"github.com/queue-booking/service-service/internal/repository"
	"github.com/queue-booking/service-service/internal/service"
)

type ServiceHandler struct {
	catalog *service.CatalogService
}

func NewServiceHandler(catalog *service.CatalogService) *ServiceHandler {
	return &ServiceHandler{catalog: catalog}
}

type listQuery struct {
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	// provider_id is not in the public spec; booking-service uses it to find
	// the services a provider owns.
	ProviderID uint `form:"provider_id"`
}

// GET /api/services
func (h *ServiceHandler) List(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	list, err := h.catalog.List(repository.ListFilter{Category: q.Category, Keyword: q.Keyword, ProviderID: q.ProviderID})
	if err != nil {
		internalError(c, err)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, s := range list {
		items = append(items, gin.H{"id": s.ID, "name": s.Name, "duration_min": s.DurationMin, "price": s.Price})
	}
	c.JSON(http.StatusOK, gin.H{"services": items})
}

// GET /api/services/:id
func (h *ServiceHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	s, err := h.catalog.Get(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": s.ID, "name": s.Name, "duration_min": s.DurationMin, "price": s.Price, "provider_id": s.ProviderID,
	})
}

type createRequest struct {
	Name        string   `json:"name" binding:"required,max=100"`
	DurationMin int      `json:"duration_min" binding:"required,min=1,max=1440"`
	Price       *float64 `json:"price" binding:"required,gte=0"`
	Category    string   `json:"category" binding:"required,max=50"`
	Description string   `json:"description"`
}

// POST /api/services (provider/admin)
func (h *ServiceHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s := &model.Service{
		Name: req.Name, DurationMin: req.DurationMin, Price: *req.Price,
		Category: req.Category, Description: req.Description,
	}
	if err := h.catalog.Create(middleware.CurrentUser(c), s); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": s.ID, "name": s.Name, "duration_min": s.DurationMin, "price": s.Price})
}

type updateRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=1,max=100"`
	DurationMin *int     `json:"duration_min" binding:"omitempty,min=1,max=1440"`
	Price       *float64 `json:"price" binding:"omitempty,gte=0"`
}

// PUT /api/services/:id (owner/admin)
func (h *ServiceHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s, err := h.catalog.Update(middleware.CurrentUser(c), id, service.UpdateInput{
		Name: req.Name, DurationMin: req.DurationMin, Price: req.Price,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "name": s.Name, "price": s.Price})
}

// DELETE /api/services/:id (owner/admin)
func (h *ServiceHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.catalog.Delete(middleware.CurrentUser(c), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type availabilityRequest struct {
	DayOfWeek *int   `json:"day_of_week" binding:"required,min=0,max=6"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// POST /api/services/:id/availability (owner/admin)
func (h *ServiceHandler) AddAvailability(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req availabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.catalog.AddAvailability(middleware.CurrentUser(c), id, *req.DayOfWeek, req.StartTime, req.EndTime)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": a.ID, "service_id": a.ServiceID, "day_of_week": a.DayOfWeek,
		"start_time": a.StartTime, "end_time": a.EndTime,
	})
}

// GET /api/services/:id/availability?date=YYYY-MM-DD
func (h *ServiceHandler) GetAvailability(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	dateStr := c.Query("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query param date (YYYY-MM-DD) is required"})
		return
	}
	slots, err := h.catalog.Slots(id, date)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"date": dateStr, "slots": slots})
}

func pathID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		internalError(c, err)
	}
}

func internalError(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
