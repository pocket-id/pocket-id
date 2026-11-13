package feature

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	DB        *gorm.DB
	AppConfig appConfigStore
}

// Module reports which optional features are on, so the admin UI shows only their pages and tabs
// Admins turn features on and off through the enabledFeatures application configuration value
type Module struct {
	service *Service
}

func New(deps Dependencies) *Module {
	return &Module{service: &Service{db: deps.DB, appConfig: deps.AppConfig}}
}

// RegisterRoutes mounts the feature endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/features", authz.AccountRead, httpserver.JSON(http.StatusOK, m.list),
		option.Summary("List optional features"),
		option.Description("Get whether each optional feature is on and whether anything uses it; a feature in use is always on"),
		option.Tags("Features"),
	)
}

func (m *Module) list(c *gin.Context) (map[Feature]Status, error) {
	return m.service.List(c.Request.Context())
}
