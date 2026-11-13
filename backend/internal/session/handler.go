package session

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service *Service
}

func (h *handler) listOwn(c *gin.Context) ([]sessionDto, error) {
	principal := authz.PrincipalFrom(c)
	sessions, err := h.service.List(c.Request.Context(), principal.UserID)
	if err != nil {
		return nil, err
	}

	return toSessionDtos(sessions, principal.SessionID), nil
}

func (h *handler) revokeOwn(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	err := h.service.Revoke(c.Request.Context(), origin.Principal.UserID, c.Param("id"), origin)
	if err != nil {
		return err
	}

	if c.Param("id") == origin.Principal.SessionID {
		cookie.AddSessionTokenCookie(c, 0, "")
	}
	return nil
}

func (h *handler) revokeOtherOwn(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	return h.service.RevokeAll(c.Request.Context(), origin.Principal.UserID, origin.Principal.SessionID, origin)
}

func (h *handler) rememberCurrent(c *gin.Context) error {
	principal := authz.PrincipalFrom(c)
	session, err := h.service.Remember(c.Request.Context(), principal.UserID, principal.SessionID)
	if err != nil {
		return err
	}

	// The cookie of a sign-in that was not remembered ends with the browser, so it is set again with the session's lifetime
	token, _ := c.Cookie(cookie.SessionTokenCookieName)
	cookie.AddSessionTokenCookie(c, int(time.Until(session.ExpiresAt).Seconds()), token)
	return nil
}

func (h *handler) listForUser(c *gin.Context) ([]sessionDto, error) {
	sessions, err := h.service.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		return nil, err
	}

	return toSessionDtos(sessions, authz.PrincipalFrom(c).SessionID), nil
}

func (h *handler) revokeForUser(c *gin.Context) error {
	return h.service.Revoke(c.Request.Context(), c.Param("id"), c.Param("sessionId"), auditlogs.OriginFrom(c))
}

func (h *handler) revokeAllForUser(c *gin.Context) error {
	return h.service.SignOutEverywhere(c.Request.Context(), c.Param("id"), auditlogs.OriginFrom(c))
}
