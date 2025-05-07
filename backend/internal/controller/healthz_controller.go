package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewHealthzController creates a new controller for the healthcheck endpoints
// @Summary Healthcheck controller
// @Description Initializes healthcheck endpoints
// @Tags Health
func NewHealthzController(r *gin.Engine) {
	hc := &HealthzController{}

	r.GET("/healthz", hc.healthzHandler)
}

type HealthzController struct{}

// healthzHandler godoc
// @Summary Responds to healthchecks
// @Description Responds with a successful status code to healthcheck requests
// @Tags Health
// @Success 204 ""
// @Router /healthz [get]
func (hc *HealthzController) healthzHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
