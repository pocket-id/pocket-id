package authz

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestSecurity(t *testing.T) {
	// A scoped route lists only the credentials that can ever hold its scope
	require.Equal(t, openapi3.SecurityRequirements{
		{"apiKey": {"users:read"}},
		{"oauth2": {"users:read"}},
		{"oauth2ClientCredentials": {"users:read"}},
		{"session": {"users:read"}},
	}, security(UsersRead, false))
	require.Equal(t, openapi3.SecurityRequirements{{"session": {"account:api-keys:create"}}}, security(AccountAPIKeysCreate, false))

	// An authenticated route accepts every credential without a scope, and an optional route accepts anonymous requests too
	require.Equal(t, openapi3.SecurityRequirements{
		{"apiKey": {}},
		{"oauth2": {}},
		{"oauth2ClientCredentials": {}},
		{"session": {}},
		{},
	}, security(Authenticated, true))

	require.Nil(t, security(Public, false))
}

func TestSecuritySchemesCoverEveryKind(t *testing.T) {
	// Every scheme that a requirement references must be defined, and every kind of credential must be documented by some scheme
	schemes := SecuritySchemes()
	var kinds PrincipalKind
	for _, scheme := range securitySchemes {
		require.Contains(t, schemes, scheme.name)
		kinds |= scheme.kinds
	}
	require.Equal(t, KindSession|KindAPIKey|KindOAuthUser|KindOAuthClient|KindStaticAPIKey, kinds)
}
