package usersignup

import (
	"context"
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

const defaultSignupTokenDuration = time.Hour

type userOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      dto.UserDto
}

type signupInput struct {
	Body signUpDto
}

type tokenCreateInput struct {
	Body signupTokenCreateDto
}

type tokenIDInput struct {
	ID string `path:"id"`
}

type handler struct {
	service   *Service
	appConfig AppConfigProvider
}

func newHandler(service *Service, appConfig AppConfigProvider) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) checkInitialAdminSetupAvailable(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	setupCompleted, err := h.service.IsInitialAdminSetupCompleted(ctx)
	if err != nil {
		return nil, err
	}
	if setupCompleted {
		return nil, &common.SetupNotAvailableError{}
	}
	return &httpapi.EmptyOutput{}, nil
}

func (h *handler) signUpInitialAdmin(ctx context.Context, input *signupInput) (*userOutput, error) {
	user, token, err := h.service.SignUpInitialAdmin(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return h.userOutput(user, token)
}

func (h *handler) createSignupToken(ctx context.Context, input *tokenCreateInput) (*httpapi.BodyOutput[signupTokenDto], error) {
	ttl := input.Body.TTL.Duration
	if ttl <= 0 {
		ttl = defaultSignupTokenDuration
	}
	token, err := h.service.CreateSignupToken(ctx, ttl, input.Body.UsageLimit, input.Body.UserGroupIDs)
	if err != nil {
		return nil, err
	}
	var output signupTokenDto
	if err := dto.MapStruct(token, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[signupTokenDto]{Body: output}, nil
}

func (h *handler) listSignupTokens(ctx context.Context, input *httpapi.ListInput) (*httpapi.BodyOutput[dto.Paginated[signupTokenDto]], error) {
	tokens, pagination, err := h.service.ListSignupTokens(ctx, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}
	var output []signupTokenDto
	if err := dto.MapStructList(tokens, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[signupTokenDto]]{Body: dto.Paginated[signupTokenDto]{Data: output, Pagination: pagination}}, nil
}

func (h *handler) deleteSignupToken(ctx context.Context, input *tokenIDInput) (*httpapi.EmptyOutput, error) {
	if err := h.service.DeleteSignupToken(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (h *handler) signup(ctx context.Context, input *signupInput) (*userOutput, error) {
	user, accessToken, err := h.service.SignUp(ctx, input.Body, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx))
	if err != nil {
		return nil, err
	}
	return h.userOutput(user, accessToken)
}

func (h *handler) userOutput(user model.User, accessToken string) (*userOutput, error) {
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	maxAge := int(h.appConfig.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	return &userOutput{SetCookie: []http.Cookie{*cookie.NewAccessTokenCookie(maxAge, accessToken)}, Body: output}, nil
}
