package devicelogin

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	baseURL   string
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, baseURL string, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{
		service:   service,
		baseURL:   baseURL,
		appConfig: appConfig,
	}
}

func (h *handler) createRequest(c *gin.Context) (requestCreateDto, error) {
	request, deviceToken, err := h.service.Create(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return requestCreateDto{}, err
	}

	verificationURI := h.baseURL + "/device"
	verificationURIComplete := verificationURI + "?user_code=" + url.QueryEscape(request.Code)
	cookie.AddDeviceLoginTokenCookie(c, request.ID, deviceToken)
	return requestCreateDto{
		ID:                      request.ID,
		UserCode:                request.Code,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresAt:               request.ExpiresAt,
		Interval:                PollingInterval,
	}, nil
}

func (h *handler) exchangeRequest(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	requestID := c.Param("id")
	deviceToken, _ := c.Cookie(cookie.DeviceLoginTokenCookieName)
	sessionDuration := dbConfig.SessionDuration.AsDurationMinutes()
	browserToken, _ := c.Cookie(cookie.KnownBrowserCookieName)
	remember := cookie.RememberSignIn(c)
	user, tokens, status, err := h.service.Exchange(c.Request.Context(), requestID, deviceToken, c.ClientIP(), c.Request.UserAgent(), browserToken, sessionDuration, remember, dbConfig.EmailLoginNotificationMode)
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

	cookie.AddSignInSessionTokenCookie(c, sessionDuration, remember, tokens.SessionToken)
	if tokens.KnownBrowserToken != "" {
		cookie.AddKnownBrowserCookie(c, tokens.KnownBrowserToken, int(auditlogs.KnownBrowserLifetime.Seconds()))
	}
	c.JSON(http.StatusOK, dto.UserDto(user))
	return nil
}

func (h *handler) inspectRequest(c *gin.Context, input verificationDto) (verificationInfoDto, error) {
	info, err := h.service.Inspect(c.Request.Context(), input.Code)
	if err != nil {
		return verificationInfoDto{}, err
	}

	return verificationInfoDto(info), nil
}

func (h *handler) decideRequest(c *gin.Context, input decisionDto) error {
	reauthenticationToken, _ := c.Cookie(cookie.ReauthenticationTokenCookieName)
	return h.service.Decide(c.Request.Context(), input.Code, input.Decision, authz.PrincipalFrom(c).UserID, reauthenticationToken)
}
