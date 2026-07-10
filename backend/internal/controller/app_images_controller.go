package controller

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type imageGetInput struct {
	Light string `query:"light" default:"true" required:"false"`
}

type imageUploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type imageUploadInput struct {
	RawBody huma.MultipartFormFiles[imageUploadForm]
}

type logoUploadInput struct {
	Light   string `query:"light" default:"true" required:"false"`
	RawBody huma.MultipartFormFiles[imageUploadForm]
}

type imageOutput struct {
	ContentType   string `header:"Content-Type"`
	ContentLength int64  `header:"Content-Length"`
	CacheControl  string `header:"Cache-Control"`
	Body          func(huma.Context)
}

func NewAppImagesController(api huma.API, authMiddleware *middleware.AuthMiddleware, appImagesService *service.AppImagesService) {
	controller := &AppImagesController{appImagesService: appImagesService}

	httpapi.Register(api, imageOperation("get-application-logo", http.MethodGet, "/api/application-images/logo", "Get logo image"), controller.getLogoHandler)
	httpapi.Register(api, imageOperation("get-email-logo", http.MethodGet, "/api/application-images/email", "Get email logo image"), controller.getEmailLogoHandler)
	httpapi.Register(api, imageOperation("get-background-image", http.MethodGet, "/api/application-images/background", "Get background image"), controller.getBackgroundImageHandler)
	httpapi.Register(api, imageOperation("get-favicon", http.MethodGet, "/api/application-images/favicon", "Get favicon"), controller.getFaviconHandler)

	auth := authMiddleware.Huma(api)
	defaultPictureOperation := imageOperation("get-default-profile-picture", http.MethodGet, "/api/application-images/default-profile-picture", "Get default profile picture")
	auth(&defaultPictureOperation)
	httpapi.Register(api, defaultPictureOperation, controller.getDefaultProfilePicture)

	logoOperation := imageOperation("update-application-logo", http.MethodPut, "/api/application-images/logo", "Update logo")
	logoOperation.DefaultStatus = http.StatusNoContent
	auth(&logoOperation)
	httpapi.Register(api, logoOperation, controller.updateLogoHandler)

	registerUpload := func(operation huma.Operation, handler func(context.Context, *imageUploadInput) (*httpapi.EmptyOutput, error)) {
		operation.DefaultStatus = http.StatusNoContent
		auth(&operation)
		httpapi.Register(api, operation, handler)
	}
	registerUpload(imageOperation("update-email-logo", http.MethodPut, "/api/application-images/email", "Update email logo"), controller.updateEmailLogoHandler)
	registerUpload(imageOperation("update-background-image", http.MethodPut, "/api/application-images/background", "Update background image"), controller.updateBackgroundImageHandler)
	registerUpload(imageOperation("update-favicon", http.MethodPut, "/api/application-images/favicon", "Update favicon"), controller.updateFaviconHandler)
	registerUpload(imageOperation("update-default-profile-picture", http.MethodPut, "/api/application-images/default-profile-picture", "Update default profile picture"), controller.updateDefaultProfilePicture)

	registerDelete := func(operation huma.Operation, handler func(context.Context, *httpapi.EmptyInput) (*httpapi.EmptyOutput, error)) {
		operation.DefaultStatus = http.StatusNoContent
		auth(&operation)
		httpapi.Register(api, operation, handler)
	}
	registerDelete(imageOperation("delete-background-image", http.MethodDelete, "/api/application-images/background", "Delete background image"), controller.deleteBackgroundImageHandler)
	registerDelete(imageOperation("delete-default-profile-picture", http.MethodDelete, "/api/application-images/default-profile-picture", "Delete default profile picture"), controller.deleteDefaultProfilePicture)
}

func imageOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"Application Images"}}
}

type AppImagesController struct {
	appImagesService *service.AppImagesService
}

func (c *AppImagesController) getLogoHandler(ctx context.Context, input *imageGetInput) (*imageOutput, error) {
	lightLogo, _ := strconv.ParseBool(input.Light)
	imageName := "logoLight"
	if !lightLogo {
		imageName = "logoDark"
	}
	return c.getImage(ctx, imageName)
}

func (c *AppImagesController) getEmailLogoHandler(ctx context.Context, _ *httpapi.EmptyInput) (*imageOutput, error) {
	return c.getImage(ctx, "logoEmail")
}

func (c *AppImagesController) getBackgroundImageHandler(ctx context.Context, _ *httpapi.EmptyInput) (*imageOutput, error) {
	return c.getImage(ctx, "background")
}

func (c *AppImagesController) getFaviconHandler(ctx context.Context, _ *httpapi.EmptyInput) (*imageOutput, error) {
	return c.getImage(ctx, "favicon")
}

func (c *AppImagesController) getDefaultProfilePicture(ctx context.Context, _ *httpapi.EmptyInput) (*imageOutput, error) {
	return c.getImage(ctx, "default-profile-picture")
}

func (c *AppImagesController) updateLogoHandler(ctx context.Context, input *logoUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	lightLogo, _ := strconv.ParseBool(input.Light)
	imageName := "logoLight"
	if !lightLogo {
		imageName = "logoDark"
	}
	if err := c.appImagesService.UpdateImage(ctx, file, imageName); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (c *AppImagesController) updateEmailLogoHandler(ctx context.Context, input *imageUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	mimeType := utils.GetImageMimeType(utils.GetFileExtension(file.Filename))
	if mimeType != "image/png" && mimeType != "image/jpeg" {
		return nil, &common.WrongFileTypeError{ExpectedFileType: ".png or .jpg/jpeg"}
	}
	return c.updateImage(ctx, file, "logoEmail")
}

func (c *AppImagesController) updateBackgroundImageHandler(ctx context.Context, input *imageUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	return c.updateImage(ctx, file, "background")
}

func (c *AppImagesController) updateFaviconHandler(ctx context.Context, input *imageUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	mimeType := utils.GetImageMimeType(strings.ToLower(utils.GetFileExtension(file.Filename)))
	if !slices.Contains([]string{"image/svg+xml", "image/png", "image/x-icon"}, mimeType) {
		return nil, &common.WrongFileTypeError{ExpectedFileType: ".svg or .png or .ico"}
	}
	return c.updateImage(ctx, file, "favicon")
}

func (c *AppImagesController) updateDefaultProfilePicture(ctx context.Context, input *imageUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	return c.updateImage(ctx, file, "default-profile-picture")
}

func (c *AppImagesController) updateImage(ctx context.Context, file *multipart.FileHeader, name string) (*httpapi.EmptyOutput, error) {
	if err := c.appImagesService.UpdateImage(ctx, file, name); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func uploadFile(form *multipart.Form) (*multipart.FileHeader, error) {
	files := form.File["file"]
	if len(files) == 0 {
		return nil, http.ErrMissingFile
	}
	return files[0], nil
}

func (c *AppImagesController) deleteBackgroundImageHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	if err := c.appImagesService.DeleteImage(ctx, "background"); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (c *AppImagesController) deleteDefaultProfilePicture(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	if err := c.appImagesService.DeleteImage(ctx, "default-profile-picture"); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (c *AppImagesController) getImage(ctx context.Context, name string) (*imageOutput, error) {
	reader, size, mimeType, err := c.appImagesService.GetImage(ctx, name)
	if err != nil {
		return nil, err
	}
	cacheControl := ""
	if !httpapi.QueryPresent(ctx, "skipCache") {
		cacheControl = utils.CacheControlValue(15*time.Minute, 24*time.Hour)
	}
	return &imageOutput{
		ContentType:   mimeType,
		ContentLength: size,
		CacheControl:  cacheControl,
		Body: func(streamCtx huma.Context) {
			defer reader.Close()
			_, _ = io.Copy(streamCtx.BodyWriter(), reader)
		},
	}, nil
}
