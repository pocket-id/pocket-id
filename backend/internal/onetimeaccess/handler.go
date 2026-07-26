package onetimeaccess

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

const defaultTokenDuration = 15 * time.Minute

type tokenCreateInput struct {
	ID   string `path:"id"`
	Body tokenCreateDto
}

type ownTokenCreateInput struct {
	Body tokenCreateDto
}

type emailAsUnauthenticatedUserInput struct {
	Body emailAsUnauthenticatedUserDto
}

type emailAsAdminInput struct {
	ID   string `path:"id"`
	Body emailAsAdminDto
}

type tokenExchangeInput struct {
	Token string `path:"token"`
}

type cookieOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type userOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      dto.UserDto
}

type handler struct {
	service   *Service
	appConfig AppConfigResolver
}

func newHandler(service *Service, appConfig AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) createOwnToken(ctx context.Context, _ *ownTokenCreateInput) (*httpapi.BodyOutput[map[string]string], error) {
	return h.createToken(ctx, httpapi.UserID(ctx), defaultTokenDuration)
}

func (h *handler) createTokenForUser(ctx context.Context, input *tokenCreateInput) (*httpapi.BodyOutput[map[string]string], error) {
	ttl := input.Body.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	return h.createToken(ctx, input.ID, ttl)
}

func (h *handler) createToken(ctx context.Context, userID string, ttl time.Duration) (*httpapi.BodyOutput[map[string]string], error) {
	if userID == "" {
		return nil, &common.UserIdNotProvidedError{}
	}
	token, err := h.service.CreateToken(ctx, userID, ttl)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[map[string]string]{Body: map[string]string{"token": token}}, nil
}

func (h *handler) requestEmailAsUnauthenticatedUser(ctx context.Context, input *emailAsUnauthenticatedUserInput) (*cookieOutput, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	deviceToken, err := h.service.RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx, dbConfig, input.Body.Email, input.Body.RedirectPath)
	if err != nil {
		return nil, err
	}
	return &cookieOutput{SetCookie: []http.Cookie{*cookie.NewDeviceTokenCookie(deviceToken)}}, nil
}

func (h *handler) requestEmailAsAdmin(ctx context.Context, input *emailAsAdminInput) (*httpapi.EmptyOutput, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	ttl := input.Body.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}
	if err := h.service.RequestOneTimeAccessEmailAsAdmin(ctx, dbConfig, input.ID, ttl); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (h *handler) exchangeToken(ctx context.Context, input *tokenExchangeInput) (*userOutput, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	if len(input.Token) != 6 && len(input.Token) != 16 {
		return nil, &common.TokenInvalidOrExpiredError{}
	}
	deviceToken := ""
	if requestCookie, err := httpapi.Cookie(ctx, cookie.DeviceTokenCookieName); err == nil {
		deviceToken = requestCookie.Value
	}
	user, token, err := h.service.ExchangeToken(ctx, dbConfig, input.Token, deviceToken, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx))
	if err != nil {
		return nil, err
	}
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	maxAge := int(dbConfig.SessionDuration.AsDurationMinutes().Seconds())
	return &userOutput{SetCookie: []http.Cookie{*cookie.NewAccessTokenCookie(maxAge, token)}, Body: output}, nil
}
