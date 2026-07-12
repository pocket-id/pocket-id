//go:build e2etest

package controller

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/service"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type testResetInput struct {
	SkipLDAP string `query:"skip-ldap" required:"false"`
	SkipSeed string `query:"skip-seed" required:"false"`
}

type testExternalIDPInput struct {
	Body struct {
		Audience string `json:"aud" required:"true"`
		Issuer   string `json:"iss" required:"true"`
		Subject  string `json:"sub" required:"true"`
	}
}

type testAccessTokenInput struct {
	Body struct {
		UserID   string `json:"user" required:"true"`
		ClientID string `json:"client" required:"true"`
		Expired  bool   `json:"expired" required:"false"`
	}
}

type testRefreshTokenInput struct {
	Body struct {
		UserID       string `json:"user" required:"true"`
		ClientID     string `json:"client" required:"true"`
		RefreshToken string `json:"rt" required:"true"`
	}
}

type testBytesOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

func NewTestController(api huma.API, testService *service.TestService) {
	controller := &TestController{TestService: testService}

	httpapi.Register(api, huma.Operation{
		OperationID:   "test-reset",
		Method:        http.MethodPost,
		Path:          "/api/test/reset",
		Tags:          []string{"E2E Test"},
		Hidden:        true,
		DefaultStatus: http.StatusNoContent,
	}, controller.resetAndSeedHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "test-sign-access-token",
		Method:      http.MethodPost,
		Path:        "/api/test/accesstoken",
		Tags:        []string{"E2E Test"},
		Hidden:      true,
	}, controller.signAccessToken)

	httpapi.Register(api, huma.Operation{
		OperationID: "test-sign-refresh-token",
		Method:      http.MethodPost,
		Path:        "/api/test/refreshtoken",
		Tags:        []string{"E2E Test"},
		Hidden:      true,
	}, controller.signRefreshToken)

	httpapi.Register(api, huma.Operation{
		OperationID: "test-external-idp-jwks",
		Method:      http.MethodGet,
		Path:        "/api/externalidp/jwks.json",
		Tags:        []string{"E2E Test"},
		Hidden:      true,
	}, controller.externalIDPJWKS)

	httpapi.Register(api, huma.Operation{
		OperationID: "test-external-idp-sign",
		Method:      http.MethodPost,
		Path:        "/api/externalidp/sign",
		Tags:        []string{"E2E Test"},
		Hidden:      true,
	}, controller.externalIDPSignToken)
}

type TestController struct {
	TestService *service.TestService
}

func (tc *TestController) resetAndSeedHandler(ctx context.Context, input *testResetInput) (*httpapi.EmptyOutput, error) {
	request := httpapi.Request(ctx)
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + request.Host

	if err := tc.TestService.ResetDatabase(); err != nil {
		return nil, err
	}
	if err := tc.TestService.ResetLock(ctx); err != nil {
		return nil, err
	}
	if err := tc.TestService.ResetApplicationImages(ctx); err != nil {
		return nil, err
	}
	if input.SkipSeed != "true" {
		if err := tc.TestService.SeedDatabase(baseURL); err != nil {
			return nil, err
		}
	}
	if err := tc.TestService.ResetAppConfig(ctx); err != nil {
		return nil, err
	}
	if input.SkipLDAP != "true" {
		if err := tc.TestService.SetLdapTestConfig(ctx); err != nil {
			return nil, err
		}
		if err := tc.TestService.SyncLdap(ctx); err != nil {
			return nil, err
		}
	}
	return &httpapi.EmptyOutput{}, nil
}

func (tc *TestController) externalIDPJWKS(_ context.Context, _ *httpapi.EmptyInput) (*testBytesOutput, error) {
	jwks, err := tc.TestService.GetExternalIdPJWKS()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(jwks)
	if err != nil {
		return nil, err
	}
	return &testBytesOutput{ContentType: "application/json; charset=utf-8", Body: body}, nil
}

func (tc *TestController) externalIDPSignToken(_ context.Context, input *testExternalIDPInput) (*testBytesOutput, error) {
	token, err := tc.TestService.SignExternalIdPToken(input.Body.Issuer, input.Body.Subject, input.Body.Audience)
	if err != nil {
		return nil, err
	}
	return &testBytesOutput{ContentType: "text/plain; charset=utf-8", Body: []byte(token)}, nil
}

func (tc *TestController) signAccessToken(ctx context.Context, input *testAccessTokenInput) (*testBytesOutput, error) {
	token, err := tc.TestService.SignAccessToken(ctx, input.Body.UserID, input.Body.ClientID, input.Body.Expired)
	if err != nil {
		return nil, err
	}
	return &testBytesOutput{ContentType: "text/plain; charset=utf-8", Body: []byte(token)}, nil
}

func (tc *TestController) signRefreshToken(ctx context.Context, input *testRefreshTokenInput) (*testBytesOutput, error) {
	token, err := tc.TestService.SignRefreshToken(ctx, input.Body.UserID, input.Body.ClientID, input.Body.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &testBytesOutput{ContentType: "text/plain; charset=utf-8", Body: []byte(token)}, nil
}
