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

func TestWithIdentityAudienceAddsIssuerForIdentityScopes(t *testing.T) {
	const issuer = "https://issuer.example.com"

	// A token granted an identity scope alongside a custom API also lists the issuer so it can be presented to /userinfo
	apiRequest := newScopeTestRequester("client-1",
		fosite.Arguments{"openid", "read:orders"},
		fosite.Arguments{"https://api.orders.example.com"},
	)
	assert.ElementsMatch(t,
		fosite.Arguments{"https://api.orders.example.com", issuer},
		withIdentityAudience(apiRequest, issuer).GetGrantedAudience(),
	)

	// A plain login token gains the issuer audience alongside the client it was bound to
	loginRequest := newScopeTestRequester("client-1",
		fosite.Arguments{"openid", "profile", "email"},
		fosite.Arguments{"client-1"},
	)
	assert.ElementsMatch(t,
		fosite.Arguments{"client-1", issuer},
		withIdentityAudience(loginRequest, issuer).GetGrantedAudience(),
	)
}

func TestWithIdentityAudienceLeavesNonIdentityTokensUntouched(t *testing.T) {
	const issuer = "https://issuer.example.com"

	// A token audienced only to a custom API with no identity scope never gains the issuer, so it cannot reach /userinfo
	apiOnly := newScopeTestRequester("client-1",
		fosite.Arguments{"read:orders"},
		fosite.Arguments{"https://api.orders.example.com"},
	)
	assert.Equal(t, fosite.Arguments{"https://api.orders.example.com"}, withIdentityAudience(apiOnly, issuer).GetGrantedAudience())

	// offline_access alone is not an identity scope, so it does not add the issuer either
	offlineOnly := newScopeTestRequester("client-1",
		fosite.Arguments{"offline_access", "read:orders"},
		fosite.Arguments{"https://api.orders.example.com"},
	)
	assert.Equal(t, fosite.Arguments{"https://api.orders.example.com"}, withIdentityAudience(offlineOnly, issuer).GetGrantedAudience())

	// An empty issuer disables the behavior entirely, leaving the audience untouched
	assert.Equal(t, fosite.Arguments{"client-1"}, withIdentityAudience(newScopeTestRequester("client-1", fosite.Arguments{"openid"}, fosite.Arguments{"client-1"}), "").GetGrantedAudience())
}
