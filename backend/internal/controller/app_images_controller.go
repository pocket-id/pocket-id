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

func NewAppImagesController(
	r *authz.Router,
	fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware,
	appImagesService *service.AppImagesService,
) {
	controller := &AppImagesController{
		appImagesService: appImagesService,
	}

	r.Public().GET("/application-images/logo", httpserver.Handle(controller.getLogoHandler))
	r.Public().GET("/application-images/email", httpserver.Handle(controller.getEmailLogoHandler))
	r.Public().GET("/application-images/background", httpserver.Handle(controller.getBackgroundImageHandler))
	r.Public().GET("/application-images/favicon", httpserver.Handle(controller.getFaviconHandler))
	r.GET("/application-images/default-profile-picture", authz.ConfigRead, httpserver.Handle(controller.getDefaultProfilePicture))

	r.PUT("/application-images/logo", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateLogoHandler))
	r.PUT("/application-images/email", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateEmailLogoHandler))
	r.PUT("/application-images/background", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateBackgroundImageHandler))
	r.PUT("/application-images/favicon", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateFaviconHandler))
	r.PUT("/application-images/default-profile-picture", authz.ConfigWrite, fileSizeLimitMiddleware.Add(10<<20), httpserver.Handle(controller.updateDefaultProfilePicture))

	r.DELETE("/application-images/logo", authz.ConfigWrite, httpserver.Handle(controller.deleteLogoHandler))
	r.DELETE("/application-images/background", authz.ConfigWrite, httpserver.Handle(controller.deleteBackgroundImageHandler))
	r.DELETE("/application-images/default-profile-picture", authz.ConfigWrite, httpserver.Handle(controller.deleteDefaultProfilePicture))
}

type AppImagesController struct {
	appImagesService *service.AppImagesService
}

// getLogoHandler godoc
// @Summary Get logo image
// @Description Get the logo image for the application
// @Tags Application Images
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param default query boolean false "Return the bundled default logo if no custom logo is set (default true)"
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Success 200 {file} binary "Logo image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/logo [get]
func (c *AppImagesController) getLogoHandler(ctx *gin.Context) error {
	return c.getImage(ctx, logoImageName(ctx))
}

// getEmailLogoHandler godoc
// @Summary Get email logo image
// @Description Get the email logo image for use in emails
// @Tags Application Images
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Email logo image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/email [get]
func (c *AppImagesController) getEmailLogoHandler(ctx *gin.Context) error {
	return c.getImage(ctx, "logoEmail")
}

// getBackgroundImageHandler godoc
// @Summary Get background image
// @Description Get the background image for the application
// @Tags Application Images
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Background image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/background [get]
func (c *AppImagesController) getBackgroundImageHandler(ctx *gin.Context) error {
	return c.getImage(ctx, "background")
}

// getFaviconHandler godoc
// @Summary Get favicon
// @Description Get the favicon for the application
// @Tags Application Images
// @Produce image/x-icon
// @Success 200 {file} binary "Favicon image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/favicon [get]
func (c *AppImagesController) getFaviconHandler(ctx *gin.Context) error {
	return c.getImage(ctx, "favicon")
}

// getDefaultProfilePicture godoc
// @Summary Get default profile picture image
// @Description Get the default profile picture image for the application
// @Tags Application Images
// @Produce image/png
// @Produce image/jpeg
// @Success 200 {file} binary "Default profile picture image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/default-profile-picture [get]
func (c *AppImagesController) getDefaultProfilePicture(ctx *gin.Context) error {
	return c.getImage(ctx, "default-profile-picture")
}

// updateLogoHandler godoc
// @Summary Update logo
// @Description Update the application logo
// @Tags Application Images
// @Accept multipart/form-data
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Param file formData file true "Logo image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/logo [put]
func (c *AppImagesController) updateLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appImagesService.UpdateImage(ctx.Request.Context(), file, logoImageName(ctx)); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteLogoHandler godoc
// @Summary Delete logo image
// @Description Delete the custom application logo and restore the default logo
// @Tags Application Images
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/logo [delete]
func (c *AppImagesController) deleteLogoHandler(ctx *gin.Context) error {
	if err := c.appImagesService.DeleteImage(ctx.Request.Context(), logoImageName(ctx)); err != nil {
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
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Email logo image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/email [put]
func (c *AppImagesController) updateEmailLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(fileType)

	if mimeType != "image/png" && mimeType != "image/jpeg" {
		return apperror.UnsupportedFileType("PNG or JPEG")
	}

	if err := c.appImagesService.UpdateImage(ctx.Request.Context(), file, "logoEmail"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// updateBackgroundImageHandler godoc
// @Summary Update background image
// @Description Update the application background image
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Background image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/background [put]
func (c *AppImagesController) updateBackgroundImageHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appImagesService.UpdateImage(ctx.Request.Context(), file, "background"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteBackgroundImageHandler godoc
// @Summary Delete background image
// @Description Delete the application background image
// @Tags Application Images
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/background [delete]
func (c *AppImagesController) deleteBackgroundImageHandler(ctx *gin.Context) error {
	if err := c.appImagesService.DeleteImage(ctx.Request.Context(), "background"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// updateFaviconHandler godoc
// @Summary Update favicon
// @Description Update the application favicon
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Favicon file (.svg/.png/.ico)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/favicon [put]
func (c *AppImagesController) updateFaviconHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(strings.ToLower(fileType))
	if !slices.Contains([]string{"image/svg+xml", "image/png", "image/x-icon"}, mimeType) {
		return apperror.UnsupportedFileType("SVG, PNG, or ICO")
	}

	if err := c.appImagesService.UpdateImage(ctx.Request.Context(), file, "favicon"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

func (c *AppImagesController) getImage(ctx *gin.Context, name string) error {
	getImage := c.appImagesService.GetImage
	if kitutils.IsTruthy(ctx.DefaultQuery("default", "true")) {
		getImage = c.appImagesService.GetImageWithDefault
	}

	reader, size, mimeType, err := getImage(ctx.Request.Context(), name)
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
// @Tags Application Images
// @Accept multipart/form-data
// @Param file formData file true "Profile picture image file"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/default-profile-picture [put]
func (c *AppImagesController) updateDefaultProfilePicture(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appImagesService.UpdateImage(ctx.Request.Context(), file, "default-profile-picture"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// deleteDefaultProfilePicture godoc
// @Summary Delete default profile picture image
// @Description Delete the default profile picture image
// @Tags Application Images
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/application-images/default-profile-picture [delete]
func (c *AppImagesController) deleteDefaultProfilePicture(ctx *gin.Context) error {
	if err := c.appImagesService.DeleteImage(ctx.Request.Context(), "default-profile-picture"); err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}
