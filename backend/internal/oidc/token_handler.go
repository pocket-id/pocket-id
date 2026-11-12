package oidc

import (
	"context"
	"log/slog"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/fosite"
)

type tokenHandler struct {
	provider      fosite.OAuth2Provider
	claimsService *ClaimsService
	apiAccess     APIAccessProvider
	sessions      Sessions
}

func newTokenHandler(provider fosite.OAuth2Provider, claimsService *ClaimsService, apiAccess APIAccessProvider, sessions Sessions) *tokenHandler {
	return &tokenHandler{
		provider:      provider,
		claimsService: claimsService,
		apiAccess:     apiAccess,
		sessions:      sessions,
	}
}

func (h *tokenHandler) token(c *gin.Context) {
	ctx := c.Request.Context()

	// For grants that continue an existing session (authorization code, refresh token),
	// fosite restores the stored session over this empty one.
	session := NewEmptySession()

	accessRequest, err := h.provider.NewAccessRequest(ctx, c.Request, session)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create access request", "error", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
		return
	}

	requestSession, ok := accessRequest.GetSession().(*Session)
	if !ok {
		slog.ErrorContext(ctx, "Failed to handle token request: session must be *oidc.Session")
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, fosite.ErrServerError)
		return
	}

	var claimsSource *userClaimsSource
	client, ok := accessRequest.GetClient().(Client)
	if ok {
		// Re-validate the resource owner on every user-bound grant, loading at the same time everything the token claims are built from
		claimsSource, err = h.claimsService.loadGrantClaimsSource(ctx, requestSession.Subject, client)
		if err != nil {
			slog.WarnContext(ctx, "Rejected token request: user no longer allowed to access client", "error", err.Error())
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}

		err = h.validateGrantSession(ctx, requestSession, accessRequest)
		if err != nil {
			slog.WarnContext(ctx, "Rejected token request: the session the grant was issued in has ended", "error", err.Error())
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}

		err = h.validateRefreshAPIGrant(ctx, accessRequest)
		if err != nil {
			slog.WarnContext(ctx, "Rejected refresh token request: API grant is no longer allowed for the user subject", "error", err.Error())
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}

		// The client credentials grant has no authorize step so the RFC 8707 resource is resolved here to stamp the API audience and limit the granted scope to what the client is allowed for that API
		// It resolves against the client-subject grants: a permission delegated by users does not let the client act as itself
		// The other grants had their audience and scope resolved at authorize or device time and restored from storage, so they must be left untouched
		if accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeClientCredentials)) {
			resource, err := accessRequest.GetResource()
			if err != nil {
				h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
				return
			}
			audience, grantedScopes, err := resolveResource(ctx, nil, h.apiAccess, client.GetID(), resource, accessRequest.GetRequestedScopes(), SubjectTypeClient, client.customScopes)
			if err != nil {
				h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
				return
			}
			// A client credentials token has no resource owner, so it must never carry identity scopes such as openid, profile or a custom scope
			// Dropping them keeps machine tokens out of the userinfo endpoint, which is gated on the openid scope
			grantedScopes = slices.DeleteFunc(grantedScopes, func(scope string) bool {
				return isStandardScope(scope) || slices.Contains(client.customScopes, scope)
			})
			accessReq, ok := accessRequest.(*fosite.AccessRequest)
			if ok {
				accessReq.GrantedScope = grantedScopes
				accessReq.GrantedAudience = nil
			}
			grantResourceIndicator(accessRequest, audience, grantedScopes)
		}
	}

	// A grant without a resource owner has no source and carries no user claims
	if claimsSource != nil {
		// Fosite grants a code's scopes only when it builds the response, but the requested scopes already are the ones the user approved
		scopes := accessRequest.GetGrantedScopes()
		if accessRequest.GetGrantTypes().HasOneOf(string(fosite.GrantTypeAuthorizationCode), string(fosite.GrantTypeDeviceCode)) {
			scopes = accessRequest.GetRequestedScopes()
		}
		err = h.claimsService.applyTokenClaims(requestSession, scopes, claimsSource)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to apply token claims", "error", err)
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}
	}
	// The client credentials grant has no resource owner, so no subject is ever set. Assign a
	// stable synthetic subject so the issued JWT access token still carries a subclaim.
	if requestSession.Subject == "" {
		client, ok := accessRequest.GetClient().(Client)
		if ok && accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeClientCredentials)) {
			requestSession.Subject = ClientSubject(client.GetID())
		}
	}

	response, err := h.provider.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create access response", "error", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
		return
	}

	h.provider.WriteAccessResponse(ctx, c.Writer, accessRequest, response)
}

// validateGrantSession rejects a grant whose Pocket ID session has ended, unless it requested offline_access
// Ending a session revokes these grants as well, but this check also covers a failed revocation and a grant stored while its session was being ended
func (h *tokenHandler) validateGrantSession(ctx context.Context, session *Session, accessRequest fosite.AccessRequester) error {
	// Fosite grants a code's scopes only when it builds the response, but the requested scopes already are the ones the user approved
	offline := accessRequest.GetGrantedScopes().Has("offline_access") || accessRequest.GetRequestedScopes().Has("offline_access")
	if session.SessionID == "" || offline {
		return nil
	}

	active, err := h.sessions.IsActive(ctx, session.Subject, session.SessionID)
	if err != nil {
		return err
	}
	if !active {
		return fosite.ErrInvalidGrant.WithHint("The session the grant was issued in has ended.")
	}
	return nil
}

func (h *tokenHandler) validateRefreshAPIGrant(ctx context.Context, accessRequest fosite.AccessRequester) error {
	if !accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeRefreshToken)) {
		return nil
	}

	resource, err := validateGrantAPIAccess(ctx, h.apiAccess, accessRequest)
	if err != nil || resource == "" {
		return err
	}

	// A user who lost a role since consenting keeps only the scopes they still hold, so the refreshed tokens don't list the others
	held, err := h.apiAccess.HeldScopes(ctx, nil, accessRequest.GetSession().GetSubject(), resource, accessRequest.GetGrantedScopes())
	if err != nil {
		return err
	}
	if accessReq, ok := accessRequest.(*fosite.AccessRequest); ok {
		accessReq.GrantedScope = held
	}
	return nil
}
