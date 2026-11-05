package controller

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	kitutils "github.com/italypaleale/go-kit/utils"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func NewAppAssetsController(
	r *authz.Router,
	fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware,
	appAssetsService *service.AppAssetsService,
) {
	controller := &AppAssetsController{
		appAssetsService: appAssetsService,
	}

	r.Public().GET("/application-assets/logo", httpserver.Handle(controller.getLogoHandler))
	r.Public().GET("/application-assets/email", httpserver.Handle(controller.getEmailLogoHandler))
	r.Public().GET("/application-assets/background", httpserver.Handle(controller.getBackgroundImageHandler))
	r.Public().GET("/application-assets/favicon", httpserver.Handle(controller.getFaviconHandler))
	r.Public().GET("/application-assets/font", httpserver.Handle(controller.getFontHandler))
	r.GET("/application-assets/default-profile-picture", authz.ConfigRead, httpserver.Handle(controller.getDefaultProfilePicture))

	r.PUT("/application-assets/logo", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateLogoHandler))
	r.PUT("/application-assets/email", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateEmailLogoHandler))
	r.PUT("/application-assets/background", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateBackgroundImageHandler))
	r.PUT("/application-assets/favicon", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateFaviconHandler))
	r.PUT("/application-assets/default-profile-picture", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateDefaultProfilePicture))
	r.PUT("/application-assets/font", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateFontHandler))

	r.DELETE("/application-assets/logo", authz.ConfigWrite, httpserver.Handle(controller.deleteLogoHandler))
	r.DELETE("/application-assets/background", authz.ConfigWrite, httpserver.Handle(controller.deleteBackgroundImageHandler))
	r.DELETE("/application-assets/default-profile-picture", authz.ConfigWrite, httpserver.Handle(controller.deleteDefaultProfilePicture))
	r.DELETE("/application-assets/font", authz.ConfigWrite, httpserver.Handle(controller.deleteFontHandler))
}

type AppAssetsController struct {
	appAssetsService *service.AppAssetsService
}

// getLogoHandler godoc
// @Summary Get logo image
// @Description Get the logo image for the application
// @Tags Application Assets
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param default query boolean false "Return the bundled default logo if no custom logo is set (default true)"
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Success 200 {file} binary "Logo image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/logo [get]
func (c *AppAssetsController) getLogoHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, logoImageName(ctx))
}

// getEmailLogoHandler godoc
// @Summary Get email logo image
// @Description Get the email logo image for use in emails
// @Tags Application Assets
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Email logo image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/email [get]
func (c *AppAssetsController) getEmailLogoHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "logoEmail")
}

// getBackgroundImageHandler godoc
// @Summary Get background image
// @Description Get the background image for the application
// @Tags Application Assets
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Background image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/background [get]
func (c *AppAssetsController) getBackgroundImageHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "background")
}

// getFaviconHandler godoc
// @Summary Get favicon
// @Description Get the favicon for the application
// @Tags Application Assets
// @Produce image/x-icon
// @Success 200 {file} binary "Favicon image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/favicon [get]
func (c *AppAssetsController) getFaviconHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "favicon")
}

// getDefaultProfilePicture godoc
// @Summary Get default profile picture image
// @Description Get the default profile picture image for the application
// @Tags Application Assets
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Default profile picture image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/default-profile-picture [get]
func (c *AppAssetsController) getDefaultProfilePicture(ctx *gin.Context) error {
	return c.getAsset(ctx, "default-profile-picture")
}

// updateLogoHandler godoc
// @Summary Update logo
// @Description Update the application logo
// @Tags Application Assets
// @Accept multipart/form-data
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param file formData file true "Logo image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/logo [put]
func (c *AppAssetsController) updateLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, logoImageName(ctx)); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteLogoHandler godoc
// @Summary Delete logo image
// @Description Delete the custom application logo and restore the default logo
// @Tags Application Assets
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/logo [delete]
func (c *AppAssetsController) deleteLogoHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), logoImageName(ctx)); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

func logoImageName(ctx *gin.Context) string {
	lightLogo := kitutils.IsTruthy(ctx.DefaultQuery("light", "true"))
	if lightLogo {
		return "logoLight"
	}
	return "logoDark"
}

// updateEmailLogoHandler godoc
// @Summary Update email logo
// @Description Update the email logo for use in emails
// @Tags Application Assets
// @Accept multipart/form-data
// @Param file formData file true "Email logo image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/email [put]
func (c *AppAssetsController) updateEmailLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(fileType)

	if mimeType != "image/png" && mimeType != "image/jpeg" {
		return apperror.UnsupportedFileType("PNG or JPEG")
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "logoEmail"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// updateBackgroundImageHandler godoc
// @Summary Update background image
// @Description Update the application background image
// @Tags Application Assets
// @Accept multipart/form-data
// @Param file formData file true "Background image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/background [put]
func (c *AppAssetsController) updateBackgroundImageHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "background"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteBackgroundImageHandler godoc
// @Summary Delete background image
// @Description Delete the application background image
// @Tags Application Assets
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/background [delete]
func (c *AppAssetsController) deleteBackgroundImageHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "background"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// updateFaviconHandler godoc
// @Summary Update favicon
// @Description Update the application favicon
// @Tags Application Assets
// @Accept multipart/form-data
// @Param file formData file true "Favicon file (.svg/.png/.ico)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/favicon [put]
func (c *AppAssetsController) updateFaviconHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(strings.ToLower(fileType))
	if !slices.Contains([]string{"image/svg+xml", "image/png", "image/x-icon"}, mimeType) {
		return apperror.UnsupportedFileType("SVG, PNG, or ICO")
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "favicon"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// getFontHandler godoc
// @Summary Get title font
// @Description Get the custom font of page and dialog titles
// @Tags Application Assets
// @Produce font/woff2
// @Produce font/woff
// @Produce font/ttf
// @Produce font/otf
// @Success 200 {file} binary "Font file"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/font [get]
func (c *AppAssetsController) getFontHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "font")
}

// updateFontHandler godoc
// @Summary Update title font
// @Description Update the custom font of page and dialog titles
// @Tags Application Assets
// @Accept multipart/form-data
// @Param file formData file true "Font file (.woff2/.woff/.ttf/.otf)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/font [put]
func (c *AppAssetsController) updateFontHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "font"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteFontHandler godoc
// @Summary Delete title font
// @Description Delete the custom font of page and dialog titles
// @Tags Application Assets
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/font [delete]
func (c *AppAssetsController) deleteFontHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "font"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

func (c *AppAssetsController) getAsset(ctx *gin.Context, name string) error {
	getAsset := c.appAssetsService.GetAsset
	if kitutils.IsTruthy(ctx.DefaultQuery("default", "true")) {
		getAsset = c.appAssetsService.GetAssetWithDefault
	}

	reader, size, mimeType, err := getAsset(ctx.Request.Context(), name)
	if err != nil {
		return err
	}
	defer reader.Close()

	ctx.Header("Content-Type", mimeType)
	utils.SetCacheControlHeader(ctx, 15*time.Minute, 24*time.Hour)
	ctx.DataFromReader(http.StatusOK, size, mimeType, reader, nil)
	return nil
}

// updateDefaultProfilePicture godoc
// @Summary Update default profile picture image
// @Description Update the default profile picture image
// @Tags Application Assets
// @Accept multipart/form-data
// @Param file formData file true "Profile picture image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/default-profile-picture [put]
func (c *AppAssetsController) updateDefaultProfilePicture(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "default-profile-picture"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteDefaultProfilePicture godoc
// @Summary Delete default profile picture image
// @Description Delete the default profile picture image
// @Tags Application Assets
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-assets/default-profile-picture [delete]
func (c *AppAssetsController) deleteDefaultProfilePicture(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "default-profile-picture"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}
