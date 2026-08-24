package environment

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, auth gin.HandlerFunc) {
	apiGroup.GET("/version/latest", httpserver.Handle(m.handler.getLatestVersion))
	apiGroup.GET("/version/current", auth, httpserver.Handle(m.handler.getCurrentVersion))
	apiGroup.GET("/storage/sqlite-warning", auth, httpserver.Handle(m.handler.getSqliteStorageWarning))
}
