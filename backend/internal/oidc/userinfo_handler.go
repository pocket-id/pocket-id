package oidc

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type userInfoHandler struct {
	provider      fosite.OAuth2Provider
	claimsService *ClaimsService
}

func newUserInfoHandler(provider fosite.OAuth2Provider, claimsService *ClaimsService) *userInfoHandler {
	return &userInfoHandler{
		provider:      provider,
		claimsService: claimsService,
	}
}

// userInfo godoc
// @Summary Get user information
// @Description Get user information based on the access token
// @Tags OIDC
// @Accept json
// @Produce json
// @Success 200 {object} object "User claims based on requested scopes"
// @Security OAuth2AccessToken
// @Router /api/oidc/userinfo [get]
func (h *userInfoHandler) userInfo(c *gin.Context) {
	ctx := c.Request.Context()
	token := fosite.AccessTokenFromRequest(c.Request)
	if token == "" {
		_ = c.Error(&common.MissingAccessToken{})
		return
	}

	tokenUse, accessRequest, err := h.provider.IntrospectToken(ctx, token, fosite.AccessToken, NewEmptySession())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to introspect token", "error", err)
		_ = c.Error(err)
		return
	}
	if tokenUse != fosite.AccessToken {
		_ = c.Error(&common.TokenInvalidError{})
		return
	}

	session, ok := accessRequest.GetSession().(*Session)
	if !ok || session.GetSubject() == "" {
		_ = c.Error(&common.TokenInvalidError{})
		return
	}

	claims, err := h.claimsService.GetUserClaims(ctx, session.GetSubject(), accessRequest.GetGrantedScopes())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, claims)
}
