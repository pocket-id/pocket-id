package oidc

import (
	"context"

	"github.com/ory/fosite"
	fositeoauth2 "github.com/ory/fosite/handler/oauth2"
)

// targetsCustomAPI reports whether the token being materialized is audienced to a custom API rather than (only) the requesting client itself
// Such a token is an API access token: it is handed to a third-party resource server and therefore must not carry Pocket ID's own identity scopes.
func targetsCustomAPI(requester fosite.Requester) bool {
	client := requester.GetClient()
	if client == nil {
		return false
	}

	clientID := client.GetID()
	for _, audience := range requester.GetGrantedAudience() {
		if audience != clientID {
			return true
		}
	}

	return false
}

// withAccessTokenScopes returns a view of the requester whose granted scopes are limited to what may
// appear on the access token
// Identity scopes stay on the underlying grant so the ID token and consent keep working, but they are stripped from an access token audienced to a custom API: those identity claims are delivered through the ID token, and keeping them off the API access token stops it from being replayed against Pocket ID's own identity endpoints such as /userinfo.
//
// The returned requester leaves every other field untouched, so callers that need the full grant (e.g. ID token issuance) are unaffected.
func withAccessTokenScopes(requester fosite.Requester) fosite.Requester {
	if !targetsCustomAPI(requester) {
		return requester
	}

	granted := requester.GetGrantedScopes()
	filtered := make(fosite.Arguments, 0, len(granted))
	for _, scope := range granted {
		if !isStandardScope(scope) {
			filtered = append(filtered, scope)
		}
	}

	return accessTokenRequester{Requester: requester, grantedScopes: filtered}
}

// accessTokenRequester overrides only the granted scopes of the wrapped requester.
type accessTokenRequester struct {
	fosite.Requester
	grantedScopes fosite.Arguments
}

func (r accessTokenRequester) GetGrantedScopes() fosite.Arguments {
	return r.grantedScopes
}

// apiAudienceAccessTokenStrategy wraps the access token strategy so the scope claim on an access token audienced to a custom API excludes identity scopes, keeping the self-contained JWT consistent with what is persisted for introspection.
type apiAudienceAccessTokenStrategy struct {
	fositeoauth2.CoreStrategy
}

func (s apiAudienceAccessTokenStrategy) GenerateAccessToken(ctx context.Context, requester fosite.Requester) (string, string, error) {
	return s.CoreStrategy.GenerateAccessToken(ctx, withAccessTokenScopes(requester))
}
