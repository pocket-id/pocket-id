package environment

import (
	"net/http"

	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	HTTPClient *http.Client

	// SQLiteOnNetworkedFilesystem is true when the SQLite database was found on a networked filesystem
	// It's always false when using Postgres
	SQLiteOnNetworkedFilesystem bool
}

// Module exposes read-only facts about the environment Pocket ID runs in, such as its version and where its database is stored
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
	r.Public().GET("/version/latest", httpserver.JSON(http.StatusOK, m.handler.getLatestVersion),
		option.Summary("Get latest available version of Pocket ID"),
		option.Tags("Version"),
	)
	r.GET("/version/current", authz.AccountRead, httpserver.JSON(http.StatusOK, m.handler.getCurrentVersion),
		option.Summary("Get current deployed version of Pocket ID"),
		option.Tags("Version"),
	)
	r.GET("/storage/sqlite-warning", authz.AccountRead, httpserver.JSON(http.StatusOK, m.handler.getSqliteStorageWarning),
		option.Summary("Get whether the SQLite storage warning should be shown"),
		option.Description("Reports whether Pocket ID found its SQLite database on a networked filesystem, which is unsupported and can lead to database corruption"),
		option.Tags("Storage"),
	)
}
