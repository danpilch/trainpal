package tfl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const northernLineStatusURL = "https://api.tfl.gov.uk/Line/northern/Status"

// Client is a TfL API client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new TfL client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetNorthernLineStatus retrieves the current status of the Northern Line.
func (c *Client) GetNorthernLineStatus(ctx context.Context) (*LineStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, northernLineStatusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "trainpal/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result []LineStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no status returned for Northern Line")
	}

	return &result[0], nil
}
