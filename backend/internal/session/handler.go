package session

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	principal := authz.PrincipalFrom(c)
	err := h.service.Revoke(c.Request.Context(), principal.UserID, c.Param("id"))
	if err != nil {
		return err
	}

	if c.Param("id") == principal.SessionID {
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
	principal := authz.PrincipalFrom(c)
	err := h.service.RevokeAll(c.Request.Context(), principal.UserID, principal.SessionID)
	if err != nil {
		return err
	}

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
	err := h.service.Revoke(c.Request.Context(), c.Param("id"), c.Param("sessionId"))
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
	err := h.service.SignOutEverywhere(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
