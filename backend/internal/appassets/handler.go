package appassets

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	kitutils "github.com/italypaleale/go-kit/utils"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func (h *handler) getLogoHandler(ctx *gin.Context) error {
	return h.getAsset(ctx, logoImageName(ctx))
}

func (h *handler) getEmailLogoHandler(ctx *gin.Context) error {
	return h.getAsset(ctx, "logoEmail")
}

func (h *handler) getBackgroundImageHandler(ctx *gin.Context) error {
	return h.getAsset(ctx, "background")
}

func (h *handler) getFaviconHandler(ctx *gin.Context) error {
	return h.getAsset(ctx, "favicon")
}

func (h *handler) getDefaultProfilePicture(ctx *gin.Context) error {
	return h.getAsset(ctx, "default-profile-picture")
}

func (h *handler) updateLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, logoImageName(ctx)); err != nil {
		return err
	}

	return nil
}

func (h *handler) deleteLogoHandler(ctx *gin.Context) error {
	if err := h.service.DeleteAsset(ctx.Request.Context(), logoImageName(ctx)); err != nil {
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

func (h *handler) updateEmailLogoHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(fileType)

	if mimeType != "image/png" && mimeType != "image/jpeg" {
		return apperror.UnsupportedFileType("PNG or JPEG")
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, "logoEmail"); err != nil {
		return err
	}

	return nil
}

func (h *handler) updateBackgroundImageHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, "background"); err != nil {
		return err
	}

	return nil
}

func (h *handler) deleteBackgroundImageHandler(ctx *gin.Context) error {
	if err := h.service.DeleteAsset(ctx.Request.Context(), "background"); err != nil {
		return err
	}

	return nil
}

func (h *handler) updateFaviconHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	fileType := utils.GetFileExtension(file.Filename)
	mimeType := utils.GetImageMimeType(strings.ToLower(fileType))
	if !slices.Contains([]string{"image/svg+xml", "image/png", "image/x-icon"}, mimeType) {
		return apperror.UnsupportedFileType("SVG, PNG, or ICO")
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, "favicon"); err != nil {
		return err
	}

	return nil
}

func (h *handler) getFontHandler(ctx *gin.Context) error {
	return h.getAsset(ctx, "font")
}

func (h *handler) updateFontHandler(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, "font"); err != nil {
		return err
	}

	return nil
}

func (h *handler) deleteFontHandler(ctx *gin.Context) error {
	if err := h.service.DeleteAsset(ctx.Request.Context(), "font"); err != nil {
		return err
	}

	return nil
}

func (h *handler) getAsset(ctx *gin.Context, name string) error {
	getAsset := h.service.GetAsset
	if kitutils.IsTruthy(ctx.DefaultQuery("default", "true")) {
		getAsset = h.service.GetAssetWithDefault
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

func (h *handler) updateDefaultProfilePicture(ctx *gin.Context) error {
	file, err := httpserver.FormFile(ctx, "file")
	if err != nil {
		return err
	}

	if err := h.service.UpdateAsset(ctx.Request.Context(), file, "default-profile-picture"); err != nil {
		return err
	}

	return nil
}

func (h *handler) deleteDefaultProfilePicture(ctx *gin.Context) error {
	if err := h.service.DeleteAsset(ctx.Request.Context(), "default-profile-picture"); err != nil {
		return err
	}

	return nil
}
