package authz

import (
	"strings"
	"testing"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"
)

func TestCatalogInvariants(t *testing.T) {
	// Scope keys end up in API key records and OAuth tokens, so they must be valid scope tokens that never collide with the identity scopes
	reserved := []string{"openid", "profile", "email", "email_verified", "groups", "offline_access"}

	seen := make(map[Scope]struct{}, len(catalog))
	for _, entry := range catalog {
		t.Run(string(entry.scope), func(t *testing.T) {
			require.True(t, fosite.IsValidScopeToken(string(entry.scope)), "scope must be a valid RFC 6749 scope token")
			require.NotContains(t, reserved, strings.ToLower(string(entry.scope)))

			_, duplicate := seen[entry.scope]
			require.False(t, duplicate, "scope is listed twice")
			seen[entry.scope] = struct{}{}

			require.Contains(t, []Category{CategoryAccount, CategoryAdmin}, entry.category)
			require.NotZero(t, entry.grantableTo, "a scope nobody can hold can never pass a route")

			// Account scopes are named after the account and admin scopes after a resource, so the prefix alone tells callers what they reach
			require.Equal(t, entry.category == CategoryAccount, strings.HasPrefix(string(entry.scope), "account:"))
		})
	}
	require.Len(t, definitions, len(catalog))
}

func TestClientCredentialsCannotHoldAnyScope(t *testing.T) {
	// Service identities are not supported yet, see the scope-authorization plan
	for _, entry := range catalog {
		require.False(t, entry.scope.GrantableTo(KindOAuthClient), entry.scope)
	}
}

func TestUserScopes(t *testing.T) {
	t.Run("admins hold every scope their credential kind allows", func(t *testing.T) {
		scopes := UserScopes(true, KindSession)
		require.Len(t, scopes, len(catalog))
	})

	t.Run("regular users hold only account scopes", func(t *testing.T) {
		scopes := UserScopes(false, KindSession)
		for _, entry := range catalog {
			require.Equal(t, entry.category == CategoryAccount, scopes.Has(entry.scope), entry.scope)
		}
	})

	t.Run("API keys never hold session-only scopes, even for admins", func(t *testing.T) {
		scopes := UserScopes(true, KindAPIKey)
		require.True(t, scopes.Has(UsersWrite))
		require.True(t, scopes.Has(AccountAPIKeys))
		require.False(t, scopes.Has(AccountSession))
		require.False(t, scopes.Has(AccountPasskeysEnroll))
		require.False(t, scopes.Has(AccountAPIKeysCreate))
	})
}

func TestUnknownScope(t *testing.T) {
	unknown := Scope("unknown:scope")
	require.False(t, unknown.Known())
	require.False(t, unknown.GrantableTo(KindSession))
	require.True(t, UsersRead.Known())
}
