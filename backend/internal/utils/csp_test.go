package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestBuildCSP(t *testing.T) {
	csp := BuildCSP("test-nonce")

	assert.Contains(t, csp, "form-action 'self';")
	assert.Contains(t, csp, "script-src 'self' 'nonce-test-nonce'")
	assert.Contains(t, csp, "frame-ancestors 'none';")
}

func TestBuildCSP_AllowedFrameAncestors(t *testing.T) {
	original := common.EnvConfig.AllowedFrameAncestors
	t.Cleanup(func() { common.EnvConfig.AllowedFrameAncestors = original })
	common.EnvConfig.AllowedFrameAncestors = []string{"https://dashboard.example.com", "http://localhost:8080"}

	csp := BuildCSP("test-nonce")

	assert.Contains(t, csp, "frame-ancestors https://dashboard.example.com http://localhost:8080;")
}

func TestBuildFormPostCSP(t *testing.T) {
	csp := BuildFormPostCSP("test-nonce", "https://client.example.com/callback", "'sha256-abc123'")

	// The origin of the client's redirect URI must be an allowed POST target
	assert.Contains(t, csp, "form-action 'self' https://client.example.com;")

	// The single auto-submit script is allow-listed by hash alongside self and the nonce
	assert.Contains(t, csp, "script-src 'self' 'nonce-test-nonce' 'sha256-abc123'")

	// Inline scripts in general must stay forbidden in script-src
	_, scriptSrc, found := strings.Cut(csp, "script-src")
	assert.True(t, found, "csp must contain a script-src directive")
	assert.NotContains(t, scriptSrc, "unsafe-inline")
}

// The client chooses its redirect URI, so it must only ever add one source to form-action and never a directive of its own
func TestBuildFormPostCSP_RedirectURICannotAddDirectives(t *testing.T) {
	tests := []struct {
		redirectURI string
		formAction  string
	}{
		{"https://client.example.com/callback?a; script-src 'none'; report-uri /x", "'self' https://client.example.com"},
		{"https://client.example.com/callback, report-uri /x", "'self' https://client.example.com"},
		{"https://client.example.com;report-uri/callback", "'self'"},
		{"https:callback; report-uri /x", "'self'"},
		{"http://127.0.0.1:49152/callback", "'self' http://127.0.0.1:49152"},
	}
	for _, tc := range tests {
		t.Run(tc.redirectURI, func(t *testing.T) {
			csp := BuildFormPostCSP("test-nonce", tc.redirectURI, "'sha256-abc123'")
			assert.Contains(t, csp, "form-action "+tc.formAction+";")
			assert.Equal(t, 1, strings.Count(csp, "script-src"))
			assert.NotContains(t, csp, "report-uri")
		})
	}
}
