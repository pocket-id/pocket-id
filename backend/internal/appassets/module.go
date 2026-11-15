package appassets

import (
	"context"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

// imageTypes are the types of every image that utils.GetImageMimeType accepts, which the logo, background and default profile picture can have
var imageTypes = []string{"image/png", "image/jpeg", "image/svg+xml", "image/x-icon", "image/gif", "image/webp", "image/avif", "image/heic"}

// Module serves the logo, background, favicon, title font and default profile picture admins can replace
type Module struct {
	service *Service
	handler *handler
}

// New copies the bundled assets into the file storage where no custom ones exist, so it must run before the module serves anything
func New(ctx context.Context, fileStorage storage.FileStorage) (*Module, error) {
	extensions, err := initApplicationAssets(ctx, fileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize application assets: %w", err)
	}

	service := &Service{extensions: extensions, storage: fileStorage}
	return &Module{
		service: service,
		handler: &handler{service: service},
	}, nil
}

// RegisterRoutes mounts the application asset endpoints, where uploads pass through uploadLimit
func (m *Module) RegisterRoutes(r *authz.Router, uploadLimit gin.HandlerFunc) {
	// Every asset GET falls back to the bundled default unless asked not to
	fallback := option.QueryBool("default", "Return the bundled default if no custom one is set", param.Default(true))

	r.Public().GET("/application-assets/logo", httpserver.Handle(m.handler.getLogoHandler),
		option.Summary("Get logo image"),
		option.Description("Get the logo image for the application"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
		fallback,
		openapi.File("Logo image", imageTypes...),
	)
	r.Public().GET("/application-assets/email", httpserver.Handle(m.handler.getEmailLogoHandler),
		option.Summary("Get email logo image"),
		option.Description("Get the email logo image for use in emails"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Email logo image", "image/png", "image/jpeg"),
	)
	r.Public().GET("/application-assets/background", httpserver.Handle(m.handler.getBackgroundImageHandler),
		option.Summary("Get background image"),
		option.Description("Get the background image for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Background image", imageTypes...),
	)
	r.Public().GET("/application-assets/favicon", httpserver.Handle(m.handler.getFaviconHandler),
		option.Summary("Get favicon"),
		option.Description("Get the favicon for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Favicon image", "image/svg+xml", "image/png", "image/x-icon"),
	)
	r.Public().GET("/application-assets/font", httpserver.Handle(m.handler.getFontHandler),
		option.Summary("Get title font"),
		option.Description("Get the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Font file", "font/woff2", "font/woff", "font/ttf", "font/otf"),
	)

	// Everything below changes the assets or reads the one only the admin UI shows
	admin := r.Group("/admin")
	admin.GET("/application-assets/default-profile-picture", authz.ConfigRead, httpserver.Handle(m.handler.getDefaultProfilePicture),
		option.Summary("Get default profile picture image"),
		option.Description("Get the default profile picture image for the application"),
		option.Tags("Application Assets"),
		fallback,
		openapi.File("Default profile picture image", imageTypes...),
	)

	uploads := admin.With(uploadLimit)
	uploads.PUT("/application-assets/logo", authz.ConfigWrite, httpserver.NoContent(m.handler.updateLogoHandler),
		option.Summary("Update logo"),
		option.Description("Update the application logo"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
		openapi.Upload("Logo image file"),
	)
	uploads.PUT("/application-assets/email", authz.ConfigWrite, httpserver.NoContent(m.handler.updateEmailLogoHandler),
		option.Summary("Update email logo"),
		option.Description("Update the email logo for use in emails"),
		option.Tags("Application Assets"),
		openapi.Upload("Email logo image file"),
	)
	uploads.PUT("/application-assets/background", authz.ConfigWrite, httpserver.NoContent(m.handler.updateBackgroundImageHandler),
		option.Summary("Update background image"),
		option.Description("Update the application background image"),
		option.Tags("Application Assets"),
		openapi.Upload("Background image file"),
	)
	uploads.PUT("/application-assets/favicon", authz.ConfigWrite, httpserver.NoContent(m.handler.updateFaviconHandler),
		option.Summary("Update favicon"),
		option.Description("Update the application favicon"),
		option.Tags("Application Assets"),
		openapi.Upload("Favicon file (.svg/.png/.ico)"),
	)
	uploads.PUT("/application-assets/default-profile-picture", authz.ConfigWrite, httpserver.NoContent(m.handler.updateDefaultProfilePicture),
		option.Summary("Update default profile picture image"),
		option.Description("Update the default profile picture image"),
		option.Tags("Application Assets"),
		openapi.Upload("Profile picture image file"),
	)
	uploads.PUT("/application-assets/font", authz.ConfigWrite, httpserver.NoContent(m.handler.updateFontHandler),
		option.Summary("Update title font"),
		option.Description("Update the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
		openapi.Upload("Font file (.woff2/.woff/.ttf/.otf)"),
	)

	admin.DELETE("/application-assets/logo", authz.ConfigWrite, httpserver.NoContent(m.handler.deleteLogoHandler),
		option.Summary("Delete logo image"),
		option.Description("Delete the custom application logo and restore the default logo"),
		option.Tags("Application Assets"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
	)
	admin.DELETE("/application-assets/background", authz.ConfigWrite, httpserver.NoContent(m.handler.deleteBackgroundImageHandler),
		option.Summary("Delete background image"),
		option.Description("Delete the application background image"),
		option.Tags("Application Assets"),
	)
	admin.DELETE("/application-assets/default-profile-picture", authz.ConfigWrite, httpserver.NoContent(m.handler.deleteDefaultProfilePicture),
		option.Summary("Delete default profile picture image"),
		option.Description("Delete the default profile picture image"),
		option.Tags("Application Assets"),
	)
	admin.DELETE("/application-assets/font", authz.ConfigWrite, httpserver.NoContent(m.handler.deleteFontHandler),
		option.Summary("Delete title font"),
		option.Description("Delete the custom font of page and dialog titles"),
		option.Tags("Application Assets"),
	)
}

// IsDefaultProfilePictureSet reports whether an admin uploaded a default profile picture
// It is consumed by the users module, which serves it in place of the generated initials
func (m *Module) IsDefaultProfilePictureSet() bool {
	return m.service.IsDefaultProfilePictureSet()
}

// GetAsset returns an uploaded asset without falling back to the bundled default
func (m *Module) GetAsset(ctx context.Context, name string) (io.ReadCloser, int64, string, error) {
	return m.service.GetAsset(ctx, name)
}
