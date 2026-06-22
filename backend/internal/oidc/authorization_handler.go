package oidc

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const parRequestURIPrefix = "urn:ietf:params:oauth:request_uri:"

type authorizationHandler struct {
	provider             fosite.OAuth2Provider
	authorizationService *authorizationService
	baseURL              string
}

func newAuthorizationHandler(
	provider fosite.OAuth2Provider,
	authorizationService *authorizationService,
	baseURL string,
) *authorizationHandler {
	return &authorizationHandler{
		provider:             provider,
		authorizationService: authorizationService,
		baseURL:              baseURL,
	}
}

func (h *authorizationHandler) authorize(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString("userID")
	authenticationMethod := c.GetString("authenticationMethod")
	authenticationTime, _ := c.Get("authenticationTime")
	typedAuthenticationTime, _ := authenticationTime.(time.Time)
	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)

	// A request that resumes an interaction only carries the interaction ID; the original
	// parameters are restored from the stored session so they never travel through the
	// front channel.
	interactionID := c.Query("interaction")
	if interactionID != "" {
		query, err := h.authorizationService.interactionRequestQuery(ctx, interactionID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to restore authorize request from interaction session", "error", err.Error())
			h.provider.WriteAuthorizeError(ctx, c.Writer, fosite.NewAuthorizeRequest(), err)
			return
		}
		c.Request.URL.RawQuery = query.Encode()
	}

	// Treat the request as a pushed authorization request only when the request_uri carries the
	// PAR prefix. Without this, a client required to use PAR could bypass that requirement by
	// sending an arbitrary (non-prefixed) request_uri, which fosite silently ignores.
	hasPushedAuthorizationRequest := strings.HasPrefix(c.Query("request_uri"), parRequestURIPrefix)

	ar, err := h.provider.NewAuthorizeRequest(ctx, c.Request)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create authorize request", "error", err.Error())
		h.provider.WriteAuthorizeError(ctx, c.Writer, ar, err)
		return
	}

	authorization, err := h.authorizationService.authorize(ctx, authorizeInput{
		userID:                        userID,
		authenticationMethod:          authenticationMethod,
		authenticationTime:            typedAuthenticationTime,
		requester:                     ar,
		hasPushedAuthorizationRequest: hasPushedAuthorizationRequest,
		reauthenticationToken:         reauthenticationToken,
		interactionID:                 interactionID,
		requestParams:                 authorizeRequestParams(ar),
		meta:                          requestMetaFromGin(c),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to authorize request", "error", err.Error())
		h.provider.WriteAuthorizeError(ctx, c.Writer, ar, err)
		return
	}

	if authorization.RequiresInteraction {
		c.Redirect(http.StatusFound, "/interaction?interaction="+authorization.InteractionID)
		return
	}

	response, err := h.provider.NewAuthorizeResponse(ctx, ar, authorization.Session)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create authorize response", "error", err.Error())
		h.provider.WriteAuthorizeError(ctx, c.Writer, ar, err)
		return
	}

	response.AddParameter("iss", h.baseURL)
	if ar.GetResponseMode() == fosite.ResponseModeFormPost && ar.GetRedirectURI() != nil {
		c.Header("Content-Security-Policy", utils.BuildCSP(utils.GetCSPNonce(c), ar.GetRedirectURI().String()))
	}
	h.provider.WriteAuthorizeResponse(ctx, c.Writer, ar, response)
}

func requestMetaFromGin(c *gin.Context) requestMeta {
	return requestMeta{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

func authorizeRequestParams(requester fosite.AuthorizeRequester) map[string]string {
	params := make(map[string]string)
	for key, values := range requester.GetRequestForm() {
		if len(values) == 0 || key == "request_uri" || key == "interaction" {
			continue
		}
		params[key] = values[0]
	}

	return params
}

func (h *authorizationHandler) getInteractionSession(c *gin.Context) {
	interactionID := c.Param("id")

	interactionSession, err := h.authorizationService.getInteractionSession(c.Request.Context(), interactionID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, interactionSession)
}

func (h *authorizationHandler) completeInteraction(c *gin.Context) {
	interactionID := c.Param("id")
	authenticationTime, _ := c.Get("authenticationTime")
	typedAuthenticationTime, _ := authenticationTime.(time.Time)

	var request completeInteractionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(&common.ValidationError{Message: "invalid interaction request"})
		return
	}

	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)
	response, err := h.authorizationService.completeInteractionStep(c.Request.Context(), interactionID, c.GetString("userID"), request.Step, reauthenticationToken, typedAuthenticationTime, requestMetaFromGin(c))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}
