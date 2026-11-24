package usersignup

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const defaultSignupTokenDuration = time.Hour

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) isInitialAdminSetupCompleted(ctx context.Context) (bool, error) {
	config, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return false, fmt.Errorf("error loading app configuration: %w", err)
	}

	return h.service.isInitialAdminSetupCompleted(ctx, h.service.db, config)
}

func (h *handler) checkInitialAdminSetupAvailable(c *gin.Context) error {
	setupCompleted, err := h.isInitialAdminSetupCompleted(c.Request.Context())
	if err != nil {
		return err
	}

	if setupCompleted {
		return apperror.SetupNotAvailable()
	}
	return nil
}

func (h *handler) signUpInitialAdmin(c *gin.Context, input signUpDto) (dto.UserDto, error) {
	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	user, token, err := h.service.SignUpInitialAdmin(c.Request.Context(), config, input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return dto.UserDto{}, err
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddSessionTokenCookie(c, maxAge, token)
	session.AnnounceBinding(c, token)

	return userDto, nil
}

func (h *handler) createSignupToken(c *gin.Context, input signupTokenCreateDto) (signupTokenDto, error) {
	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultSignupTokenDuration
	}

	signupToken, err := h.service.CreateSignupToken(c.Request.Context(), ttl, input.UsageLimit, input.UserGroupIDs, input.Email)
	if err != nil {
		return signupTokenDto{}, err
	}

	// Email the link when the admin entered an address
	if input.Email != "" {
		dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
		if err != nil {
			return signupTokenDto{}, fmt.Errorf("error loading app configuration: %w", err)
		}
		h.service.SendSignupInvitation(c.Request.Context(), dbConfig, signupToken, input.Email, ttl)
	}

	var tokenDto signupTokenDto
	err = dto.MapStruct(signupToken, &tokenDto)
	if err != nil {
		return signupTokenDto{}, err
	}

	return tokenDto, nil
}

func (h *handler) listSignupTokens(c *gin.Context) (dto.Paginated[signupTokenDto], error) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	tokens, pagination, err := h.service.ListSignupTokens(c.Request.Context(), listRequestOptions)
	if err != nil {
		return dto.Paginated[signupTokenDto]{}, err
	}

	var tokensDto []signupTokenDto
	err = dto.MapStructList(tokens, &tokensDto)
	if err != nil {
		return dto.Paginated[signupTokenDto]{}, err
	}

	return dto.Paginated[signupTokenDto]{
		Data:       tokensDto,
		Pagination: pagination,
	}, nil
}

func (h *handler) deleteSignupToken(c *gin.Context) error {
	tokenID := c.Param("id")

	return h.service.DeleteSignupToken(c.Request.Context(), tokenID)
}

func (h *handler) signup(c *gin.Context, input signUpDto) (dto.UserDto, error) {
	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	user, sessionToken, err := h.service.SignUp(c.Request.Context(), config, input, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		return dto.UserDto{}, err
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddSessionTokenCookie(c, maxAge, sessionToken)
	session.AnnounceBinding(c, sessionToken)

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return dto.UserDto{}, err
	}

	return userDto, nil
}
