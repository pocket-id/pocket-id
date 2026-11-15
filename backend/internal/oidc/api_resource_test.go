package oidc

import (
	"context"
	"slices"
	"testing"

	"github.com/pocket-id/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// fakeAPIAccess implements APIAccessProvider from an audience -> subject type -> allowed-scopes map.
// An audience present in the map exists as an API; a subject type present under it grants access, together with the scopes that subject may request, which can be none.
// Scopes listed in unheld are ones no user holds, standing in for Pocket ID scopes the user's role lacks.
type fakeAPIAccess struct {
	allowed map[string]map[SubjectType][]string
	unheld  []string
}

// userAccess builds a fakeAPIAccess with only user-delegated grants, for tests that don't care about the split.
func userAccess(allowed map[string][]string) fakeAPIAccess {
	f := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{}}
	for audience, scopes := range allowed {
		f.allowed[audience] = map[SubjectType][]string{SubjectTypeUser: scopes}
	}
	return f
}

func (f fakeAPIAccess) ClientAPIScopes(_ context.Context, _ *gorm.DB, _ string, _ bool) ([]string, []string, error) {
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

func (f fakeAPIAccess) AllowedScopesForAudience(_ context.Context, _ *gorm.DB, _ string, audience string, subjectType SubjectType) ([]string, bool, bool, error) {
	bySubject, exists := f.allowed[audience]
	if !exists {
		return nil, false, false, nil
	}
	scopes, hasAccess := bySubject[subjectType]
	return scopes, true, hasAccess, nil
}

func (f fakeAPIAccess) DescribePermissions(_ context.Context, audience string, keys []string) ([]dto.ScopeInfoDto, error) {
	bySubject, ok := f.allowed[audience]
	if !ok {
		return nil, nil
	}
	var infos []dto.ScopeInfoDto
	for _, key := range keys {
		for _, allowed := range bySubject {
			if slices.Contains(allowed, key) {
				infos = append(infos, dto.ScopeInfoDto{Key: key, Name: key})
				break
			}
		}
	}
	return infos, nil
}

func (f fakeAPIAccess) HeldScopes(_ context.Context, _ *gorm.DB, _, _ string, scopes []string) ([]string, error) {
	return slices.DeleteFunc(slices.Clone(scopes), func(scope string) bool {
		return slices.Contains(f.unheld, scope)
	}), nil
}

func TestResolveResourceDefaultIsLoginToken(t *testing.T) {
	// With no resource the token is a plain login token audienced to the requesting client
	audience, granted, err := resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "profile"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "client-1", audience)
	assert.Equal(t, []string{"openid", "profile"}, granted)
}

func TestResolveResourceRejectsCustomScopeWithoutResource(t *testing.T) {
	// Requesting a custom scope without targeting its API must be rejected, not dropped.
	_, _, err := resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "read:orders"}, SubjectTypeUser, nil)
	require.Error(t, err)
}

func TestResolveResourceCustomAPIGrantsValidScopes(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	})

	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"openid", "read:orders"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	// openid stays (identity, for the ID token); read:orders is allowed for this API.
	assert.ElementsMatch(t, []string{"openid", "read:orders"}, granted)
}

func TestResolveResourceGrantsAllowedCustomScopes(t *testing.T) {
	_, granted, err := resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "kuber"}, SubjectTypeUser, []string{"kuber"})
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "kuber"}, granted)

	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders"},
	})
	_, granted, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"kuber", "read:orders"}, SubjectTypeUser, []string{"kuber"})
	require.NoError(t, err)
	assert.Equal(t, []string{"kuber", "read:orders"}, granted)

	_, _, err = resolveResource(t.Context(), nil, nil, "client-1", "", []string{"openid", "kuber"}, SubjectTypeUser, nil)
	require.ErrorIs(t, err, fosite.ErrInvalidScope)
}

func TestResolveResourceTrimsTrailingSlashes(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders"},
	})

	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com///", []string{"read:orders"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.Equal(t, []string{"read:orders"}, granted)
}

func TestResolveResourceIssuesMCPTokensForPocketIDAPI(t *testing.T) {
	origAppURL := common.EnvConfig.AppURL
	t.Cleanup(func() { common.EnvConfig.AppURL = origAppURL })
	common.EnvConfig.AppURL = "https://id.example.com"

	provider := userAccess(map[string][]string{
		"https://id.example.com/api": {"users:read"},
	})

	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://id.example.com/mcp", []string{"users:read"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com/api", audience, "the API's bearer check and refresh grants only know the API's own audience")
	assert.Equal(t, []string{"users:read"}, granted)
}

func TestResolveResourceRejectsScopeFromAnotherAPI(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {"read:orders", "write:orders"},
	})
	// write:billing belongs to a different API than the one targeted -> rejected.
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"openid", "write:billing"}, SubjectTypeUser, nil)
	require.Error(t, err)
}

func TestResolveResourceUnknownIsRejected(t *testing.T) {
	provider := userAccess(map[string][]string{})
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.unknown.example.com", []string{"read"}, SubjectTypeUser, nil)
	require.Error(t, err)
}

func TestResolveResourceUnauthorizedClientIsRejected(t *testing.T) {
	// The API exists but the client was not granted access to it for any subject type.
	provider := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		"https://api.orders.example.com": {},
	}}
	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeUser, nil)
	require.Error(t, err)
}

// TestResolveResourceAccessWithoutPermissions covers a client that may reach an API without holding a single
// permission, which is what the MCP specification expects: the token is audienced to the resource and carries no scope.
func TestResolveResourceAccessWithoutPermissions(t *testing.T) {
	provider := userAccess(map[string][]string{
		"https://api.orders.example.com": {},
	})

	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", nil, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.Empty(t, granted)

	// Identity scopes still ride along, so the same request can also produce an ID token
	_, granted, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"openid"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid"}, granted)

	// Access alone does not make a custom scope requestable
	_, _, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeUser, nil)
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
	audience, granted, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeUser, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.ElementsMatch(t, []string{"read:orders"}, granted)

	// ...but not for the client itself.
	_, _, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"read:orders"}, SubjectTypeClient, nil)
	require.Error(t, err)

	// The client grant works machine-to-machine...
	audience, granted, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"write:orders"}, SubjectTypeClient, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.orders.example.com", audience)
	assert.ElementsMatch(t, []string{"write:orders"}, granted)

	// ...but users cannot be asked to delegate it.
	_, _, err = resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", []string{"write:orders"}, SubjectTypeUser, nil)
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

	_, _, err := resolveResource(t.Context(), nil, provider, "client-1", "https://api.orders.example.com", nil, SubjectTypeClient, nil)
	require.Error(t, err)
}

func TestValidateGrantAPIAccessFollowsCurrentClientGrant(t *testing.T) {
	const api = "https://api.orders.example.com"
	const issuer = "https://id.example.com"
	provider := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		api: {
			SubjectTypeUser:   {"read:orders"},
			SubjectTypeClient: {"read:orders", "write:orders"},
		},
	}}

	token := func(subject string, audience fosite.Arguments, scopes ...string) fosite.Requester {
		session := NewEmptySession()
		session.Subject = subject
		request := fosite.NewAccessRequest(session)
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: "client-1"}}}
		request.GrantedAudience = audience
		request.GrantedScope = scopes
		return request
	}
	machineWrite := token(ClientSubject("client-1"), fosite.Arguments{api}, "write:orders")
	machineRead := token(ClientSubject("client-1"), fosite.Arguments{api}, "read:orders")
	user := token("user-1", fosite.Arguments{api, issuer}, "openid", "read:orders")
	login := token("user-1", fosite.Arguments{"client-1", issuer}, "openid")

	for _, requester := range []fosite.Requester{machineWrite, machineRead, user, login} {
		_, err := validateGrantAPIAccess(t.Context(), provider, issuer, requester)
		require.NoError(t, err)
	}

	provider.allowed[api][SubjectTypeClient] = []string{"read:orders"}
	_, err := validateGrantAPIAccess(t.Context(), provider, issuer, machineWrite)
	require.ErrorIs(t, err, fosite.ErrInvalidScope)
	_, err = validateGrantAPIAccess(t.Context(), provider, issuer, machineRead)
	require.NoError(t, err)

	delete(provider.allowed[api], SubjectTypeClient)
	_, err = validateGrantAPIAccess(t.Context(), provider, issuer, machineRead)
	require.ErrorIs(t, err, fosite.ErrAccessDenied)
	_, err = validateGrantAPIAccess(t.Context(), provider, issuer, user)
	require.NoError(t, err, "the access users delegate is a separate grant")

	delete(provider.allowed[api], SubjectTypeUser)
	_, err = validateGrantAPIAccess(t.Context(), provider, issuer, user)
	require.ErrorIs(t, err, fosite.ErrAccessDenied)
	_, err = validateGrantAPIAccess(t.Context(), provider, issuer, login)
	require.NoError(t, err, "a login token targets no API")
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
