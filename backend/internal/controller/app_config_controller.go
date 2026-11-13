package controller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
)

type TestEmailSender interface {
	SendTestEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, recipientUserID string) error
}

// NewAppConfigController creates a new controller for application configuration endpoints
func NewAppConfigController(
	r *authz.Router,
	appConfigService *appconfig.AppConfigService,
	emailSender TestEmailSender,
) {

	acc := &AppConfigController{
		appConfigService: appConfigService,
		emailSender:      emailSender,
	}
	r.Public().GET("/application-configuration", httpserver.JSON(http.StatusOK, acc.listAppConfigHandler),
		option.Summary("List public application configurations"),
		option.Description("Get all public application configurations"),
		option.Tags("Application Configuration"),
	)
	r.GET("/application-configuration/all", authz.ConfigRead, httpserver.JSON(http.StatusOK, acc.listAllAppConfigHandler),
		option.Summary("List all application configurations"),
		option.Description("Get all application configurations including private ones"),
		option.Tags("Application Configuration"),
	)
	updateAppConfigDocs := option.Group(
		option.Summary("Update application configurations"),
		option.Description("Update application configuration settings, where PATCH changes only the fields in the body"),
		option.Tags("Application Configuration"),
	)
	r.PUT("/application-configuration", authz.ConfigWrite, httpserver.UpdateJSON(acc.updateAppConfigHandler), updateAppConfigDocs)
	r.PATCH("/application-configuration", authz.ConfigWrite, httpserver.UpdateJSON(acc.updateAppConfigHandler), updateAppConfigDocs)

	r.POST("/application-configuration/test-email", authz.ConfigWrite, httpserver.NoContent(acc.testEmailHandler),
		option.Summary("Send test email"),
		option.Description("Send a test email to verify email configuration"),
		option.Tags("Application Configuration"),
	)
}

type AppConfigController struct {
	appConfigService *appconfig.AppConfigService
	emailSender      TestEmailSender
}

func (acc *AppConfigController) listAppConfigHandler(c *gin.Context) ([]dto.PublicAppConfigVariableDto, error) {
	configuration, err := acc.appConfigService.ListAppConfig(c.Request.Context(), false)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []dto.PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return nil, err
	}

	// Manually add uiConfigDisabled which isn't in the database but defined with an environment variable
	configVariablesDto = append(configVariablesDto, dto.PublicAppConfigVariableDto{
		Key:   "uiConfigDisabled",
		Value: strconv.FormatBool(common.EnvConfig.UiConfigDisabled),
		Type:  "boolean",
	})

	// Manually add tracingEnabled, derived from the OTel environment, so the frontend only exports traces when the backend can forward them to a collector
	configVariablesDto = append(configVariablesDto, dto.PublicAppConfigVariableDto{
		Key:   "tracingEnabled",
		Value: strconv.FormatBool(tracing.FrontendTracingEnabled()),
	})

	// Manually add iconLibrary, derived from ICON_LIBRARY_URL, so the frontend can hide the icon picker and only credits selfh.st when its collection is used
	configVariablesDto = append(configVariablesDto, dto.PublicAppConfigVariableDto{
		Key:   "iconLibrary",
		Value: iconLibraryStatus(),
	})

	return configVariablesDto, nil
}

// iconLibraryStatus reports whether the icon library is disabled, uses the default selfh.st collection, or points to a custom source
func iconLibraryStatus() string {
	switch common.EnvConfig.IconLibraryURL {
	case common.IconLibraryDisabled:
		return "disabled"
	case common.DefaultIconLibraryURL:
		return "default"
	default:
		return "custom"
	}
}

func (acc *AppConfigController) listAllAppConfigHandler(c *gin.Context) ([]dto.AppConfigVariableDto, error) {
	configuration, err := acc.appConfigService.ListAppConfig(c.Request.Context(), true)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return nil, err
	}

	return configVariablesDto, nil
}

func (acc *AppConfigController) updateAppConfigHandler(c *gin.Context, update dto.Update[dto.AppConfigUpdateDto]) ([]dto.AppConfigVariableDto, error) {
	savedConfigVariables, err := acc.appConfigService.UpdateAppConfig(c.Request.Context(), update)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(savedConfigVariables, &configVariablesDto); err != nil {
		return nil, err
	}

	return configVariablesDto, nil
}

func (acc *AppConfigController) testEmailHandler(c *gin.Context) error {
	dbConfig, err := acc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	// The test email goes to the caller, which a client acting as itself doesn't have
	userID := authz.PrincipalFrom(c).UserID
	if userID == "" {
		return apperror.ClientTokenAuthNotAllowed()
	}

	return acc.emailSender.SendTestEmail(c.Request.Context(), dbConfig, userID)
}
