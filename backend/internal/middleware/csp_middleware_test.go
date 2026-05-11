package middleware

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCSP(t *testing.T) {
	t.Run("uses self form action by default", func(t *testing.T) {
		csp := BuildCSP("test-nonce")

		assert.Contains(t, csp, "form-action 'self';")
		assert.Contains(t, csp, "script-src 'self' 'nonce-test-nonce'")
	})

	t.Run("adds validated form action targets", func(t *testing.T) {
		csp := BuildCSP("test-nonce", "https://example.com/callback")

		assert.Contains(t, csp, "form-action 'self' https://example.com/callback;")
		assert.Equal(t, 1, strings.Count(csp, "form-action"))
	})
}
