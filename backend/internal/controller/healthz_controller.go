package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

// NewHealthzController creates a new controller for the healthcheck endpoints
func NewHealthzController(r *authz.PublicRouter) {
	hc := &HealthzController{}

	r.GET("/healthz", httpserver.NoContent(hc.healthzHandler),
		option.Summary("Responds to healthchecks"),
		option.Description("Responds with a successful status code to healthcheck requests"),
		option.Tags("Health"),
	)
}

type HealthzController struct{}

func (hc *HealthzController) healthzHandler(*gin.Context) error {
	return nil
}
