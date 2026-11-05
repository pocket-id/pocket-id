package oidc

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type endSessionHandler struct {
	endSessionService *endSessionService
	sessions          SessionRevoker
	baseURL           string
}

func newEndSessionHandler(endSessionService *endSessionService, sessions SessionRevoker, baseURL string) *endSessionHandler {
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

	principal := authz.PrincipalFrom(c)
	callbackURL, err := h.endSessionService.endSession(c.Request.Context(), input, principal.UserID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "Error getting logout callback URL, the user has to confirm the logout manually", "error", err)
		c.Redirect(http.StatusFound, h.baseURL+"/logout")
		return
	}

	// The redirect must still happen if revoking fails, and the cleared cookie signs the browser out either way
	if principal.SessionID != "" {
		err = h.sessions.Revoke(c.Request.Context(), principal.UserID, principal.SessionID)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "Failed to revoke the session on logout", "error", err)
		}
	}
	cookie.AddSessionTokenCookie(c, 0, "")
	if callbackURL == "" {
		c.Redirect(http.StatusFound, h.baseURL+"/logout")
		return
	}

	c.Redirect(http.StatusFound, appendStateToURL(callbackURL, input.State))
}

func bindEndSessionRequest(c *gin.Context) (dto.OidcLogoutDto, error) {
	var input dto.OidcLogoutDto

	switch c.Request.Method {
	case http.MethodGet:
		return input, c.ShouldBindQuery(&input)
	case http.MethodPost:
		return input, c.ShouldBind(&input)
	default:
		return input, nil
	}
}
