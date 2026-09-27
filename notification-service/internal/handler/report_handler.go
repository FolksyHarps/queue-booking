package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/notification-service/internal/middleware"
	"github.com/queue-booking/notification-service/internal/service"
)

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(r *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: r}
}

// GET /api/reports/summary?start_date&end_date (admin)
func (h *ReportHandler) Summary(c *gin.Context) {
	dr, ok := dateRange(c)
	if !ok {
		return
	}
	sum, err := h.reports.Summary(middleware.CurrentUser(c).ID, c.GetHeader("Authorization"), dr)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, sum)
}

type rateQuery struct {
	ServiceID uint `form:"service_id"`
}

// GET /api/reports/cancellation-rate?service_id (admin)
// With service_id: {"service_id", "cancellation_rate"}.
// Without: {"services": [{"service_id", "cancellation_rate"}, ...]} for every service.
func (h *ReportHandler) CancellationRate(c *gin.Context) {
	var q rateQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_id must be a positive integer"})
		return
	}
	rates, err := h.reports.CancellationRates(middleware.CurrentUser(c).ID, c.GetHeader("Authorization"), q.ServiceID)
	if err != nil {
		writeError(c, err)
		return
	}
	if q.ServiceID != 0 {
		c.JSON(http.StatusOK, rates[0])
		return
	}
	c.JSON(http.StatusOK, gin.H{"services": rates})
}

type exportQuery struct {
	Type string `form:"type" binding:"required,oneof=bookings cancellations"`
}

// GET /api/reports/export?type&start_date&end_date (admin) -> report.csv
func (h *ReportHandler) Export(c *gin.Context) {
	var q exportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be bookings or cancellations"})
		return
	}
	dr, ok := dateRange(c)
	if !ok {
		return
	}
	data, err := h.reports.Export(middleware.CurrentUser(c).ID, c.GetHeader("Authorization"), q.Type, dr)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="report.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

func dateRange(c *gin.Context) (service.DateRange, bool) {
	dr := service.DateRange{Start: c.Query("start_date"), End: c.Query("end_date")}
	for _, d := range []string{dr.Start, dr.End} {
		if d == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_date/end_date must be YYYY-MM-DD"})
			return dr, false
		}
	}
	return dr, true
}
