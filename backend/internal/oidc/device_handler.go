package oidc

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type deviceHandler struct {
	provider      fosite.OAuth2Provider
	deviceService *deviceService
}

func newDeviceHandler(provider fosite.OAuth2Provider, deviceService *deviceService) *deviceHandler {
	return &deviceHandler{
		provider:      provider,
		deviceService: deviceService,
	}
}

func (h *deviceHandler) authorizeDevice(c *gin.Context) {
	ctx := c.Request.Context()

	response, request, err := h.deviceService.createDeviceAuthorization(ctx, c.Request)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create device authorization", "error", err)
		h.provider.WriteAccessError(ctx, c.Writer, request, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response)
}

func (h *deviceHandler) verifyDeviceCode(c *gin.Context) {
	principal := authz.PrincipalFrom(c)
	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)

	userCode := c.Query("code")
	if userCode == "" {
		_ = c.Error(apperror.MissingField("code"))
		return
	}

	err := h.deviceService.acceptDeviceCode(
		c.Request.Context(),
		userCode,
		principal.UserID,
		principal.AuthenticationMethod,
		principal.AuthenticationTime,
		reauthenticationToken,
		requestMetaFromGin(c),
	)
	if err != nil {
		if errors.Is(err, fosite.ErrAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You're not allowed to access this service."})
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *deviceHandler) deviceCodeInfo(c *gin.Context) {
	userCode := c.Query("code")
	if userCode == "" {
		_ = c.Error(apperror.MissingField("code"))
		return
	}

	deviceCodeInfo, err := h.deviceService.getDeviceCodeInfo(c.Request.Context(), userCode, authz.PrincipalFrom(c).UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, deviceCodeInfo)
}
