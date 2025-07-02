// This file is only imported by unit tests

package testing

import (
	"io"
	"net/http"
	"strings"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MockRoundTripper is a custom http.RoundTripper that returns responses based on the URL
type MockRoundTripper struct {
	Err       error
	Responses map[string]*http.Response
}

// RoundTrip implements the http.RoundTripper interface
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if we have a specific response for this URL
	for url, resp := range m.Responses {
		if req.URL.String() == url {
			return resp, nil
		}
	}

	return NewMockResponse(http.StatusNotFound, ""), nil
}

// NewMockResponse creates an http.Response with the given status code and body
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
