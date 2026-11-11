package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service *Service
}

// listOwn godoc
// @Summary List own sessions
// @Description Get the browsers in which the current user is signed in, most recently active first
// @Tags Sessions
// @Success 200 {array} sessionDto
// @Router /api/sessions [get]
func (h *handler) listOwn(c *gin.Context) error {
	principal := authz.PrincipalFrom(c)
	sessions, err := h.service.List(c.Request.Context(), principal.UserID)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, toSessionDtos(sessions, principal.SessionID))
	return nil
}

// revokeOwn godoc
// @Summary Revoke own session
// @Description Sign the current user out of one of their sessions
// @Tags Sessions
// @Param id path string true "Session ID"
// @Success 204 "No Content"
// @Router /api/sessions/{id} [delete]
func (h *handler) revokeOwn(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	err := h.service.Revoke(c.Request.Context(), origin.Principal.UserID, c.Param("id"), origin)
	if err != nil {
		return err
	}

	if c.Param("id") == origin.Principal.SessionID {
		cookie.AddSessionTokenCookie(c, 0, "")
	}
	c.Status(http.StatusNoContent)
	return nil
}

// revokeOtherOwn godoc
// @Summary Revoke other own sessions
// @Description Sign the current user out of every session except the current one
// @Tags Sessions
// @Success 204 "No Content"
// @Router /api/sessions [delete]
func (h *handler) revokeOtherOwn(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	err := h.service.RevokeAll(c.Request.Context(), origin.Principal.UserID, origin.Principal.SessionID, origin)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// rememberCurrent godoc
// @Summary Remember current session
// @Description Keep the current session signed in after the browser closes and while it goes unused, for a sign-in that was not remembered
// @Tags Sessions
// @Success 204 "No Content"
// @Router /api/sessions/current/remember [post]
func (h *handler) rememberCurrent(c *gin.Context) error {
	principal := authz.PrincipalFrom(c)
	session, err := h.service.Remember(c.Request.Context(), principal.UserID, principal.SessionID)
	if err != nil {
		return err
	}

	// The cookie of a sign-in that was not remembered ends with the browser, so it is set again with the session's lifetime
	token, _ := c.Cookie(cookie.SessionTokenCookieName)
	cookie.AddSessionTokenCookie(c, int(time.Until(session.ExpiresAt).Seconds()), token)
	c.Status(http.StatusNoContent)
	return nil
}

// listForUser godoc
// @Summary List sessions of a user
// @Description Get the browsers in which a user is signed in, most recently active first
// @Tags Sessions
// @Param id path string true "User ID"
// @Success 200 {array} sessionDto
// @Router /api/users/{id}/sessions [get]
func (h *handler) listForUser(c *gin.Context) error {
	sessions, err := h.service.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, toSessionDtos(sessions, authz.PrincipalFrom(c).SessionID))
	return nil
}

// revokeForUser godoc
// @Summary Revoke a session of a user
// @Description Sign a user out of one of their sessions
// @Tags Sessions
// @Param id path string true "User ID"
// @Param sessionId path string true "Session ID"
// @Success 204 "No Content"
// @Router /api/users/{id}/sessions/{sessionId} [delete]
func (h *handler) revokeForUser(c *gin.Context) error {
	err := h.service.Revoke(c.Request.Context(), c.Param("id"), c.Param("sessionId"), auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// revokeAllForUser godoc
// @Summary Revoke all sessions of a user
// @Description Sign a user out of every session and revoke every grant their apps hold, offline_access ones included
// @Tags Sessions
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Router /api/users/{id}/sessions [delete]
func (h *handler) revokeAllForUser(c *gin.Context) error {
	err := h.service.SignOutEverywhere(c.Request.Context(), c.Param("id"), auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
