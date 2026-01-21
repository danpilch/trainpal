package rtt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://api.rtt.io/api/v1"

// Client is a RealTimeTrains API client.
type Client struct {
	httpClient *http.Client
	username   string
	password   string
}

// NewClient creates a new RTT client.
func NewClient(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   username,
		password:   password,
	}
}

// Search finds services between two stations at a specific time.
func (c *Client) Search(ctx context.Context, from, to string, t time.Time) (*SearchResponse, error) {
	url := fmt.Sprintf("%s/json/search/%s/to/%s/%04d/%02d/%02d/%02d%02d",
		baseURL, from, to,
		t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// GetService retrieves detailed information about a specific service.
func (c *Client) GetService(ctx context.Context, serviceUid string, runDate time.Time) (*ServiceDetailResponse, error) {
	url := fmt.Sprintf("%s/json/service/%s/%04d/%02d/%02d",
		baseURL, serviceUid,
		runDate.Year(), int(runDate.Month()), runDate.Day())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result ServiceDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}
