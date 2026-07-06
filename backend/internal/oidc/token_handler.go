package oidc

import (
	"log/slog"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
)

type tokenHandler struct {
	provider      fosite.OAuth2Provider
	claimsService *ClaimsService
	apiAccess     APIAccessProvider
}

func newTokenHandler(provider fosite.OAuth2Provider, claimsService *ClaimsService, apiAccess APIAccessProvider) *tokenHandler {
	return &tokenHandler{
		provider:      provider,
		claimsService: claimsService,
		apiAccess:     apiAccess,
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

	if client, ok := accessRequest.GetClient().(Client); ok {
		// Re-validate the resource owner on every user-bound grant.
		err := h.claimsService.ValidateUserAccess(ctx, requestSession.Subject, client)
		if err != nil {
			slog.WarnContext(ctx, "Rejected token request: user no longer allowed to access client", "error", err.Error())
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}

		// The client credentials grant has no authorize step so the RFC 8707 resource is resolved here to stamp the API audience and limit the granted scope to what the client is allowed for that API
		// It resolves against the client-subject grants: a permission delegated by users does not let the client act as itself
		// The other grants had their audience and scope resolved at authorize or device time and restored from storage, so they must be left untouched
		if accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeClientCredentials)) {
			resource := accessRequest.GetRequestForm().Get("resource")
			audience, grantedScopes, err := resolveResource(ctx, h.apiAccess, client.GetID(), resource, accessRequest.GetRequestedScopes(), SubjectTypeClient)
			if err != nil {
				h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
				return
			}
			// A client credentials token has no resource owner, so it must never carry identity scopes such as openid or profile
			// Dropping them keeps machine tokens out of the userinfo endpoint, which is gated on the openid scope
			grantedScopes = slices.DeleteFunc(grantedScopes, isStandardScope)
			accessReq, ok := accessRequest.(*fosite.AccessRequest)
			if ok {
				accessReq.GrantedScope = fosite.Arguments(grantedScopes)
				accessReq.GrantedAudience = nil
			}
			fosite.GrantResourceIndicator(accessRequest, fosite.ResourceIndicatorGrant{
				Audience: audience,
				Scopes:   fosite.Arguments(grantedScopes),
			})
		}
	}

	err = h.claimsService.applyIDTokenClaims(ctx, requestSession, accessRequest.GetGrantedScopes())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to apply ID token claims", "error", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
		return
	}

	// The client credentials grant has no resource owner, so no subject is ever set. Assign a
	// stable synthetic subject so the issued JWT access token still carries a subclaim.
	if requestSession.Subject == "" {
		client, ok := accessRequest.GetClient().(Client)
		if ok && accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeClientCredentials)) {
			requestSession.Subject = "client-" + client.GetID()
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
