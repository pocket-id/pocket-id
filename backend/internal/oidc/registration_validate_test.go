package oidc

import (
	"testing"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func TestValidateRegistrationRedirectURIs(t *testing.T) {
	allow := []string{"https://app.example.com/**"}

	require.NoError(t, ValidateRegistrationRedirectURIs([]string{"https://app.example.com/cb"}, allow))

	// No redirect URIs is invalid.
	require.Error(t, ValidateRegistrationRedirectURIs(nil, allow))

	// A URI outside the allowlist is rejected.
	require.Error(t, ValidateRegistrationRedirectURIs([]string{"https://evil.example.com/cb"}, allow))

	// An empty allowlist denies everything (fail-closed).
	require.Error(t, ValidateRegistrationRedirectURIs([]string{"https://app.example.com/cb"}, nil))
}

// TestDynamicClientGrantTypesAreRedirectBased locks in the grant policy for
// dynamically registered clients.
//
// Registration is unauthenticated and the administrator's redirect URI allowlist is the
// only thing constraining it. That allowlist can only secure flows that actually redirect,
// so a dynamic client must not receive the device-code grant (which has no redirect URI
// and could otherwise be driven by someone who registered under an allowlisted pattern
// they do not control) or client credentials (which would let self-registration mint
// machine tokens). This must also match the grant_types advertised in the RFC 7591
// registration response.
func TestDynamicClientGrantTypesAreRedirectBased(t *testing.T) {
	redirectBased := []string{"authorization_code", "refresh_token"}

	for _, tc := range []struct {
		name     string
		isPublic bool
	}{
		{name: "public", isPublic: true},
		{name: "confidential", isPublic: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := Client{OidcClient: model.OidcClient{
				ClientType: model.OidcClientTypeDynamic,
				IsPublic:   tc.isPublic,
			}}

			got := []string(client.GetGrantTypes())
			assert.ElementsMatch(t, redirectBased, got,
				"a dynamic client must only receive the redirect-based grants")
			assert.NotContains(t, got, string(fosite.GrantTypeDeviceCode),
				"the device flow has no redirect URI, so the allowlist cannot constrain it")
			assert.NotContains(t, got, string(fosite.GrantTypeClientCredentials),
				"self-registration must not be able to mint machine tokens")
		})
	}
}
