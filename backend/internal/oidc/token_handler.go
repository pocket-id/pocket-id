package oidc

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
)

type tokenHandler struct {
	provider      fosite.OAuth2Provider
	claimsService *ClaimsService
}

func newTokenHandler(provider fosite.OAuth2Provider, claimsService *ClaimsService) *tokenHandler {
	return &tokenHandler{
		provider:      provider,
		claimsService: claimsService,
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
		if err := h.claimsService.ValidateUserAccess(ctx, requestSession.Subject, client); err != nil {
			slog.WarnContext(ctx, "Rejected token request: user no longer allowed to access client", "error", err.Error())
			h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
			return
		}

		// Bind every issued JWT access token to the requesting client so it always carries an aud claim.
		accessRequest.GrantAudience(client.GetID())
	}

	if err := h.claimsService.applyIDTokenClaims(ctx, requestSession, accessRequest.GetGrantedScopes()); err != nil {
		slog.ErrorContext(ctx, "Failed to apply ID token claims", "error", err)
		h.provider.WriteAccessError(ctx, c.Writer, accessRequest, err)
		return
	}

	// The client credentials grant has no resource owner, so no subject is ever set. Assign a
	// stable synthetic subject so the issued JWT access token still carries a subclaim.
	if requestSession.Subject == "" {
		if client, ok := accessRequest.GetClient().(Client); ok && accessRequest.GetGrantTypes().Has(string(fosite.GrantTypeClientCredentials)) {
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
