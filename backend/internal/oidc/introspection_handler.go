package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/fosite"
)

type introspectionHandler struct {
	provider      fosite.OAuth2Provider
	authenticator *federatedClientAuthenticator
	claimsService *ClaimsService
	apiAccess     APIAccessProvider
	baseURL       string
}

func newIntrospectionHandler(provider fosite.OAuth2Provider, authenticator *federatedClientAuthenticator, claimsService *ClaimsService, apiAccess APIAccessProvider, baseURL string) *introspectionHandler {
	return &introspectionHandler{
		provider:      provider,
		authenticator: authenticator,
		claimsService: claimsService,
		apiAccess:     apiAccess,
		baseURL:       baseURL,
	}
}

// introspectionForm documents the form body of the introspection endpoint, which fosite parses itself
type introspectionForm struct {
	Token         string `json:"token" binding:"required" description:"The token to be introspected"`
	TokenTypeHint string `json:"token_type_hint,omitempty" description:"The type of the token, access_token or refresh_token"`
	Scope         string `json:"scope,omitempty" description:"Space-separated scopes the token must have"`
	ClientID      string `json:"client_id,omitempty" description:"The client whose federated client assertion is sent as the bearer token"`
}

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

	h.writeCheckedResponse(ctx, c, response)
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

	h.writeCheckedResponse(ctx, c, response)
	return true
}

// writeCheckedResponse writes the introspection response, reporting the token as inactive when its user may no longer use the client or the client lost the access the token carries
// Disabling a user, removing them from the client's groups or narrowing the client's API access does not revoke the tokens already issued, so resource servers learn about it only here
func (h *introspectionHandler) writeCheckedResponse(ctx context.Context, c *gin.Context, response fosite.IntrospectionResponder) {
	requester := response.GetAccessRequester()
	client, _ := requester.GetClient().(Client)
	_, err := h.claimsService.loadGrantClaimsSource(ctx, requester.GetSession().GetSubject(), client)
	if err == nil {
		_, err = validateGrantAPIAccess(ctx, h.apiAccess, h.baseURL, requester)
	}
	var rfcErr *fosite.RFC6749Error
	if errors.As(err, &rfcErr) {
		h.provider.WriteIntrospectionResponse(ctx, c.Writer, &fosite.IntrospectionResponse{Active: false})
		return
	} else if err != nil {
		h.provider.WriteIntrospectionError(ctx, c.Writer, err)
		return
	}

	h.provider.WriteIntrospectionResponse(ctx, c.Writer, response)
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
