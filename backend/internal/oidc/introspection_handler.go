package oidc

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
)

type introspectionHandler struct {
	provider      fosite.OAuth2Provider
	authenticator *federatedClientAuthenticator
	baseURL       string
}

func newIntrospectionHandler(provider fosite.OAuth2Provider, authenticator *federatedClientAuthenticator, baseURL string) *introspectionHandler {
	return &introspectionHandler{
		provider:      provider,
		authenticator: authenticator,
		baseURL:       baseURL,
	}
}

// introspectToken godoc
// @Summary Introspect OIDC tokens
// @Description Pass a token to verify if it is considered valid.
// @Tags OIDC
// @Produce json
// @Param token formData string true "The token to be introspected."
// @Success 200 {object} object "Response with the introspection result."
// @Router /api/oidc/introspect [post]
func (h *introspectionHandler) introspectToken(c *gin.Context) {
	ctx := c.Request.Context()

	if h.tryFederatedClientAssertionIntrospection(c) {
		return
	}

	response, err := h.provider.NewIntrospectionRequest(ctx, c.Request, NewEmptySession())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create introspection request", "error", err)
		h.provider.WriteIntrospectionError(ctx, c.Writer, err)
		return
	}

	// A client may only introspect its own tokens. If it was issued to another client, report it as
	// inactive instead of leaking its existence or contents.
	callerClientID, err := h.callerClientID(ctx, c)
	if err != nil || callerClientID == "" || response.GetAccessRequester().GetClient().GetID() != callerClientID {
		h.provider.WriteIntrospectionResponse(ctx, c.Writer, &fosite.IntrospectionResponse{Active: false})
		return
	}

	h.provider.WriteIntrospectionResponse(ctx, c.Writer, response)
}

// tryFederatedClientAssertionIntrospection handles introspection requests authenticated
// with a federated client assertion passed as bearer token instead of client credentials.
func (h *introspectionHandler) tryFederatedClientAssertionIntrospection(c *gin.Context) bool {
	ctx := c.Request.Context()
	assertion := fosite.AccessTokenFromRequest(c.Request)
	clientID := c.PostForm("client_id")
	if assertion == "" || clientID == "" {
		return false
	}

	client, err := h.authenticator.authenticateAssertion(ctx, assertion, clientID)
	if err != nil {
		h.provider.WriteIntrospectionError(ctx, c.Writer, fosite.ErrRequestUnauthorized.WithWrap(err))
		return true
	}

	tokenUse, accessRequester, err := h.provider.IntrospectToken(ctx, c.PostForm("token"), fosite.TokenUse(c.PostForm("token_type_hint")), NewEmptySession(), strings.Fields(c.PostForm("scope"))...)
	if err != nil {
		h.provider.WriteIntrospectionError(ctx, c.Writer, fosite.ErrInactiveToken.WithWrap(err))
		return true
	}

	response := &fosite.IntrospectionResponse{
		Active:          true,
		AccessRequester: accessRequester,
		TokenUse:        tokenUse,
	}
	if tokenUse == fosite.AccessToken {
		response.AccessTokenType = fosite.BearerAccessToken
	}

	if accessRequester.GetClient().GetID() != client.GetID() {
		h.provider.WriteIntrospectionResponse(ctx, c.Writer, &fosite.IntrospectionResponse{Active: false})
		return true
	}

	h.provider.WriteIntrospectionResponse(ctx, c.Writer, response)
	return true
}

// callerClientID resolves the client that authenticated this introspection request.
func (h *introspectionHandler) callerClientID(ctx context.Context, c *gin.Context) (string, error) {
	if bearer := fosite.AccessTokenFromRequest(c.Request); bearer != "" {
		_, accessRequester, err := h.provider.IntrospectToken(ctx, bearer, fosite.AccessToken, NewEmptySession())
		if err != nil {
			return "", err
		}
		return accessRequester.GetClient().GetID(), nil
	}

	if id, _, ok := c.Request.BasicAuth(); ok {
		return url.QueryUnescape(id)
	}

	return "", nil
}
