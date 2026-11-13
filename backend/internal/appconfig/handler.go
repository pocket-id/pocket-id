package appconfig

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
)

// TestEmailSender sends the test email that checks the SMTP settings
type TestEmailSender interface {
	SendTestEmail(ctx context.Context, dbConfig *AppConfigModel, recipientUserID string) error
}

// RegisterRoutes mounts the application configuration endpoints and, on base, the web app manifest
func (s *AppConfigService) RegisterRoutes(r *authz.Router, base *gin.RouterGroup, emailSender TestEmailSender) {
	acc := &handler{service: s, emailSender: emailSender}

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

	// The manifest carries the configured application name when the app is added to a home screen
	base.GET("/app.webmanifest", httpserver.Handle(acc.webManifestHandler).Handler)
}

type handler struct {
	service     *AppConfigService
	emailSender TestEmailSender
}

func (acc *handler) listAppConfigHandler(c *gin.Context) ([]PublicAppConfigVariableDto, error) {
	configuration, err := acc.service.ListAppConfig(c.Request.Context(), false)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return nil, err
	}

	// Manually add uiConfigDisabled which isn't in the database but defined with an environment variable
	configVariablesDto = append(configVariablesDto, PublicAppConfigVariableDto{
		Key:   "uiConfigDisabled",
		Value: strconv.FormatBool(common.EnvConfig.UiConfigDisabled),
		Type:  "boolean",
	})

	// Manually add tracingEnabled, derived from the OTel environment, so the frontend only exports traces when the backend can forward them to a collector
	configVariablesDto = append(configVariablesDto, PublicAppConfigVariableDto{
		Key:   "tracingEnabled",
		Value: strconv.FormatBool(tracing.FrontendTracingEnabled()),
	})

	// Manually add iconLibrary, derived from ICON_LIBRARY_URL, so the frontend can hide the icon picker and only credits selfh.st when its collection is used
	configVariablesDto = append(configVariablesDto, PublicAppConfigVariableDto{
		Key:   "iconLibrary",
		Value: iconLibraryStatus(),
	})

	return configVariablesDto, nil
}

// iconLibraryStatus reports whether the icon library is disabled, uses the default selfh.st collection, or points to a custom source
func iconLibraryStatus() string {
	switch common.EnvConfig.IconLibraryURL {
	case common.URLDisabled:
		return "disabled"
	case common.DefaultIconLibraryURL:
		return "default"
	default:
		return "custom"
	}
}

func (acc *handler) listAllAppConfigHandler(c *gin.Context) ([]AppConfigVariableDto, error) {
	configuration, err := acc.service.ListAppConfig(c.Request.Context(), true)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []AppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return nil, err
	}

	return configVariablesDto, nil
}

func (acc *handler) updateAppConfigHandler(c *gin.Context, update dto.Update[AppConfigUpdateDto]) ([]AppConfigVariableDto, error) {
	savedConfigVariables, err := acc.service.UpdateAppConfig(c.Request.Context(), update)
	if err != nil {
		return nil, err
	}

	var configVariablesDto []AppConfigVariableDto
	if err := dto.MapStructList(savedConfigVariables, &configVariablesDto); err != nil {
		return nil, err
	}

	return configVariablesDto, nil
}

func (acc *handler) testEmailHandler(c *gin.Context) error {
	dbConfig, err := acc.service.GetConfig(c.Request.Context())
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

func (acc *handler) webManifestHandler(c *gin.Context) error {
	cfg, err := acc.service.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	manifest, err := json.Marshal(map[string]any{
		"name":        cfg.AppName.String(),
		"orientation": "any",
		"display":     "minimal-ui",
		"dir":         "auto",
		"start_url":   "/",
		"icons": []map[string]string{
			{"src": "img/static-logo-rounded-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
			{"src": "img/static-logo-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
			{"src": "img/static-logo.svg", "sizes": "any", "type": "image/svg+xml", "purpose": "maskable monochrome"},
		},
	})
	if err != nil {
		return err
	}

	c.Data(http.StatusOK, "application/manifest+json", manifest)
	return nil
}
