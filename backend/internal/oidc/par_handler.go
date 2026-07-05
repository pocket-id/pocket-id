package oidc

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
)

type parHandler struct {
	provider fosite.OAuth2Provider
}

func newPARHandler(provider fosite.OAuth2Provider) *parHandler {
	return &parHandler{
		provider: provider,
	}
}

func (h *parHandler) pushedAuthorizationRequest(c *gin.Context) {
	ctx := c.Request.Context()

	ar, err := h.provider.NewPushedAuthorizeRequest(ctx, c.Request)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create pushed authorize request", "error", err)
		h.provider.WritePushedAuthorizeError(ctx, c.Writer, ar, err)
		return
	}

	response, err := h.provider.NewPushedAuthorizeResponse(ctx, ar, NewEmptySession())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create pushed authorize response", "error", err)
		h.provider.WritePushedAuthorizeError(ctx, c.Writer, ar, err)
		return
	}

	h.provider.WritePushedAuthorizeResponse(ctx, c.Writer, ar, response)
}
