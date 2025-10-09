package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func NewAppImagesController(
	group *gin.RouterGroup,
	authMiddleware *middleware.AuthMiddleware,
	appImagesService *service.AppImagesService,
) {
	controller := &AppImagesController{
		appImagesService: appImagesService,
	}

	group.GET("/application-images/logo", controller.getLogoHandler)
	group.GET("/application-images/background", controller.getBackgroundImageHandler)
	group.GET("/application-images/favicon", controller.getFaviconHandler)

	group.PUT("/application-images/logo", authMiddleware.Add(), controller.updateLogoHandler)
	group.PUT("/application-images/background", authMiddleware.Add(), controller.updateBackgroundImageHandler)
	group.PUT("/application-images/favicon", authMiddleware.Add(), controller.updateFaviconHandler)
}

type AppImagesController struct {
	appImagesService *service.AppImagesService
}

// getLogoHandler godoc
// @Summary Get logo image
// @Description Get the logo image for the application
// @Tags Application Images
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Success 200 {file} binary "Logo image"
// @Router /api/application-images/logo [get]
func (c *AppImagesController) getLogoHandler(ctx *gin.Context) {
	lightLogo, _ := strconv.ParseBool(ctx.DefaultQuery("light", "true"))
	imageName := "logoLight"
	if !lightLogo {
		imageName = "logoDark"
	}

	c.getImage(ctx, imageName)
}

// getBackgroundImageHandler godoc
// @Summary Get background image
// @Description Get the background image for the application
// @Tags Application Images
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Background image"
// @Router /api/application-images/background [get]
func (c *AppImagesController) getBackgroundImageHandler(ctx *gin.Context) {
	c.getImage(ctx, "background")
}

// getFaviconHandler godoc
// @Summary Get favicon
// @Description Get the favicon for the application
// @Tags Application Images
// @Produce image/x-icon
// @Success 200 {file} binary "Favicon image"
// @Router /api/application-images/favicon [get]
func (c *AppImagesController) getFaviconHandler(ctx *gin.Context) {
	c.getImage(ctx, "favicon")
}

// updateLogoHandler godoc
// @Summary Update logo
// @Description Update the application logo
// @Tags Application Images
// @Accept multipart/form-data
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param file formData file true "Logo image file"
// @Success 204 "No Content"
// @Router /api/application-images/logo [put]
func (c *AppImagesController) updateLogoHandler(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	lightLogo, _ := strconv.ParseBool(ctx.DefaultQuery("light", "true"))
	imageName := "logoLight"
	if !lightLogo {
		imageName = "logoDark"
	}

	if err := c.appImagesService.UpdateImage(file, imageName); err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// updateBackgroundImageHandler godoc
// @Summary Update background image
// @Description Update the application background image
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Background image file"
// @Success 204 "No Content"
// @Router /api/application-images/background [put]
func (c *AppImagesController) updateBackgroundImageHandler(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	if err := c.appImagesService.UpdateImage(file, "background"); err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// updateFaviconHandler godoc
// @Summary Update favicon
// @Description Update the application favicon
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Favicon file (.ico)"
// @Success 204 "No Content"
// @Router /api/application-images/favicon [put]
func (c *AppImagesController) updateFaviconHandler(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	fileType := utils.GetFileExtension(file.Filename)
	if fileType != "ico" {
		_ = ctx.Error(&common.WrongFileTypeError{ExpectedFileType: ".ico"})
		return
	}

	if err := c.appImagesService.UpdateImage(file, "favicon"); err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *AppImagesController) getImage(ctx *gin.Context, name string) {
	imagePath, mimeType, err := c.appImagesService.GetImage(name)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.Header("Content-Type", mimeType)
	utils.SetCacheControlHeader(ctx, 15*time.Minute, 24*time.Hour)
	ctx.File(imagePath)
}
