package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCSP(t *testing.T) {
	csp := BuildCSP("test-nonce")

	assert.Contains(t, csp, "form-action 'self';")
	assert.Contains(t, csp, "script-src 'self' 'nonce-test-nonce'")
}

func TestBuildFormPostCSP(t *testing.T) {
	csp := BuildFormPostCSP("test-nonce", "https://client.example.com/callback", "'sha256-abc123'")

	// The client's redirect URI must be an allowed POST target
	assert.Contains(t, csp, "form-action 'self' https://client.example.com/callback;")

	// The single auto-submit script is allow-listed by hash alongside self and the nonce
	assert.Contains(t, csp, "script-src 'self' 'nonce-test-nonce' 'sha256-abc123'")

	// Inline scripts in general must stay forbidden in script-src
	_, scriptSrc, found := strings.Cut(csp, "script-src")
	assert.True(t, found, "csp must contain a script-src directive")
	assert.NotContains(t, scriptSrc, "unsafe-inline")
}
