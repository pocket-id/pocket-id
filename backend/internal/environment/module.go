package environment

import (
	"net/http"

	"github.com/go-fuego/fuego/option"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	DB         *gorm.DB
	HTTPClient *http.Client

	// SQLiteOnNetworkedFilesystem is true when the SQLite database was found on a networked filesystem
	// It's always false when using Postgres
	SQLiteOnNetworkedFilesystem bool
}

// Module exposes read-only facts about the environment Pocket ID runs in, such as its version and the problems admins should fix
type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps)
	return &Module{
		service: service,
		handler: newHandler(service),
	}
}

// RegisterRoutes mounts the environment endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/version/latest", authz.Public, httpserver.JSON(http.StatusOK, m.handler.getLatestVersion),
		option.Summary("Get latest available version of Pocket ID"),
		option.Tags("Version"),
	)
	r.GET("/version/current", authz.Authenticated, httpserver.JSON(http.StatusOK, m.handler.getCurrentVersion),
		option.Summary("Get current deployed version of Pocket ID"),
		option.Tags("Version"),
	)

	admin := r.Group("/admin")
	admin.GET("/warnings", authz.ConfigRead, httpserver.JSON(http.StatusOK, m.handler.getWarnings),
		option.Summary("Get the warnings for admins"),
		option.Description("Lists the problems with this instance that admins should fix, such as a SQLite database on a networked filesystem or a database that a newer Pocket ID version migrated"),
		option.Tags("Warnings"),
	)
}
