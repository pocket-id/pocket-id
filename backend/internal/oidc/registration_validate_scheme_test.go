//go:build unit

package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateRegistrationRedirectURIsRejectsActiveContentSchemes guards against
// registering a redirect URI that executes script or carries inline content.
// Dynamic registration is driven by untrusted callers, so this must hold even
// when an administrator configures a scheme wildcard in the allowlist.
func TestValidateRegistrationRedirectURIsRejectsActiveContentSchemes(t *testing.T) {
	// The permissive allowlists that previously admitted these URIs
	for _, allowlist := range [][]string{
		{"*"},
		{"*://app.example.com/**"},
	} {
		for _, uri := range []string{
			"javascript:alert(1)",
			"JavaScript:alert(1)", // scheme comparison is case-insensitive
			"data:text/html,<script>alert(1)</script>",
			"data://app.example.com/text/html,x",
			"javascript://app.example.com/%0Aalert(1)",
			"vbscript:msgbox(1)",
			"file:///etc/passwd",
			"blob:https://app.example.com/uuid",
			"about:blank",
		} {
			err := ValidateRegistrationRedirectURIs([]string{uri}, allowlist)
			assert.Error(t, err, "allowlist %v must not admit %q", allowlist, uri)
		}
	}
}

// A relative URI is not a usable redirect target and must be rejected outright.
func TestValidateRegistrationRedirectURIsRejectsRelativeURIs(t *testing.T) {
	require.Error(t, ValidateRegistrationRedirectURIs([]string{"/relative/callback"}, []string{"*"}))
	require.Error(t, ValidateRegistrationRedirectURIs([]string{"//app.example.com/cb"}, []string{"*"}))
}

// Legitimate redirect URIs must keep working, including the private-use custom
// schemes native apps rely on (RFC 8252) and http loopback URIs.
func TestValidateRegistrationRedirectURIsAllowsLegitimateSchemes(t *testing.T) {
	require.NoError(t, ValidateRegistrationRedirectURIs(
		[]string{"https://app.example.com/cb"},
		[]string{"https://app.example.com/**"},
	))
	require.NoError(t, ValidateRegistrationRedirectURIs(
		[]string{"com.example.app:/oauth2redirect"},
		[]string{"*"},
	))
	require.NoError(t, ValidateRegistrationRedirectURIs(
		[]string{"http://127.0.0.1:8080/cb"},
		[]string{"http://127.0.0.1:*/**"},
	))
}
