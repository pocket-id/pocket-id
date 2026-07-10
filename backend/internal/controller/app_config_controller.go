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

	httpapi.Register(api, appConfigOperation("list-public-application-configuration", http.MethodGet, "/api/application-configuration", "List public application configurations"), controller.listAppConfigHandler)

	auth := authMiddleware.Huma(api)
	allOperation := appConfigOperation("list-all-application-configuration", http.MethodGet, "/api/application-configuration/all", "List all application configurations")
	auth(&allOperation)
	httpapi.Register(api, allOperation, controller.listAllAppConfigHandler)

	updateOperation := appConfigOperation("update-application-configuration", http.MethodPut, "/api/application-configuration", "Update application configurations")
	auth(&updateOperation)
	httpapi.Register(api, updateOperation, controller.updateAppConfigHandler)

	testEmailOperation := appConfigOperation("test-email-configuration", http.MethodPost, "/api/application-configuration/test-email", "Send test email")
	testEmailOperation.DefaultStatus = http.StatusNoContent
	auth(&testEmailOperation)
	httpapi.Register(api, testEmailOperation, controller.testEmailHandler)

	syncLDAPOperation := appConfigOperation("sync-ldap", http.MethodPost, "/api/application-configuration/sync-ldap", "Synchronize LDAP")
	syncLDAPOperation.DefaultStatus = http.StatusNoContent
	auth(&syncLDAPOperation)
	httpapi.Register(api, syncLDAPOperation, controller.syncLDAPHandler)
}

func appConfigOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"Application Configuration"}}
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
