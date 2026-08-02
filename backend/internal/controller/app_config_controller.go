package controller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
)

type TestEmailSender interface {
	SendTestEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, recipientUserID string) error
}

// NewAppConfigController creates a new controller for application configuration endpoints
// @Summary Create a new application configuration controller
// @Description Initialize routes for application configuration
// @Tags Application Configuration
func NewAppConfigController(
	group *gin.RouterGroup,
	authMiddleware *middleware.AuthMiddleware,
	appConfigService *appconfig.AppConfigService,
	emailSender TestEmailSender,
	ldapService *service.LdapService,
) {

	acc := &AppConfigController{
		appConfigService: appConfigService,
		emailSender:      emailSender,
		ldapService:      ldapService,
	}
	group.GET("/application-configuration", httpserver.Handle(acc.listAppConfigHandler))
	group.GET("/application-configuration/all", authMiddleware.Add(), httpserver.Handle(acc.listAllAppConfigHandler))
	group.PUT("/application-configuration", authMiddleware.Add(), httpserver.Handle(acc.updateAppConfigHandler))

	group.POST("/application-configuration/test-email", authMiddleware.Add(), httpserver.Handle(acc.testEmailHandler))
	group.POST("/application-configuration/sync-ldap", authMiddleware.Add(), httpserver.Handle(acc.syncLdapHandler))
}

type AppConfigController struct {
	appConfigService *appconfig.AppConfigService
	emailSender      TestEmailSender
	ldapService      *service.LdapService
}

// listAppConfigHandler godoc
// @Summary List public application configurations
// @Description Get all public application configurations
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Success 200 {array} dto.PublicAppConfigVariableDto
// @Router /api/application-configuration [get]
func (acc *AppConfigController) listAppConfigHandler(c *gin.Context) error {
	dbConfig, err := acc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}
	configuration := dbConfig.ToAppConfigVariableSlice(false, true)

	var configVariablesDto []dto.PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return err
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

	c.JSON(http.StatusOK, configVariablesDto)
	return nil
}

// listAllAppConfigHandler godoc
// @Summary List all application configurations
// @Description Get all application configurations including private ones
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Success 200 {array} dto.AppConfigVariableDto
// @Router /api/application-configuration/all [get]
func (acc *AppConfigController) listAllAppConfigHandler(c *gin.Context) error {
	dbConfig, err := acc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}
	configuration := dbConfig.ToAppConfigVariableSlice(true, true)

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, configVariablesDto)
	return nil
}

// updateAppConfigHandler godoc
// @Summary Update application configurations
// @Description Update application configuration settings
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Param body body dto.AppConfigUpdateDto true "Application Configuration"
// @Success 200 {array} dto.AppConfigVariableDto
// @Router /api/application-configuration [put]
func (acc *AppConfigController) updateAppConfigHandler(c *gin.Context) error {
	var input dto.AppConfigUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	savedConfigVariables, err := acc.appConfigService.UpdateAppConfig(c.Request.Context(), input)
	if err != nil {
		return err
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(savedConfigVariables, &configVariablesDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, configVariablesDto)
	return nil
}

// syncLdapHandler godoc
// @Summary Synchronize LDAP
// @Description Manually trigger LDAP synchronization
// @Tags Application Configuration
// @Success 204 "No Content"
// @Router /api/application-configuration/sync-ldap [post]
func (acc *AppConfigController) syncLdapHandler(c *gin.Context) error {
	dbConfig, err := acc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	err = acc.ldapService.SyncAll(c.Request.Context(), dbConfig)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// testEmailHandler godoc
// @Summary Send test email
// @Description Send a test email to verify email configuration
// @Tags Application Configuration
// @Success 204 "No Content"
// @Router /api/application-configuration/test-email [post]
func (acc *AppConfigController) testEmailHandler(c *gin.Context) error {
	dbConfig, err := acc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	userID := c.GetString("userID")

	err = acc.emailSender.SendTestEmail(c.Request.Context(), dbConfig, userID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
