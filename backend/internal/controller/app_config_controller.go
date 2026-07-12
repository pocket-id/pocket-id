package controller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type appConfigUpdateInput struct {
	Body dto.AppConfigUpdateDto
}

// NewAppConfigController registers application configuration endpoints
func NewAppConfigController(
	api huma.API,
	authMiddleware *middleware.AuthMiddleware,
	appConfigService *service.AppConfigService,
	emailService *service.EmailService,
	ldapService *service.LdapService,
) {
	controller := &AppConfigController{appConfigService: appConfigService, emailService: emailService, ldapService: ldapService}
	auth := authMiddleware.Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-public-application-configuration",
		Method:      http.MethodGet,
		Path:        "/api/application-configuration",
		Summary:     "List public application configurations",
		Tags:        []string{"Application Configuration"},
	}, controller.listAppConfigHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-all-application-configuration",
		Method:      http.MethodGet,
		Path:        "/api/application-configuration/all",
		Summary:     "List all application configurations",
		Tags:        []string{"Application Configuration"},
	}, controller.listAllAppConfigHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-application-configuration",
		Method:      http.MethodPut,
		Path:        "/api/application-configuration",
		Summary:     "Update application configurations",
		Tags:        []string{"Application Configuration"},
	}, controller.updateAppConfigHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "test-email-configuration",
		Method:        http.MethodPost,
		Path:          "/api/application-configuration/test-email",
		Summary:       "Send test email",
		Tags:          []string{"Application Configuration"},
		DefaultStatus: http.StatusNoContent,
	}, controller.testEmailHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "sync-ldap",
		Method:        http.MethodPost,
		Path:          "/api/application-configuration/sync-ldap",
		Summary:       "Synchronize LDAP",
		Tags:          []string{"Application Configuration"},
		DefaultStatus: http.StatusNoContent,
	}, controller.syncLDAPHandler, auth)
}

type AppConfigController struct {
	appConfigService *service.AppConfigService
	emailService     *service.EmailService
	ldapService      *service.LdapService
}

func (acc *AppConfigController) listAppConfigHandler(_ context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[[]dto.PublicAppConfigVariableDto], error) {
	configuration := acc.appConfigService.ListAppConfig(false)

	var output []dto.PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &output); err != nil {
		return nil, err
	}
	output = append(output,
		dto.PublicAppConfigVariableDto{Key: "uiConfigDisabled", Value: strconv.FormatBool(common.EnvConfig.UiConfigDisabled), Type: "boolean"},
		dto.PublicAppConfigVariableDto{Key: "tracingEnabled", Value: strconv.FormatBool(tracing.FrontendTracingEnabled()), Type: "boolean"},
	)
	return &httpapi.BodyOutput[[]dto.PublicAppConfigVariableDto]{Body: output}, nil
}

func (acc *AppConfigController) listAllAppConfigHandler(_ context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[[]dto.AppConfigVariableDto], error) {
	configuration := acc.appConfigService.ListAppConfig(true)
	var output []dto.AppConfigVariableDto
	if err := dto.MapStructList(configuration, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.AppConfigVariableDto]{Body: output}, nil
}

func (acc *AppConfigController) updateAppConfigHandler(ctx context.Context, input *appConfigUpdateInput) (*httpapi.BodyOutput[[]dto.AppConfigVariableDto], error) {
	saved, err := acc.appConfigService.UpdateAppConfig(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	var output []dto.AppConfigVariableDto
	if err := dto.MapStructList(saved, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.AppConfigVariableDto]{Body: output}, nil
}

func (acc *AppConfigController) syncLDAPHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	if err := acc.ldapService.SyncAll(ctx); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (acc *AppConfigController) testEmailHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	if err := acc.emailService.SendTestEmail(ctx, httpapi.UserID(ctx)); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}
