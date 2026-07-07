package oidc

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeAPIAccess implements APIAccessProvider from an audience -> subject type -> allowed-scopes map.
// An audience present in the map exists as an API even when a subject type has no grants.
type fakeAPIAccess struct {
	allowed map[string]map[SubjectType][]string
}

// userAccess builds a fakeAPIAccess with only user-delegated grants, for tests that don't care about the split.
func userAccess(allowed map[string][]string) fakeAPIAccess {
	f := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{}}
	for audience, scopes := range allowed {
		f.allowed[audience] = map[SubjectType][]string{SubjectTypeUser: scopes}
	}
	return f
}

func (f fakeAPIAccess) ClientAPIScopes(_ context.Context, _ *gorm.DB, _ string) ([]string, []string, error) {
	seen := map[string]struct{}{}
	var scopes, audiences []string
	for audience, bySubject := range f.allowed {
		audiences = append(audiences, audience)
		for _, scopeKeys := range bySubject {
			for _, key := range scopeKeys {
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					scopes = append(scopes, key)
				}
			}
		}
	}
	return scopes, audiences, nil
}

func (f fakeAPIAccess) AllowedScopesForAudience(_ context.Context, _ *gorm.DB, _ string, audience string, subjectType SubjectType) ([]string, bool, error) {
	bySubject, exists := f.allowed[audience]
	if !exists {
		return nil, false, nil
	}
	return bySubject[subjectType], true, nil
}

func (f fakeAPIAccess) DescribePermissions(_ context.Context, audience string, keys []string) ([]PermissionInfo, error) {
	bySubject, ok := f.allowed[audience]
	if !ok {
		return nil, nil
	}
	var infos []PermissionInfo
	for _, key := range keys {
		for _, allowed := range bySubject {
			if slices.Contains(allowed, key) {
				infos = append(infos, PermissionInfo{Key: key, Name: key})
				break
			}
		}
	}
	return infos, nil
}

func TestResolveResourceDefaultIsLoginToken(t *testing.T) {
	// With no resource the token is a plain login token audienced to the requesting client
	audience, granted, err := resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "profile"}, SubjectTypeUser)
	require.NoError(t, err)
	assert.Equal(t, "client-1", audience)
	assert.Equal(t, []string{"openid", "profile"}, granted)
}

func TestResolveResourceRejectsCustomScopeWithoutResource(t *testing.T) {
	// Requesting a custom scope without targeting its API must be rejected, not dropped.
	_, _, err := resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "read:orders"}, SubjectTypeUser)
	require.Error(t, err)
}

func TestResolveResourceCustomAPIGrantsValidScopes(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	})

	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"openid", "read:orders"}, SubjectTypeUser)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	// openid stays (identity, for the ID token); read:orders is allowed for this API.
	assert.ElementsMatch(t, []string{"openid", "read:orders"}, granted)
}

func TestResolveResourceRejectsScopeFromAnotherAPI(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	})
	// write:billing belongs to a different API than the one targeted -> rejected.
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"openid", "write:billing"}, SubjectTypeUser)
	require.Error(t, err)
}

func TestResolveResourceUnknownIsRejected(t *testing.T) {
	provider := userAccess(map[string][]string{})
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.unknown.example.com", []string{"read"}, SubjectTypeUser)
	require.Error(t, err)
}

func TestResolveResourceUnauthorizedClientIsRejected(t *testing.T) {
	// The API exists but the client has no allowed permissions for it.
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {},
	})
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeUser)
	require.Error(t, err)
}

// TestResolveResourceSubjectTypesAreIndependent guards the core of the user/client separation:
// a permission granted for one subject type must not leak into the other, so a scope users may
// delegate cannot be minted machine-to-machine and a machine scope cannot ride along on a login.
func TestResolveResourceSubjectTypesAreIndependent(t *testing.T) {
	provider := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		"https://api.orders.example.com": {
			SubjectTypeUser:   {"read:orders"},
			SubjectTypeClient: {"write:orders"},
		},
	}}

	// The user-delegated grant works for user flows...
	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeUser)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.ElementsMatch(t, []string{"read:orders"}, granted)

	// ...but not for the client itself.
	_, _, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeClient)
	require.Error(t, err)

	// The client grant works machine-to-machine...
	audience, granted, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"write:orders"}, SubjectTypeClient)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.ElementsMatch(t, []string{"write:orders"}, granted)

	// ...but users cannot be asked to delegate it.
	_, _, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"write:orders"}, SubjectTypeUser)
	require.Error(t, err)
}

// TestResolveResourceClientWithoutClientGrantsIsDenied models a client that only has user-delegated
// grants: the API must stay unreachable through the client credentials grant entirely.
func TestResolveResourceClientWithoutClientGrantsIsDenied(t *testing.T) {
	provider := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		"https://api.orders.example.com": {
			SubjectTypeUser: {"read:orders"},
		},
	}}

	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", nil, SubjectTypeClient)
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
