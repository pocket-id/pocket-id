package onetimeaccess

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const defaultTokenDuration = 15 * time.Minute

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

// createTokenForUser godoc
// @Summary Create one-time access token for user (admin)
// @Description Generate a one-time access token for a specific user (admin only)
// @Tags Users
// @Param id path string true "User ID"
// @Param body body tokenCreateDto true "Token options"
// @Success 201 {object} object "{ \"token\": \"string\" }"
// @Router /api/users/{id}/one-time-access-token [post]
func (h *handler) createTokenForUser(c *gin.Context) error {
	var input tokenCreateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	// Get the target user ID from the URL and apply the default expiration when no TTL is provided
	userID := c.Param("id")
	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	if userID == "" {
		return apperror.MissingField("userId")
	}

	token, err := h.service.CreateToken(c.Request.Context(), userID, ttl, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
	return nil
}

// requestEmailAsUnauthenticatedUser godoc
// @Summary Request one-time access email
// @Description Request a one-time access email for unauthenticated users
// @Tags Users
// @Accept json
// @Produce json
// @Param body body emailAsUnauthenticatedUserDto true "Email request information"
// @Success 204 "No Content"
// @Router /api/one-time-access-email [post]
func (h *handler) requestEmailAsUnauthenticatedUser(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input emailAsUnauthenticatedUserDto
	err = httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	deviceToken, err := h.service.RequestOneTimeAccessEmailAsUnauthenticatedUser(c.Request.Context(), dbConfig, input.Email, input.RedirectPath, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	cookie.AddDeviceTokenCookie(c, deviceToken)
	c.Status(http.StatusNoContent)
	return nil
}

// requestEmailAsAdmin godoc
// @Summary Request one-time access email (admin)
// @Description Request a one-time access email for a specific user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body emailAsAdminDto true "Email request options"
// @Success 204 "No Content"
// @Router /api/users/{id}/one-time-access-email [post]
func (h *handler) requestEmailAsAdmin(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input emailAsAdminDto
	err = httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	userID := c.Param("id")

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	err = h.service.RequestOneTimeAccessEmailAsAdmin(c.Request.Context(), dbConfig, userID, ttl, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// exchangeToken godoc
// @Summary Exchange one-time access token
// @Description Exchange a one-time access token for a session token
// @Tags Users
// @Accept json
// @Param body body exchangeTokenDto true "One-time access token"
// @Param remember query bool false "Set to false for a session that ends when the browser closes or after 20 minutes without activity" default(true)
// @Success 200 {object} dto.UserDto
// @Router /api/one-time-access-token [post]
func (h *handler) exchangeToken(c *gin.Context) error {
	cfg, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input exchangeTokenDto
	err = httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	loginCode := input.Token
	// Reject values that cannot match either supported login code format
	if len(loginCode) != shortTokenLength && len(loginCode) != longTokenLength {
		return apperror.TokenInvalidOrExpired()
	}

	deviceToken, _ := c.Cookie(cookie.DeviceTokenCookieName)
	browserToken, _ := c.Cookie(cookie.KnownBrowserCookieName)
	remember := cookie.RememberSignIn(c)
	user, tokens, err := h.service.ExchangeToken(c.Request.Context(), cfg, loginCode, deviceToken, c.ClientIP(), c.Request.UserAgent(), browserToken, remember)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return err
	}

	cookie.AddSignInSessionTokenCookie(c, cfg.SessionDuration.AsDurationMinutes(), remember, tokens.SessionToken)
	if tokens.KnownBrowserToken != "" {
		cookie.AddKnownBrowserCookie(c, tokens.KnownBrowserToken, int(auditlogs.KnownBrowserLifetime.Seconds()))
	}

	c.JSON(http.StatusOK, userDto)
	return nil
}
