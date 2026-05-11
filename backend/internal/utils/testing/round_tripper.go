// This file is only imported by unit tests

package testing

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MockRoundTripper is a custom http.RoundTripper that returns responses based on the URL
type MockRoundTripper struct {
	Err       error
	Responses map[string]*http.Response

	mu             sync.Mutex
	responseBodies map[string][]byte
}

// RoundTrip implements the http.RoundTripper interface
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	// Check if we have a specific response for this URL
	for url, resp := range m.Responses {
		if req.URL.String() == url {
			return m.cloneResponse(url, resp)
		}
	}

	return NewMockResponse(http.StatusNotFound, ""), nil
}

func (m *MockRoundTripper) cloneResponse(url string, resp *http.Response) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.responseBodies == nil {
		m.responseBodies = make(map[string][]byte, len(m.Responses))
	}

	body, ok := m.responseBodies[url]
	if !ok {
		var err error
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		m.responseBodies[url] = body
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	cloned := new(http.Response)
	*cloned = *resp
	cloned.Header = resp.Header.Clone()
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))

	return cloned, nil
}

// NewMockResponse creates an http.Response with the given status code and body
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        make(http.Header),
		ContentLength: int64(len(body)),
	}
}
