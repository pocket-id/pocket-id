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
	sessions          SessionEnder
	baseURL           string
}

func newEndSessionHandler(endSessionService *endSessionService, sessions SessionEnder, baseURL string) *endSessionHandler {
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

	callbackURL, err := h.endSessionService.endSession(c.Request.Context(), input, authz.PrincipalFrom(c).UserID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "Error getting logout callback URL, the user has to confirm the logout manually", "error", err)
		c.Redirect(http.StatusFound, h.baseURL+"/logout")
		return
	}

	// Deliberately only on this success path, so an unvalidated logout request can't end the session; best-effort because the RP is already owed its redirect
	if sessionID := authz.PrincipalFrom(c).SessionID; sessionID != "" {
		err = h.sessions.Delete(c.Request.Context(), sessionID)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "Failed to end user session on RP-initiated logout", "error", err)
		}
	}

	cookie.AddAccessTokenCookie(c, 0, "")
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
