package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewAppConfigController creates a new controller for application configuration endpoints
// @Summary Create a new application configuration controller
// @Description Initialize routes for application configuration
// @Tags Application Configuration
func NewAppConfigController(
	group *gin.RouterGroup,
	authMiddleware *middleware.AuthMiddleware,
	appConfigService *service.AppConfigService,
	emailService *service.EmailService,
	ldapService *service.LdapService,
) {

	acc := &AppConfigController{
		appConfigService: appConfigService,
		emailService:     emailService,
		ldapService:      ldapService,
	}
	group.GET("/application-configuration", acc.listAppConfigHandler)
	group.GET("/application-configuration/all", authMiddleware.Add(), acc.listAllAppConfigHandler)
	group.PUT("/application-configuration", authMiddleware.Add(), acc.updateAppConfigHandler)

	group.POST("/application-configuration/test-email", authMiddleware.Add(), acc.testEmailHandler)
	group.POST("/application-configuration/sync-ldap", authMiddleware.Add(), acc.syncLdapHandler)
}

type AppConfigController struct {
	appConfigService *service.AppConfigService
	emailService     *service.EmailService
	ldapService      *service.LdapService
}

// listAppConfigHandler godoc
// @Summary List public application configurations
// @Description Get all public application configurations
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Success 200 {array} dto.PublicAppConfigVariableDto
// @Router /application-configuration [get]
func (acc *AppConfigController) listAppConfigHandler(c *gin.Context) {
	configuration := acc.appConfigService.ListAppConfig(false)

	var configVariablesDto []dto.PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		_ = c.Error(err)
		return
	}

	// Manually add uiConfigDisabled which isn't in the database but defined with an environment variable
	configVariablesDto = append(configVariablesDto, dto.PublicAppConfigVariableDto{
		Key:   "uiConfigDisabled",
		Value: strconv.FormatBool(common.EnvConfig.UiConfigDisabled),
		Type:  "boolean",
	})

	c.JSON(http.StatusOK, configVariablesDto)
}

// listAllAppConfigHandler godoc
// @Summary List all application configurations
// @Description Get all application configurations including private ones
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Success 200 {array} dto.AppConfigVariableDto
// @Router /application-configuration/all [get]
func (acc *AppConfigController) listAllAppConfigHandler(c *gin.Context) {
	configuration := acc.appConfigService.ListAppConfig(true)

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configVariablesDto)
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
func (acc *AppConfigController) updateAppConfigHandler(c *gin.Context) {
	var input dto.AppConfigUpdateDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	savedConfigVariables, err := acc.appConfigService.UpdateAppConfig(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(savedConfigVariables, &configVariablesDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configVariablesDto)
}

// syncLdapHandler godoc
// @Summary Synchronize LDAP
// @Description Manually trigger LDAP synchronization
// @Tags Application Configuration
// @Success 204 "No Content"
// @Router /api/application-configuration/sync-ldap [post]
func (acc *AppConfigController) syncLdapHandler(c *gin.Context) {
	err := acc.ldapService.SyncAll(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// testEmailHandler godoc
// @Summary Send test email
// @Description Send a test email to verify email configuration
// @Tags Application Configuration
// @Success 204 "No Content"
// @Router /api/application-configuration/test-email [post]
func (acc *AppConfigController) testEmailHandler(c *gin.Context) {
	userID := c.GetString("userID")

	err := acc.emailService.SendTestEmail(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
