package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/queue-booking/notification-service/internal/client"
	"github.com/queue-booking/notification-service/internal/model"
	"github.com/queue-booking/notification-service/internal/repository"
)

// ReportService aggregates booking data fetched from booking-service using the
// requesting admin's token, and records each report in report_logs.
type ReportService struct {
	repo      *repository.Repository
	bookings  *client.BookingClient
	reportDir string
}

func NewReportService(repo *repository.Repository, bookings *client.BookingClient, reportDir string) *ReportService {
	return &ReportService{repo: repo, bookings: bookings, reportDir: reportDir}
}

// DateRange filters on the booking date; empty bounds are open.
type DateRange struct{ Start, End string }

func (r DateRange) contains(date string) bool {
	return (r.Start == "" || date >= r.Start) && (r.End == "" || date <= r.End)
}

type Summary struct {
	TotalBookings int `json:"total_bookings"`
	Completed     int `json:"completed"`
	Cancelled     int `json:"cancelled"`
	NoShow        int `json:"no_show"`
}

func (s *ReportService) Summary(adminID uint, authHeader string, dr DateRange) (*Summary, error) {
	list, err := s.fetch(authHeader, dr)
	if err != nil {
		return nil, err
	}
	var sum Summary
	for _, b := range list {
		sum.TotalBookings++
		switch b.Status {
		case "completed":
			sum.Completed++
		case "cancelled":
			sum.Cancelled++
		case "no_show":
			sum.NoShow++
		}
	}
	return &sum, s.log(adminID, "summary", "")
}

type CancellationRate struct {
	ServiceID        uint    `json:"service_id"`
	CancellationRate float64 `json:"cancellation_rate"`
}

// CancellationRates returns cancelled/total per service; serviceID 0 = all services.
func (s *ReportService) CancellationRates(adminID uint, authHeader string, serviceID uint) ([]CancellationRate, error) {
	list, err := s.fetch(authHeader, DateRange{})
	if err != nil {
		return nil, err
	}
	total, cancelled := map[uint]int{}, map[uint]int{}
	for _, b := range list {
		if serviceID != 0 && b.ServiceID != serviceID {
			continue
		}
		total[b.ServiceID]++
		if b.Status == "cancelled" {
			cancelled[b.ServiceID]++
		}
	}
	if serviceID != 0 {
		total[serviceID] += 0 // report the requested service even with no bookings
	}
	out := make([]CancellationRate, 0, len(total))
	for id, n := range total {
		rate := 0.0
		if n > 0 {
			rate = math.Round(float64(cancelled[id])/float64(n)*100) / 100
		}
		out = append(out, CancellationRate{ServiceID: id, CancellationRate: rate})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServiceID < out[j].ServiceID })
	return out, s.log(adminID, "cancellation-rate", "")
}

// Export builds a CSV of bookings (or only cancelled ones), saves it under
// reportDir and returns its content.
func (s *ReportService) Export(adminID uint, authHeader, reportType string, dr DateRange) ([]byte, error) {
	list, err := s.fetch(authHeader, dr)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "service_id", "date", "time", "status"})
	for _, b := range list {
		if reportType == "cancellations" && b.Status != "cancelled" {
			continue
		}
		_ = w.Write([]string{
			strconv.FormatUint(uint64(b.ID), 10), strconv.FormatUint(uint64(b.ServiceID), 10),
			b.Date, b.Time, b.Status,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.reportDir, fmt.Sprintf("report-%s-%s.csv", reportType, time.Now().UTC().Format("20060102T150405.000")))
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	return buf.Bytes(), s.log(adminID, "export", path)
}

func (s *ReportService) fetch(authHeader string, dr DateRange) ([]client.Booking, error) {
	all, err := s.bookings.ListAll(authHeader)
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, b := range all {
		if dr.contains(b.Date) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (s *ReportService) log(adminID uint, reportType, filePath string) error {
	return s.repo.LogReport(&model.ReportLog{ReportType: reportType, GeneratedBy: adminID, FilePath: filePath})
}
