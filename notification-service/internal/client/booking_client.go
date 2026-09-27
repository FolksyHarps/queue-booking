package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrBookingNotFound = errors.New("booking not found")

type Booking struct {
	ID         uint   `json:"id"`
	CustomerID uint   `json:"customer_id"`
	ServiceID  uint   `json:"service_id"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	Status     string `json:"status"`
}

// BookingClient reads data from booking-service.
type BookingClient struct {
	baseURL     string
	internalKey string
	http        *http.Client
}

func NewBookingClient(baseURL, internalKey string) *BookingClient {
	return &BookingClient{baseURL: baseURL, internalKey: internalKey, http: &http.Client{Timeout: 10 * time.Second}}
}

// Get fetches one booking using the internal service key.
func (c *BookingClient) Get(id uint) (*Booking, error) {
	var b Booking
	status, err := c.getJSON(fmt.Sprintf("/api/bookings/%d", id), "X-Internal-Key", c.internalKey, &b)
	if status == http.StatusNotFound {
		return nil, ErrBookingNotFound
	}
	return &b, err
}

// ListAll pages through GET /api/bookings with the caller's (admin) token.
func (c *BookingClient) ListAll(authHeader string) ([]Booking, error) {
	const limit = 100
	var all []Booking
	for page := 1; ; page++ {
		var out struct {
			Bookings []Booking `json:"bookings"`
		}
		path := fmt.Sprintf("/api/bookings?page=%d&limit=%d", page, limit)
		if _, err := c.getJSON(path, "Authorization", authHeader, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Bookings...)
		if len(out.Bookings) < limit {
			return all, nil
		}
	}
}

func (c *BookingClient) getJSON(path, header, value string, dst any) (int, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set(header, value)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("booking-service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("booking-service: unexpected status %d for %s", resp.StatusCode, path)
	}
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(dst)
}
