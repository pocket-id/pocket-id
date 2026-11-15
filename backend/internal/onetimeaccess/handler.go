package onetimeaccess

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
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

func (h *handler) createTokenForUser(c *gin.Context, input tokenCreateDto) (tokenCreatedDto, error) {
	// Get the target user ID from the URL and apply the default expiration when no TTL is provided
	userID := c.Param("id")
	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	if userID == "" {
		return tokenCreatedDto{}, apperror.MissingField("userId")
	}

	token, err := h.service.CreateToken(c.Request.Context(), userID, ttl, auditlogs.OriginFrom(c))
	if err != nil {
		return tokenCreatedDto{}, err
	}

	return tokenCreatedDto{Token: token}, nil
}

func (h *handler) requestEmailAsUnauthenticatedUser(c *gin.Context, input emailAsUnauthenticatedUserDto) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	deviceToken, err := h.service.RequestOneTimeAccessEmailAsUnauthenticatedUser(c.Request.Context(), dbConfig, input.Email, input.RedirectPath, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	cookie.AddDeviceTokenCookie(c, deviceToken)
	return nil
}

func (h *handler) requestEmailAsAdmin(c *gin.Context, input emailAsAdminDto) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	userID := c.Param("id")

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	return h.service.RequestOneTimeAccessEmailAsAdmin(c.Request.Context(), dbConfig, userID, ttl, auditlogs.OriginFrom(c))
}

func (h *handler) exchangeToken(c *gin.Context, input exchangeTokenDto) (dto.UserDto, error) {
	cfg, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	loginCode := input.Token
	// Reject values that cannot match either supported login code format
	if len(loginCode) != shortTokenLength && len(loginCode) != longTokenLength {
		return dto.UserDto{}, apperror.TokenInvalidOrExpired()
	}

	deviceToken, _ := c.Cookie(cookie.DeviceTokenCookieName)
	browserToken, _ := c.Cookie(cookie.KnownBrowserCookieName)
	remember := cookie.RememberSignIn(c)
	user, tokens, err := h.service.ExchangeToken(c.Request.Context(), cfg, loginCode, deviceToken, c.ClientIP(), c.Request.UserAgent(), browserToken, remember)
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return dto.UserDto{}, err
	}

	cookie.AddSignInSessionTokenCookie(c, cfg.SessionDuration.AsDurationMinutes(), remember, tokens.SessionToken)
	if tokens.KnownBrowserToken != "" {
		cookie.AddKnownBrowserCookie(c, tokens.KnownBrowserToken, int(auditlogs.KnownBrowserLifetime.Seconds()))
	}

	return userDto, nil
}
