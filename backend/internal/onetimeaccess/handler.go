package onetimeaccess

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const defaultTokenDuration = 15 * time.Minute

type handler struct {
	service   *Service
	appConfig AppConfigResolver
}

func newHandler(service *Service, appConfig AppConfigResolver) *handler {
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
func (h *handler) createTokenForUser(c *gin.Context) {
	var input tokenCreateDto
	err := c.ShouldBindJSON(&input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Get the target user ID from the URL and apply the default expiration when no TTL is provided
	userID := c.Param("id")
	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	if userID == "" {
		_ = c.Error(&common.UserIdNotProvidedError{})
		return
	}

	token, err := h.service.CreateToken(c.Request.Context(), userID, ttl)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
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
func (h *handler) requestEmailAsUnauthenticatedUser(c *gin.Context) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	var input emailAsUnauthenticatedUserDto
	err = dto.ShouldBindWithNormalizedJSON(c, &input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	deviceToken, err := h.service.RequestOneTimeAccessEmailAsUnauthenticatedUser(c.Request.Context(), dbConfig, input.Email, input.RedirectPath)
	if err != nil {
		_ = c.Error(err)
		return
	}

	cookie.AddDeviceTokenCookie(c, deviceToken)
	c.Status(http.StatusNoContent)
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
func (h *handler) requestEmailAsAdmin(c *gin.Context) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	var input emailAsAdminDto
	err = c.ShouldBindJSON(&input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	userID := c.Param("id")

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	err = h.service.RequestOneTimeAccessEmailAsAdmin(c.Request.Context(), dbConfig, userID, ttl)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// exchangeToken godoc
// @Summary Exchange one-time access token
// @Description Exchange a one-time access token for a session token
// @Tags Users
// @Param token path string true "One-time access token"
// @Success 200 {object} dto.UserDto
// @Router /api/one-time-access-token/{token} [post]
func (h *handler) exchangeToken(c *gin.Context) {
	cfg, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	loginCode := c.Param("token")
	// reject invalid length login codes
	if len(loginCode) != 6 && len(loginCode) != 16 {
		_ = c.Error(&common.TokenInvalidOrExpiredError{})
		return
	}

	deviceToken, _ := c.Cookie(cookie.DeviceTokenCookieName)
	user, token, err := h.service.ExchangeToken(c.Request.Context(), cfg, loginCode, deviceToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(cfg.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}
