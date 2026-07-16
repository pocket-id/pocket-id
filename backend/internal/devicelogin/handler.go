package devicelogin

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
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
func (h *handler) createRequest(c *gin.Context) {
	request, deviceToken, err := h.service.Create(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
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
}

// exchangeRequest godoc
// @Summary Exchange device login request
// @Description Poll a device login request and create a browser session after it has been approved
// @Tags Device Login
// @Produce json
// @Param id path string true "Device login request ID"
// @Success 200 {object} dto.UserDto "Approved request exchanged for a user session"
// @Success 202 "Authorization pending"
// @Router /api/device-login/requests/{id}/exchange [post]
func (h *handler) exchangeRequest(c *gin.Context) {
	requestID := c.Param("id")
	deviceToken, _ := c.Cookie(cookie.DeviceLoginTokenCookieName)
	user, accessToken, status, err := h.service.Exchange(c.Request.Context(), requestID, deviceToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
	}
	if status == RequestStatusPending {
		c.Status(http.StatusAccepted)
		return
	}

	var userDto dto.UserDto
	if err = dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(h.appConfig.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)
	c.JSON(http.StatusOK, userDto)
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
func (h *handler) inspectRequest(c *gin.Context) {
	var input verificationDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	info, err := h.service.Inspect(c.Request.Context(), input.Code)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, verificationInfoDto(info))
}

// decideRequest godoc
// @Summary Decide device login request
// @Description Approve or deny a device login request; approval requires fresh passkey reauthentication
// @Tags Device Login
// @Accept json
// @Param decision body decisionDto true "Device login decision"
// @Success 204 "No Content"
// @Router /api/device-login/verification/decision [post]
func (h *handler) decideRequest(c *gin.Context) {
	var input decisionDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)
	if err := h.service.Decide(c.Request.Context(), input.Code, input.Decision, c.GetString("userID"), reauthenticationToken); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
