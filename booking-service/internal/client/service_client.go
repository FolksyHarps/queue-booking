package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ErrServiceNotFound = errors.New("service not found")

type ServiceInfo struct {
	ID          uint `json:"id"`
	ProviderID  uint `json:"provider_id"`
	DurationMin int  `json:"duration_min"`
}

// ServiceClient talks to service-service's public API.
type ServiceClient struct {
	baseURL string
	http    *http.Client
}

func NewServiceClient(baseURL string) *ServiceClient {
	return &ServiceClient{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *ServiceClient) Get(id uint) (*ServiceInfo, error) {
	var s ServiceInfo
	if err := c.getJSON(fmt.Sprintf("/api/services/%d", id), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Slots returns the bookable start times ("HH:MM") of a service on a date.
func (c *ServiceClient) Slots(id uint, date string) ([]string, error) {
	var out struct {
		Slots []string `json:"slots"`
	}
	err := c.getJSON(fmt.Sprintf("/api/services/%d/availability?date=%s", id, url.QueryEscape(date)), &out)
	return out.Slots, err
}

// IDsByProvider returns the ids of all services owned by a provider.
func (c *ServiceClient) IDsByProvider(providerID uint) ([]uint, error) {
	var out struct {
		Services []struct {
			ID uint `json:"id"`
		} `json:"services"`
	}
	if err := c.getJSON("/api/services?provider_id="+strconv.FormatUint(uint64(providerID), 10), &out); err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(out.Services))
	for _, s := range out.Services {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

func (c *ServiceClient) getJSON(path string, dst any) error {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("service-service: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(dst)
	case http.StatusNotFound:
		return ErrServiceNotFound
	default:
		return fmt.Errorf("service-service: unexpected status %d for %s", resp.StatusCode, path)
	}
}
