package feature

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	r.GET("/features", authz.ConfigRead, httpserver.Handle(m.list))
}

// list godoc
// @Summary List optional features
// @Description Get whether each optional feature is on and whether anything uses it; a feature in use is always on
// @Tags Features
// @Produce json
// @Success 200 {object} map[string]feature.Status
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/features [get]
func (m *Module) list(c *gin.Context) error {
	statuses, err := m.service.List(c.Request.Context())
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, statuses)
	return nil
}
