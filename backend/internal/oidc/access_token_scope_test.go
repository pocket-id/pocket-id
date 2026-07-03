package oidc

import (
	"testing"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func newScopeTestRequester(clientID string, granted fosite.Arguments, audience fosite.Arguments) *fosite.Request {
	request := fosite.NewRequest()
	request.Client = Client{
		OidcClient: model.OidcClient{
			Base: model.Base{ID: clientID},
		},
	}
	request.GrantedScope = granted
	request.GrantedAudience = audience
	return request
}

func TestTargetsCustomAPI(t *testing.T) {
	// A login token is audienced only to the requesting client
	assert.False(t, targetsCustomAPI(newScopeTestRequester("client-1", nil, fosite.Arguments{"client-1"})))
	// No audience at all is not a custom API
	assert.False(t, targetsCustomAPI(newScopeTestRequester("client-1", nil, nil)))
	// Any audience other than the client itself means the token targets a custom API
	assert.True(t, targetsCustomAPI(newScopeTestRequester("client-1", nil, fosite.Arguments{"https://api.example.com"})))
	assert.True(t, targetsCustomAPI(newScopeTestRequester("client-1", nil, fosite.Arguments{"client-1", "https://api.example.com"})))
}

func TestWithAccessTokenScopesStripsIdentityScopesForCustomAPI(t *testing.T) {
	// An access token audienced to a custom API keeps only the API scopes
	// Because identity scopes are dropped, the token cannot be used at /userinfo
	apiRequest := newScopeTestRequester("client-1",
		fosite.Arguments{"openid", "profile", "email", "groups", "offline_access", "read:orders"},
		fosite.Arguments{"https://api.orders.example.com"},
	)
	assert.Equal(t, fosite.Arguments{"read:orders"}, withAccessTokenScopes(apiRequest).GetGrantedScopes())

	// A login token audienced to the client keeps all its scopes untouched.
	loginRequest := newScopeTestRequester("client-1",
		fosite.Arguments{"openid", "profile", "email"},
		fosite.Arguments{"client-1"},
	)
	assert.Equal(t, fosite.Arguments{"openid", "profile", "email"}, withAccessTokenScopes(loginRequest).GetGrantedScopes())
}
