package oidc

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"gorm.io/gorm"
)

type userInfoHandler struct {
	provider      fosite.OAuth2Provider
	claimsService *ClaimsService
	issuer        string
}

func newUserInfoHandler(provider fosite.OAuth2Provider, claimsService *ClaimsService, issuer string) *userInfoHandler {
	return &userInfoHandler{
		provider:      provider,
		claimsService: claimsService,
		issuer:        issuer,
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

	// userinfo is one of Pocket ID's own identity endpoints, so the presented token must be audienced to Pocket ID itself (the issuer)
	// A token granted an identity scope carries the issuer audience and is accepted here even when it also targets a custom API, while a token audienced only to a custom API belongs to that third-party resource server and cannot be replayed here to read the user's profile
	if !accessRequest.GetGrantedAudience().Has(h.issuer) {
		writeUserInfoError(c, fosite.ErrAccessDenied.WithDescription("The access token is not audienced to this server and cannot be used to access user information."))
		return
	}

	// userinfo serves OIDC identity tokens, which are exactly the ones granted the openid scope
	// An access token issued purely for a custom API never carries openid and is rejected here regardless of its audience
	if !accessRequest.GetGrantedScopes().Has("openid") {
		writeUserInfoError(c, fosite.ErrAccessDenied.WithDescription("The access token is missing the openid scope."))
		return
	}

	claims, err := h.claimsService.GetUserClaims(ctx, session.GetSubject(), accessRequest.GetGrantedScopes())
	if err != nil {
		// A token whose subject no longer resolves to a user is an authentication failure, not a missing resource
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeUserInfoError(c, fosite.ErrRequestUnauthorized.WithDescription("The access token is invalid"))
			return
		}
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
