package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
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

type deviceCodeInput struct {
	Code string `query:"code" required:"true"`
}

func (h *deviceHandler) verifyDeviceCode(ctx context.Context, input *deviceCodeInput) (*httpapi.EmptyOutput, error) {
	reauthenticationToken := ""
	if requestCookie, err := httpapi.Cookie(ctx, cookie.ReauthenticationTokenCookieName); err == nil {
		reauthenticationToken = requestCookie.Value
	}

	err := h.deviceService.acceptDeviceCode(
		ctx,
		input.Code,
		httpapi.UserID(ctx),
		httpapi.AuthenticationMethod(ctx),
		httpapi.AuthenticationTime(ctx),
		reauthenticationToken,
		requestMetaFromContext(ctx),
	)
	if err != nil {
		if errors.Is(err, fosite.ErrAccessDenied) {
			return nil, &common.OidcAccessDeniedError{}
		}
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (h *deviceHandler) deviceCodeInfo(ctx context.Context, input *deviceCodeInput) (*httpapi.BodyOutput[dto.DeviceCodeInfoDto], error) {
	deviceCodeInfo, err := h.deviceService.getDeviceCodeInfo(ctx, input.Code, httpapi.UserID(ctx))
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.DeviceCodeInfoDto]{Body: *deviceCodeInfo}, nil
}
