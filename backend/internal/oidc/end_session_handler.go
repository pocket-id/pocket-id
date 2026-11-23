package oidc

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type endSessionHandler struct {
	endSessionService *endSessionService
	sessions          Sessions
	baseURL           string
}

func newEndSessionHandler(endSessionService *endSessionService, sessions Sessions, baseURL string) *endSessionHandler {
	return &endSessionHandler{
		endSessionService: endSessionService,
		sessions:          sessions,
		baseURL:           baseURL,
	}
}

func (h *endSessionHandler) endSession(c *gin.Context) {
	input, err := bindEndSessionRequest(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	origin := auditlogs.OriginFrom(c)
	principal := origin.Principal

	// A cross-site POST carries no SameSite=Lax cookie, but the top-level GET it is redirected to does
	if c.Request.Method == http.MethodPost && principal.SessionID == "" {
		c.Redirect(http.StatusSeeOther, h.baseURL+"/api/oidc/end-session?"+endSessionQuery(input))
		return
	}

	req, err := h.endSessionService.resolve(c.Request.Context(), input, principal.UserID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "Error getting logout callback URL, the user has to confirm the logout manually", "error", err)
		c.Redirect(http.StatusFound, h.baseURL+"/logout")
		return
	}

	// RP-Initiated Logout 1.0 requires asking the user when the ID token doesn't belong to the browser's session
	if principal.SessionID == "" || (req.sessionID != "" && req.sessionID != principal.SessionID) {
		c.Redirect(http.StatusFound, h.baseURL+"/end-session?"+endSessionQuery(input))
		return
	}

	redirectURL := h.logOut(c, req, origin)
	if redirectURL == "" {
		c.Redirect(http.StatusFound, h.baseURL+"/logout")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

// getEndSessionRequest returns what the confirmation page shows about a logout request
func (h *endSessionHandler) getEndSessionRequest(c *gin.Context) {
	var input dto.OidcLogoutDto
	if err := c.ShouldBindQuery(&input); err != nil {
		_ = c.Error(err)
		return
	}

	req, err := h.endSessionService.resolve(c.Request.Context(), input, authz.PrincipalFrom(c).UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, endSessionRequestDto{ClientName: req.clientName})
}

// confirmEndSession performs a logout request the user confirmed on the confirmation page
// Whoever holds the ID token can call it without the page, so as in Keycloak the page guards against forged browser requests, not against the token's holder
func (h *endSessionHandler) confirmEndSession(c *gin.Context) {
	var input dto.OidcLogoutDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}
	origin := auditlogs.OriginFrom(c)

	req, err := h.endSessionService.resolve(c.Request.Context(), input, origin.Principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, endSessionConfirmationDto{RedirectURL: h.logOut(c, req, origin)})
}

// logOut ends the session the ID token was issued in and the browser's own session, and returns the client's post-logout URL
// The caller already made sure that both sessions belong to the same user
func (h *endSessionHandler) logOut(c *gin.Context, req endSessionRequest, origin auditlogs.Origin) string {
	ctx := c.Request.Context()
	principal := origin.Principal

	// Every step is best effort, because the user has to be sent back to the client either way
	err := h.endSessionService.revokeClientGrant(ctx, req)
	if err != nil {
		slog.WarnContext(ctx, "Failed to revoke the client's grant on logout", "error", err)
	}
	sessionIDs := []string{req.sessionID}
	if principal.SessionID != req.sessionID {
		sessionIDs = append(sessionIDs, principal.SessionID)
	}
	for _, sessionID := range sessionIDs {
		if sessionID == "" {
			continue
		}
		err = h.sessions.Revoke(ctx, req.userID, sessionID, origin)
		if err != nil && !apperror.IsCode(err, apperror.CodeNotFound) {
			slog.WarnContext(ctx, "Failed to revoke the session on logout", "error", err)
		}
	}
	if principal.SessionID != "" {
		cookie.AddSessionTokenCookie(c, 0, "")
	}

	if req.callbackURL == "" {
		return ""
	}
	return appendStateToURL(req.callbackURL, req.state)
}

func bindEndSessionRequest(c *gin.Context) (dto.OidcLogoutDto, error) {
	var input dto.OidcLogoutDto

	switch c.Request.Method {
	case http.MethodGet:
		return input, c.ShouldBindQuery(&input)
	case http.MethodPost:
		// The spec only allows a form body, and picking the binder from Content-Type would hand anonymous input to the MessagePack and YAML decoders
		return input, c.ShouldBindWith(&input, binding.FormPost)
	default:
		return input, nil
	}
}

// endSessionQuery encodes a logout request as the query of a GET request
func endSessionQuery(input dto.OidcLogoutDto) string {
	query := url.Values{}
	for key, value := range map[string]string{
		"id_token_hint":            input.IdTokenHint,
		"client_id":                input.ClientId,
		"post_logout_redirect_uri": input.PostLogoutRedirectUri,
		"state":                    input.State,
	} {
		if value != "" {
			query.Set(key, value)
		}
	}
	return query.Encode()
}
