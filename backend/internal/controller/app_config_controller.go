package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
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

	group.GET("/application-configuration/logo", acc.getLogoHandler)
	group.GET("/application-configuration/background-image", acc.getBackgroundImageHandler)
	group.GET("/application-configuration/favicon", acc.getFaviconHandler)
	group.PUT("/application-configuration/logo", authMiddleware.Add(), acc.updateLogoHandler)
	group.PUT("/application-configuration/favicon", authMiddleware.Add(), acc.updateFaviconHandler)
	group.PUT("/application-configuration/background-image", authMiddleware.Add(), acc.updateBackgroundImageHandler)

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
// @Failure 500 {object} object "{"error": "error message"}"
// @Router /application-configuration [get]
func (acc *AppConfigController) listAppConfigHandler(c *gin.Context) {
	configuration, err := acc.appConfigService.ListAppConfig(false)
	if err != nil {
		c.Error(err)
		return
	}

	var configVariablesDto []dto.PublicAppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, configVariablesDto)
}

// listAllAppConfigHandler godoc
// @Summary List all application configurations
// @Description Get all application configurations including private ones
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Success 200 {array} dto.AppConfigVariableDto
// @Security BearerAuth
// @Router /application-configuration/all [get]
func (acc *AppConfigController) listAllAppConfigHandler(c *gin.Context) {
	configuration, err := acc.appConfigService.ListAppConfig(true)
	if err != nil {
		c.Error(err)
		return
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(configuration, &configVariablesDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, configVariablesDto)
}

// updateAppConfigHandler godoc
// @Summary Update application configurations
// @Description Update application configuration settings
// @Tags Application Configuration
// @Accept json
// @Produce json
// @Param body body dto.AppConfigUpdateDto true "Application Configuration"
// @Success 200 {array} dto.AppConfigVariableDto
// @Security BearerAuth
// @Router /application-configuration [put]
func (acc *AppConfigController) updateAppConfigHandler(c *gin.Context) {
	var input dto.AppConfigUpdateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	savedConfigVariables, err := acc.appConfigService.UpdateAppConfig(input)
	if err != nil {
		c.Error(err)
		return
	}

	var configVariablesDto []dto.AppConfigVariableDto
	if err := dto.MapStructList(savedConfigVariables, &configVariablesDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configVariablesDto)
}

// getLogoHandler godoc
// @Summary Get logo image
// @Description Get the logo image for the application
// @Tags Application Configuration
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Success 200 {file} binary "Logo image"
// @Router /application-configuration/logo [get]
func (acc *AppConfigController) getLogoHandler(c *gin.Context) {
	lightLogo := c.DefaultQuery("light", "true") == "true"

	var imageName string
	var imageType string

	if lightLogo {
		imageName = "logoLight"
		imageType = acc.appConfigService.DbConfig.LogoLightImageType.Value
	} else {
		imageName = "logoDark"
		imageType = acc.appConfigService.DbConfig.LogoDarkImageType.Value
	}

	acc.getImage(c, imageName, imageType)
}

// getFaviconHandler godoc
// @Summary Get favicon
// @Description Get the favicon for the application
// @Tags Application Configuration
// @Produce image/x-icon
// @Success 200 {file} binary "Favicon image"
// @Failure 404 {object} object "{"error": "File not found"}"
// @Router /application-configuration/favicon [get]
func (acc *AppConfigController) getFaviconHandler(c *gin.Context) {
	acc.getImage(c, "favicon", "ico")
}

// getBackgroundImageHandler godoc
// @Summary Get background image
// @Description Get the background image for the application
// @Tags Application Configuration
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Background image"
// @Failure 404 {object} object "{"error": "File not found"}"
// @Router /application-configuration/background-image [get]
func (acc *AppConfigController) getBackgroundImageHandler(c *gin.Context) {
	imageType := acc.appConfigService.DbConfig.BackgroundImageType.Value
	acc.getImage(c, "background", imageType)
}

// updateLogoHandler godoc
// @Summary Update logo
// @Description Update the application logo
// @Tags Application Configuration
// @Accept multipart/form-data
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param file formData file true "Logo image file"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /application-configuration/logo [put]
func (acc *AppConfigController) updateLogoHandler(c *gin.Context) {
	lightLogo := c.DefaultQuery("light", "true") == "true"

	var imageName string
	var imageType string

	if lightLogo {
		imageName = "logoLight"
		imageType = acc.appConfigService.DbConfig.LogoLightImageType.Value
	} else {
		imageName = "logoDark"
		imageType = acc.appConfigService.DbConfig.LogoDarkImageType.Value
	}

	acc.updateImage(c, imageName, imageType)
}

// updateFaviconHandler godoc
// @Summary Update favicon
// @Description Update the application favicon
// @Tags Application Configuration
// @Accept multipart/form-data
// @Param file formData file true "Favicon file (.ico)"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /application-configuration/favicon [put]
func (acc *AppConfigController) updateFaviconHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}

	fileType := utils.GetFileExtension(file.Filename)
	if fileType != "ico" {
		c.Error(&common.WrongFileTypeError{ExpectedFileType: ".ico"})
		return
	}
	acc.updateImage(c, "favicon", "ico")
}

// updateBackgroundImageHandler godoc
// @Summary Update background image
// @Description Update the application background image
// @Tags Application Configuration
// @Accept multipart/form-data
// @Param file formData file true "Background image file"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /application-configuration/background-image [put]
func (acc *AppConfigController) updateBackgroundImageHandler(c *gin.Context) {
	imageType := acc.appConfigService.DbConfig.BackgroundImageType.Value
	acc.updateImage(c, "background", imageType)
}

// getImage is a helper function to serve image files
func (acc *AppConfigController) getImage(c *gin.Context, name string, imageType string) {
	imagePath := fmt.Sprintf("%s/application-images/%s.%s", common.EnvConfig.UploadPath, name, imageType)
	mimeType := utils.GetImageMimeType(imageType)

	c.Header("Content-Type", mimeType)
	c.File(imagePath)
}

// updateImage is a helper function to update image files
func (acc *AppConfigController) updateImage(c *gin.Context, imageName string, oldImageType string) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}

	err = acc.appConfigService.UpdateImage(file, imageName, oldImageType)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// syncLdapHandler godoc
// @Summary Synchronize LDAP
// @Description Manually trigger LDAP synchronization
// @Tags Application Configuration
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /application-configuration/sync-ldap [post]
func (acc *AppConfigController) syncLdapHandler(c *gin.Context) {
	err := acc.ldapService.SyncAll()
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// testEmailHandler godoc
// @Summary Send test email
// @Description Send a test email to verify email configuration
// @Tags Application Configuration
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /application-configuration/test-email [post]
func (acc *AppConfigController) testEmailHandler(c *gin.Context) {
	userID := c.GetString("userID")

	err := acc.emailService.SendTestEmail(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
