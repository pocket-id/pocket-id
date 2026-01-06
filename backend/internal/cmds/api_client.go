package cmds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// APIClient is a client for interacting with the Pocket-ID API
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, apiKey string) *APIClient {
	if baseURL == "" {
		baseURL = "http://localhost:" + common.EnvConfig.Port
	}

	// Ensure baseURL doesn't end with a slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAPIKey sets or updates the API key
func (c *APIClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// RequestOptions represents options for an API request
type RequestOptions struct {
	Method    string
	Path      string
	Query     map[string]string
	Body      interface{}
	Headers   map[string]string
	Output    interface{}
	RawOutput bool
}

// Do sends an HTTP request to the API
func (c *APIClient) Do(ctx context.Context, opts RequestOptions) ([]byte, error) {
	// Build URL
	u, err := url.Parse(c.baseURL + opts.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Add query parameters
	if opts.Query != nil {
		q := u.Query()
		for key, value := range opts.Query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	// Prepare request body
	var body io.Reader
	if opts.Body != nil {
		// Check if body is already an io.Reader (for multipart/form-data, etc.)
		if reader, ok := opts.Body.(io.Reader); ok {
			body = reader
		} else {
			// Otherwise, marshal as JSON
			jsonData, err := json.Marshal(opts.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewReader(jsonData)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, opts.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Accept", "application/json")

	// Only set Content-Type to application/json if body is not nil and no custom Content-Type is provided
	if opts.Body != nil {
		// Check if a custom Content-Type header will be set
		hasCustomContentType := false
		if opts.Headers != nil {
			for key := range opts.Headers {
				if strings.EqualFold(key, "Content-Type") {
					hasCustomContentType = true
					break
				}
			}
		}

		// Only set default JSON Content-Type if no custom one is provided
		if !hasCustomContentType {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Set API key if provided
	if c.apiKey != "" {
		req.Header.Set("X-API-KEY", c.apiKey)
	}

	// Set custom headers
	if opts.Headers != nil {
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse as JSON error
		var errorResponse struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil && errorResponse.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResponse.Message)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response if output struct is provided
	if opts.Output != nil && !opts.RawOutput {
		if err := json.Unmarshal(respBody, opts.Output); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return respBody, nil
}

// Get performs a GET request
func (c *APIClient) Get(ctx context.Context, path string, output interface{}, query map[string]string) error {
	_, err := c.Do(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
		Output: output,
	})
	return err
}

// Post performs a POST request
func (c *APIClient) Post(ctx context.Context, path string, body, output interface{}) error {
	_, err := c.Do(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
		Output: output,
	})
	return err
}

// Put performs a PUT request
func (c *APIClient) Put(ctx context.Context, path string, body, output interface{}) error {
	_, err := c.Do(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
		Output: output,
	})
	return err
}

// Delete performs a DELETE request
func (c *APIClient) Delete(ctx context.Context, path string) error {
	_, err := c.Do(ctx, RequestOptions{
		Method: http.MethodDelete,
		Path:   path,
	})
	return err
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalPages int `json:"totalPages"`
		TotalItems int `json:"totalItems"`
	} `json:"pagination"`
}

// ListRequestOptions represents options for listing resources
type ListRequestOptions struct {
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
	Search    string
}

// ToQuery converts ListRequestOptions to query parameters
func (o ListRequestOptions) ToQuery() map[string]string {
	query := make(map[string]string)

	if o.Page > 0 {
		query["pagination[page]"] = fmt.Sprintf("%d", o.Page)
	}
	if o.Limit > 0 {
		query["pagination[limit]"] = fmt.Sprintf("%d", o.Limit)
	}
	if o.SortBy != "" {
		query["sort[column]"] = o.SortBy
	}
	if o.SortOrder != "" {
		query["sort[direction]"] = o.SortOrder
	}
	if o.Search != "" {
		query["search"] = o.Search
	}

	return query
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}
