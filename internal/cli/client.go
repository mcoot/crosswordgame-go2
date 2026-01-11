package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for the API
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken updates the client's token
func (c *Client) SetToken(token string) {
	c.token = token
}

// APIError represents an error response from the API
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps an API error
type ErrorResponse struct {
	Error APIError `json:"error"`
}

func (e *APIError) String() string {
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// Do performs an HTTP request
func (c *Client) Do(method, path string, body, result any) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error responses
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Code != "" {
			return fmt.Errorf("%s", errResp.Error.String())
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request
func (c *Client) Get(path string, result any) error {
	return c.Do(http.MethodGet, path, nil, result)
}

// Post performs a POST request
func (c *Client) Post(path string, body, result any) error {
	return c.Do(http.MethodPost, path, body, result)
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body, result any) error {
	return c.Do(http.MethodPatch, path, body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) error {
	return c.Do(http.MethodDelete, path, nil, nil)
}
