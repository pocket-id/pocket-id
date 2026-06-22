package oidc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
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
	tokenType, accessRequest, err := h.provider.IntrospectToken(ctx, fosite.AccessTokenFromRequest(c.Request), fosite.AccessToken, NewEmptySession())
	if err != nil {
		writeUserInfoError(c, err)
		return
	}
	if tokenType != fosite.AccessToken {
		writeUserInfoError(c, fosite.ErrRequestUnauthorized.WithDescription("Only access tokens are allowed in the authorization header."))
		return
	}

	session, ok := accessRequest.GetSession().(*Session)
	if !ok || session.GetSubject() == "" {
		writeUserInfoError(c, fosite.ErrRequestUnauthorized.WithDescription("The access token is invalid"))
		return
	}

	claims, err := h.claimsService.GetUserClaims(ctx, session.GetSubject(), accessRequest.GetGrantedScopes())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, claims)
}

func writeUserInfoError(c *gin.Context, err error) {
	rfcErr := fosite.ErrorToRFC6749Error(err)
	if rfcErr.StatusCode() == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", fmt.Sprintf(`Bearer error="%s", error_description="%s"`, rfcErr.ErrorField, rfcErr.GetDescription()))
	}

	c.JSON(rfcErr.StatusCode(), rfcErr)
}
