package oidc

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeAPIAccess implements APIAccessProvider from an audience -> allowed-scopes map.
type fakeAPIAccess struct {
	allowed map[string][]string
}

func (f fakeAPIAccess) ClientAPIScopes(_ context.Context, _ *gorm.DB, _ string) ([]string, []string, error) {
	seen := map[string]struct{}{}
	var scopes, audiences []string
	for audience, scopeKeys := range f.allowed {
		audiences = append(audiences, audience)
		for _, key := range scopeKeys {
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				scopes = append(scopes, key)
			}
		}
	}
	return scopes, audiences, nil
}

func (f fakeAPIAccess) AllowedScopesForAudience(_ context.Context, _ string, audience string) ([]string, bool, error) {
	scopes, exists := f.allowed[audience]
	return scopes, exists, nil
}

func (f fakeAPIAccess) DescribePermissions(_ context.Context, audience string, keys []string) ([]PermissionInfo, error) {
	allowed, ok := f.allowed[audience]
	if !ok {
		return nil, nil
	}
	var infos []PermissionInfo
	for _, key := range keys {
		if slices.Contains(allowed, key) {
			infos = append(infos, PermissionInfo{Key: key, Name: key})
		}
	}
	return infos, nil
}

func TestResolveResourceDefaultIsLoginToken(t *testing.T) {
	// With no resource the token is a plain login token audienced to the requesting client
	audience, granted, err := resolveResource(t.Context(), nil, "client-1", "", []string{"openid", "profile"})
	require.NoError(t, err)
	assert.Equal(t, "client-1", audience)
	assert.Equal(t, []string{"openid", "profile"}, granted)
}

func TestResolveResourceRejectsCustomScopeWithoutResource(t *testing.T) {
	// Requesting a custom scope without targeting its API must be rejected, not dropped.
	_, _, err := resolveResource(t.Context(), nil, "client-1", "", []string{"openid", "read:orders"})
	require.Error(t, err)
}

func TestResolveResourceCustomAPIGrantsValidScopes(t *testing.T) {
	provider := fakeAPIAccess{allowed: map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	}}

	audience, granted, err := resolveResource(t.Context(), provider, "client-1", "https://api.orders.example.com", []string{"openid", "read:orders"})
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	// openid stays (identity, for the ID token); read:orders is allowed for this API.
	assert.ElementsMatch(t, []string{"openid", "read:orders"}, granted)
}

func TestResolveResourceRejectsScopeFromAnotherAPI(t *testing.T) {
	provider := fakeAPIAccess{allowed: map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	}}
	// write:billing belongs to a different API than the one targeted -> rejected.
	_, _, err := resolveResource(t.Context(), provider, "client-1", "https://api.orders.example.com", []string{"openid", "write:billing"})
	require.Error(t, err)
}

func TestResolveResourceUnknownIsRejected(t *testing.T) {
	provider := fakeAPIAccess{allowed: map[string][]string{}}
	_, _, err := resolveResource(t.Context(), provider, "client-1", "https://api.unknown.example.com", []string{"read"})
	require.Error(t, err)
}

func TestResolveResourceUnauthorizedClientIsRejected(t *testing.T) {
	// The API exists but the client has no allowed permissions for it.
	provider := fakeAPIAccess{allowed: map[string][]string{
		"https://api.orders.example.com": {},
	}}
	_, _, err := resolveResource(t.Context(), provider, "client-1", "https://api.orders.example.com", []string{"read:orders"})
	require.Error(t, err)
}

func TestConsentScopeKeyQualifiesCustomScopesOnly(t *testing.T) {
	// Identity scopes stay bare for backward compatibility with existing consents.
	assert.Equal(t, "profile", consentScopeKey("https://api.orders.example.com", "profile"))
	// Custom scopes are qualified by audience so the same key on two APIs is distinct.
	assert.Equal(t, "https://api.orders.example.com\x1fread", consentScopeKey("https://api.orders.example.com", "read"))
	assert.NotEqual(t,
		consentScopeKey("https://api.orders.example.com", "read"),
		consentScopeKey("https://api.billing.example.com", "read"),
	)
}
