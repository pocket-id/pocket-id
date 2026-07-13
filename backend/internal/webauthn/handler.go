package webauthn

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type emptyOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type bodyOutput[T any] struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      T
}

type credentialBodyInput struct {
	Body json.RawMessage
}

type optionalCredentialBodyInput struct {
	Body *json.RawMessage `required:"false"`
}

type credentialIDInput struct {
	ID string `path:"id"`
}

type credentialUpdateInput struct {
	ID   string `path:"id"`
	Body dto.WebauthnCredentialUpdateDto
}

type handler struct {
	service   *Service
	appConfig AppConfigProvider
}

func newHandler(service *Service, appConfig AppConfigProvider) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) beginRegistration(ctx context.Context, _ *httpapi.EmptyInput) (*bodyOutput[protocol.PublicKeyCredentialCreationOptions], error) {
	options, err := h.service.BeginRegistration(ctx, httpapi.UserID(ctx))
	if err != nil {
		return nil, err
	}
	return &bodyOutput[protocol.PublicKeyCredentialCreationOptions]{
		SetCookie: []http.Cookie{*cookie.NewSessionIDCookie(int(options.Timeout.Seconds()), options.SessionID)},
		Body:      options.Response,
	}, nil
}

func (h *handler) verifyRegistration(ctx context.Context, input *credentialBodyInput) (*bodyOutput[dto.WebauthnCredentialDto], error) {
	sessionID, err := sessionID(ctx)
	if err != nil {
		return nil, err
	}
	request := requestWithBody(ctx, input.Body)
	credential, err := h.service.VerifyRegistration(ctx, sessionID, httpapi.UserID(ctx), request, httpapi.ClientIP(ctx))
	if err != nil {
		return nil, err
	}
	var output dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &output); err != nil {
		return nil, err
	}
	return &bodyOutput[dto.WebauthnCredentialDto]{Body: output}, nil
}

func (h *handler) beginLogin(ctx context.Context, _ *httpapi.EmptyInput) (*bodyOutput[protocol.PublicKeyCredentialRequestOptions], error) {
	options, err := h.service.BeginLogin(ctx)
	if err != nil {
		return nil, err
	}
	return &bodyOutput[protocol.PublicKeyCredentialRequestOptions]{
		SetCookie: []http.Cookie{*cookie.NewSessionIDCookie(int(options.Timeout.Seconds()), options.SessionID)},
		Body:      options.Response,
	}, nil
}

func (h *handler) verifyLogin(ctx context.Context, input *credentialBodyInput) (*bodyOutput[dto.UserDto], error) {
	sessionID, err := sessionID(ctx)
	if err != nil {
		return nil, err
	}
	assertion, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Body))
	if err != nil {
		return nil, err
	}
	user, token, err := h.service.VerifyLogin(ctx, sessionID, assertion, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx))
	if err != nil {
		return nil, err
	}
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	maxAge := int(h.appConfig.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	return &bodyOutput[dto.UserDto]{SetCookie: []http.Cookie{*cookie.NewAccessTokenCookie(maxAge, token)}, Body: output}, nil
}

func (h *handler) listCredentials(ctx context.Context, _ *httpapi.EmptyInput) (*bodyOutput[[]dto.WebauthnCredentialDto], error) {
	credentials, err := h.service.ListCredentials(ctx, httpapi.UserID(ctx))
	if err != nil {
		return nil, err
	}
	var output []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &output); err != nil {
		return nil, err
	}
	return &bodyOutput[[]dto.WebauthnCredentialDto]{Body: output}, nil
}

func (h *handler) deleteCredential(ctx context.Context, input *credentialIDInput) (*emptyOutput, error) {
	userID := httpapi.UserID(ctx)
	if err := h.service.DeleteCredential(ctx, userID, input.ID, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx), userID); err != nil {
		return nil, err
	}
	return &emptyOutput{}, nil
}

func (h *handler) updateCredential(ctx context.Context, input *credentialUpdateInput) (*bodyOutput[dto.WebauthnCredentialDto], error) {
	credential, err := h.service.UpdateCredential(ctx, httpapi.UserID(ctx), input.ID, input.Body.Name)
	if err != nil {
		return nil, err
	}
	var output dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &output); err != nil {
		return nil, err
	}
	return &bodyOutput[dto.WebauthnCredentialDto]{Body: output}, nil
}

func (h *handler) logout(_ context.Context, _ *httpapi.EmptyInput) (*emptyOutput, error) {
	return &emptyOutput{SetCookie: []http.Cookie{*cookie.NewAccessTokenCookie(-1, "")}}, nil
}

func (h *handler) reauthenticate(ctx context.Context, input *optionalCredentialBodyInput) (*emptyOutput, error) {
	var token string
	var err error
	if input.Body != nil {
		assertion, parseErr := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(*input.Body))
		if parseErr == nil {
			sessionCookieID, sessionErr := sessionID(ctx)
			if sessionErr != nil {
				return nil, sessionErr
			}
			token, err = h.service.CreateReauthenticationTokenWithWebauthn(ctx, sessionCookieID, assertion)
		} else {
			token, err = h.reauthenticateWithAccessToken(ctx)
		}
	} else {
		token, err = h.reauthenticateWithAccessToken(ctx)
	}
	if err != nil {
		return nil, err
	}
	return &emptyOutput{SetCookie: []http.Cookie{*cookie.NewReauthenticationTokenCookie(token)}}, nil
}

func (h *handler) reauthenticateWithAccessToken(ctx context.Context) (string, error) {
	accessToken, _ := httpapi.Cookie(ctx, cookie.AccessTokenCookieName)
	value := ""
	if accessToken != nil {
		value = accessToken.Value
	}
	return h.service.CreateReauthenticationTokenWithAccessToken(ctx, value)
}

func sessionID(ctx context.Context) (string, error) {
	id, err := httpapi.Cookie(ctx, cookie.SessionIdCookieName)
	if err != nil {
		return "", &common.MissingSessionIdError{}
	}
	return id.Value, nil
}

func requestWithBody(ctx context.Context, body []byte) *http.Request {
	request := httpapi.Request(ctx).Clone(ctx)
	request.Body = http.NoBody
	if len(body) > 0 {
		request.Body = io.NopCloser(bytes.NewReader(body))
	}
	request.ContentLength = int64(len(body))
	return request
}
