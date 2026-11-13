package controller

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"
	kitutils "github.com/italypaleale/go-kit/utils"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// imageTypes are the types of every image that utils.GetImageMimeType accepts, which the logo, background and default profile picture can have
var imageTypes = []string{"image/png", "image/jpeg", "image/svg+xml", "image/x-icon", "image/gif", "image/webp", "image/avif", "image/heic"}

func NewAppAssetsController(
	r *authz.Router,
	fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware,
	appAssetsService *service.AppAssetsService,
) {
	controller := &AppAssetsController{
		appAssetsService: appAssetsService,
	}

	// Every asset GET falls back to the bundled default unless asked not to
	fallback := option.QueryBool("default", "Return the bundled default if no custom one is set", param.Default(true))

	r.Public().GET("/application-assets/logo", httpserver.Handle(controller.getLogoHandler),
		option.Summary("Get logo image"),
		option.Description("Get the logo image for the application"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
		fallback,
		openapi.File("Logo image", imageTypes...),
	)
	r.Public().GET("/application-assets/email", httpserver.Handle(controller.getEmailLogoHandler),
		option.Summary("Get email logo image"),
		option.Description("Get the email logo image for use in emails"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Email logo image", "image/png", "image/jpeg"),
	)
	r.Public().GET("/application-assets/background", httpserver.Handle(controller.getBackgroundImageHandler),
		option.Summary("Get background image"),
		option.Description("Get the background image for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Background image", imageTypes...),
	)
	r.Public().GET("/application-assets/favicon", httpserver.Handle(controller.getFaviconHandler),
		option.Summary("Get favicon"),
		option.Description("Get the favicon for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Favicon image", "image/svg+xml", "image/png", "image/x-icon"),
	)
	r.Public().GET("/application-assets/font", httpserver.Handle(controller.getFontHandler),
		option.Summary("Get title font"),
		option.Description("Get the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Font file", "font/woff2", "font/woff", "font/ttf", "font/otf"),
	)
	r.GET("/application-assets/default-profile-picture", authz.ConfigRead, httpserver.Handle(controller.getDefaultProfilePicture),
		option.Summary("Get default profile picture image"),
		option.Description("Get the default profile picture image for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Default profile picture image", imageTypes...),
	)

	uploads := r.With(fileSizeLimitMiddleware.Add(10 << 20))
	uploads.PUT("/application-assets/logo", authz.ConfigWrite, httpserver.NoContent(controller.updateLogoHandler),
		option.Summary("Update logo"),
		option.Description("Update the application logo"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
		openapi.Upload("Logo image file"),
	)
	uploads.PUT("/application-assets/email", authz.ConfigWrite, httpserver.NoContent(controller.updateEmailLogoHandler),
		option.Summary("Update email logo"),
		option.Description("Update the email logo for use in emails"),
		option.Tags("Application Assets"),
		openapi.Upload("Email logo image file"),
	)
	uploads.PUT("/application-assets/background", authz.ConfigWrite, httpserver.NoContent(controller.updateBackgroundImageHandler),
		option.Summary("Update background image"),
		option.Description("Update the application background image"),
		option.Tags("Application Assets"),
		openapi.Upload("Background image file"),
	)
	uploads.PUT("/application-assets/favicon", authz.ConfigWrite, httpserver.NoContent(controller.updateFaviconHandler),
		option.Summary("Update favicon"),
		option.Description("Update the application favicon"),
		option.Tags("Application Assets"),
		openapi.Upload("Favicon file (.svg/.png/.ico)"),
	)
	uploads.PUT("/application-assets/default-profile-picture", authz.ConfigWrite, httpserver.NoContent(controller.updateDefaultProfilePicture),
		option.Summary("Update default profile picture image"),
		option.Description("Update the default profile picture image"),
		option.Tags("Application Assets"),
		openapi.Upload("Profile picture image file"),
	)
	uploads.PUT("/application-assets/font", authz.ConfigWrite, httpserver.NoContent(controller.updateFontHandler),
		option.Summary("Update title font"),
		option.Description("Update the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
		openapi.Upload("Font file (.woff2/.woff/.ttf/.otf)"),
	)

	r.DELETE("/application-assets/logo", authz.ConfigWrite, httpserver.NoContent(controller.deleteLogoHandler),
		option.Summary("Delete logo image"),
		option.Description("Delete the custom application logo and restore the default logo"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
	)
	r.DELETE("/application-assets/background", authz.ConfigWrite, httpserver.NoContent(controller.deleteBackgroundImageHandler),
		option.Summary("Delete background image"),
		option.Description("Delete the application background image"),
		option.Tags("Application Assets"),
	)
	r.DELETE("/application-assets/default-profile-picture", authz.ConfigWrite, httpserver.NoContent(controller.deleteDefaultProfilePicture),
		option.Summary("Delete default profile picture image"),
		option.Description("Delete the default profile picture image"),
		option.Tags("Application Assets"),
	)
	r.DELETE("/application-assets/font", authz.ConfigWrite, httpserver.NoContent(controller.deleteFontHandler),
		option.Summary("Delete title font"),
		option.Description("Delete the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
	)
}

type AppAssetsController struct {
	appAssetsService *service.AppAssetsService
}

func (c *AppAssetsController) getLogoHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, logoImageName(ctx))
}

func (c *AppAssetsController) getEmailLogoHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "logoEmail")
}

func (c *AppAssetsController) getBackgroundImageHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "background")
}

func (c *AppAssetsController) getFaviconHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "favicon")
}

func (c *AppAssetsController) getDefaultProfilePicture(ctx *gin.Context) error {
	return c.getAsset(ctx, "default-profile-picture")
}

func (c *AppAssetsController) updateLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, logoImageName(ctx)); err != nil {
		return err
	}

	return nil
}

func (c *AppAssetsController) deleteLogoHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), logoImageName(ctx)); err != nil {
		return err
	}

	return nil
}

func logoImageName(ctx *gin.Context) string {
	lightLogo := kitutils.IsTruthy(ctx.DefaultQuery("light", "true"))
	if lightLogo {
		return "logoLight"
	}
	return "logoDark"
}

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

	return nil
}

func (c *AppAssetsController) updateBackgroundImageHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "background"); err != nil {
		return err
	}

	return nil
}

func (c *AppAssetsController) deleteBackgroundImageHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "background"); err != nil {
		return err
	}

	return nil
}

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

	return nil
}

func (c *AppAssetsController) getFontHandler(ctx *gin.Context) error {
	return c.getAsset(ctx, "font")
}

func (c *AppAssetsController) updateFontHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "font"); err != nil {
		return err
	}

	return nil
}

func (c *AppAssetsController) deleteFontHandler(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "font"); err != nil {
		return err
	}

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
	// An uploaded SVG opened directly renders as a document, so a second policy sandboxes it to keep it from running scripts
	ctx.Writer.Header().Add("Content-Security-Policy", "sandbox")
	utils.SetCacheControlHeader(ctx, 15*time.Minute, 24*time.Hour)
	ctx.DataFromReader(http.StatusOK, size, mimeType, reader, nil)
	return nil
}

func (c *AppAssetsController) updateDefaultProfilePicture(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := c.appAssetsService.UpdateAsset(ctx.Request.Context(), file, "default-profile-picture"); err != nil {
		return err
	}

	return nil
}

func (c *AppAssetsController) deleteDefaultProfilePicture(ctx *gin.Context) error {
	if err := c.appAssetsService.DeleteAsset(ctx.Request.Context(), "default-profile-picture"); err != nil {
		return err
	}

	return nil
}
