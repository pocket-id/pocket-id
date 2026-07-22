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

func NewAppImagesController(api huma.API, authMiddleware *middleware.AuthMiddleware, fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware, appImagesService *service.AppImagesService) {
	controller := &AppImagesController{appImagesService: appImagesService}
	auth := authMiddleware.Huma(api)
	uploadLimit := httpapi.WithMiddleware(fileSizeLimitMiddleware.Huma(api, 2<<20))

	httpapi.Register(api, huma.Operation{
		OperationID: "get-application-logo",
		Method:      http.MethodGet,
		Path:        "/api/application-images/logo",
		Summary:     "Get logo image",
		Tags:        []string{"Application Images"},
	}, controller.getLogoHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-email-logo",
		Method:      http.MethodGet,
		Path:        "/api/application-images/email",
		Summary:     "Get email logo image",
		Tags:        []string{"Application Images"},
	}, controller.getEmailLogoHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-background-image",
		Method:      http.MethodGet,
		Path:        "/api/application-images/background",
		Summary:     "Get background image",
		Tags:        []string{"Application Images"},
	}, controller.getBackgroundImageHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-favicon",
		Method:      http.MethodGet,
		Path:        "/api/application-images/favicon",
		Summary:     "Get favicon",
		Tags:        []string{"Application Images"},
	}, controller.getFaviconHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-default-profile-picture",
		Method:      http.MethodGet,
		Path:        "/api/application-images/default-profile-picture",
		Summary:     "Get default profile picture",
		Tags:        []string{"Application Images"},
	}, controller.getDefaultProfilePicture, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-application-logo",
		Method:        http.MethodPut,
		Path:          "/api/application-images/logo",
		Summary:       "Update logo",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateLogoHandler, auth, uploadLimit)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-email-logo",
		Method:        http.MethodPut,
		Path:          "/api/application-images/email",
		Summary:       "Update email logo",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateEmailLogoHandler, auth, uploadLimit)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-background-image",
		Method:        http.MethodPut,
		Path:          "/api/application-images/background",
		Summary:       "Update background image",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateBackgroundImageHandler, auth, uploadLimit)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-favicon",
		Method:        http.MethodPut,
		Path:          "/api/application-images/favicon",
		Summary:       "Update favicon",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateFaviconHandler, auth, uploadLimit)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-default-profile-picture",
		Method:        http.MethodPut,
		Path:          "/api/application-images/default-profile-picture",
		Summary:       "Update default profile picture",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateDefaultProfilePicture, auth, uploadLimit)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-background-image",
		Method:        http.MethodDelete,
		Path:          "/api/application-images/background",
		Summary:       "Delete background image",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.deleteBackgroundImageHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-default-profile-picture",
		Method:        http.MethodDelete,
		Path:          "/api/application-images/default-profile-picture",
		Summary:       "Delete default profile picture",
		Tags:          []string{"Application Images"},
		DefaultStatus: http.StatusNoContent,
	}, controller.deleteDefaultProfilePicture, auth)
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
