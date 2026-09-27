package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

type CurrentUser struct {
	ID   uint   `json:"id"`
	Role string `json:"role"`
}

// AuthClient validates bearer tokens against auth-service (GET /api/users/me),
// so token revocation and account suspension take effect in every service.
type AuthClient struct {
	baseURL string
	http    *http.Client
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *AuthClient) Me(authHeader string) (*CurrentUser, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/users/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth-service: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var u CurrentUser
		if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
			return nil, fmt.Errorf("auth-service: decode: %w", err)
		}
		return &u, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("auth-service: unexpected status %d", resp.StatusCode)
	}
}
