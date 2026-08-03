package devicelogin

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	baseURL   string
	appConfig AppConfigProvider
}

func newHandler(service *Service, baseURL string, appConfig AppConfigProvider) *handler {
	return &handler{
		service:   service,
		baseURL:   baseURL,
		appConfig: appConfig,
	}
}

// createRequest godoc
// @Summary Create device login request
// @Description Create a short-lived request that can be approved from another authenticated device
// @Tags Device Login
// @Produce json
// @Success 201 {object} requestCreateDto "Created device login request"
// @Router /api/device-login/requests [post]
func (h *handler) createRequest(c *gin.Context) error {
	request, deviceToken, err := h.service.Create(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return err
	}

	verificationURI := h.baseURL + "/device"
	verificationURIComplete := verificationURI + "?user_code=" + url.QueryEscape(request.Code)
	cookie.AddDeviceLoginTokenCookie(c, request.ID, deviceToken)
	c.JSON(http.StatusCreated, requestCreateDto{
		ID:                      request.ID,
		UserCode:                request.Code,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresAt:               request.ExpiresAt,
		Interval:                PollingInterval,
	})
	return nil
}

// exchangeRequest godoc
// @Summary Exchange device login request
// @Description Wait for a device login decision and create a browser session after it has been approved
// @Tags Device Login
// @Produce json
// @Param id path string true "Device login request ID"
// @Success 200 {object} dto.UserDto "Approved request exchanged for a user session"
// @Success 202 "Authorization pending"
// @Router /api/device-login/requests/{id}/exchange [post]
func (h *handler) exchangeRequest(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	requestID := c.Param("id")
	deviceToken, _ := c.Cookie(cookie.DeviceLoginTokenCookieName)
	sessionDuration := dbConfig.SessionDuration.AsDurationMinutes()
	user, accessToken, status, err := h.service.Exchange(c.Request.Context(), requestID, deviceToken, c.ClientIP(), c.Request.UserAgent(), sessionDuration)
	if err != nil {
		if c.Request.Context().Err() != nil {
			// Context canceled = the client stopped the request
			// Nothing to do here
			return c.Request.Context().Err()
		}
		return err
	}
	if status == RequestStatusPending {
		// Request is pending, so respond with a 202
		c.Status(http.StatusAccepted)
		return nil
	}

	maxAge := int(sessionDuration.Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)
	c.JSON(http.StatusOK, dto.UserDto(user))
	return nil
}

// inspectRequest godoc
// @Summary Inspect device login request
// @Description Retrieve the requesting device details for an authenticated user before approval or denial
// @Tags Device Login
// @Accept json
// @Produce json
// @Param request body verificationDto true "Device login code"
// @Success 200 {object} verificationInfoDto "Device login request details"
// @Router /api/device-login/verification [post]
func (h *handler) inspectRequest(c *gin.Context) error {
	var input verificationDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	info, err := h.service.Inspect(c.Request.Context(), input.Code)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, verificationInfoDto(info))
	return nil
}

// decideRequest godoc
// @Summary Decide device login request
// @Description Approve or deny a device login request; approval requires fresh passkey reauthentication
// @Tags Device Login
// @Accept json
// @Param decision body decisionDto true "Device login decision"
// @Success 204 "No Content"
// @Router /api/device-login/verification/decision [post]
func (h *handler) decideRequest(c *gin.Context) error {
	var input decisionDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)
	err = h.service.Decide(c.Request.Context(), input.Code, input.Decision, c.GetString("userID"), reauthenticationToken)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
