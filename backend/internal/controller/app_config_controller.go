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

func NewAppConfigController(
	group *gin.RouterGroup,
	jwtAuthMiddleware *middleware.JwtAuthMiddleware,
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
	group.GET("/application-configuration/all", jwtAuthMiddleware.Add(true), acc.listAllAppConfigHandler)
	group.PUT("/application-configuration", jwtAuthMiddleware.Add(true), acc.updateAppConfigHandler)

	group.GET("/application-configuration/logo", acc.getLogoHandler)
	group.GET("/application-configuration/background-image", acc.getBackgroundImageHandler)
	group.GET("/application-configuration/favicon", acc.getFaviconHandler)
	group.PUT("/application-configuration/logo", jwtAuthMiddleware.Add(true), acc.updateLogoHandler)
	group.PUT("/application-configuration/favicon", jwtAuthMiddleware.Add(true), acc.updateFaviconHandler)
	group.PUT("/application-configuration/background-image", jwtAuthMiddleware.Add(true), acc.updateBackgroundImageHandler)

	group.POST("/application-configuration/test-email", jwtAuthMiddleware.Add(true), acc.testEmailHandler)
	group.POST("/application-configuration/sync-ldap", jwtAuthMiddleware.Add(true), acc.syncLdapHandler)
}

type AppConfigController struct {
	appConfigService *service.AppConfigService
	emailService     *service.EmailService
	ldapService      *service.LdapService
}

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

func (acc *AppConfigController) getFaviconHandler(c *gin.Context) {
	acc.getImage(c, "favicon", "ico")
}

func (acc *AppConfigController) getBackgroundImageHandler(c *gin.Context) {
	imageType := acc.appConfigService.DbConfig.BackgroundImageType.Value
	acc.getImage(c, "background", imageType)
}

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

func (acc *AppConfigController) updateBackgroundImageHandler(c *gin.Context) {
	imageType := acc.appConfigService.DbConfig.BackgroundImageType.Value
	acc.updateImage(c, "background", imageType)
}

func (acc *AppConfigController) getImage(c *gin.Context, name string, imageType string) {
	imagePath := fmt.Sprintf("%s/application-images/%s.%s", common.EnvConfig.UploadPath, name, imageType)
	mimeType := utils.GetImageMimeType(imageType)

	c.Header("Content-Type", mimeType)
	c.File(imagePath)
}

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

func (acc *AppConfigController) syncLdapHandler(c *gin.Context) {
	err := acc.ldapService.SyncAll()
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
func (acc *AppConfigController) testEmailHandler(c *gin.Context) {
	userID := c.GetString("userID")

	err := acc.emailService.SendTestEmail(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
