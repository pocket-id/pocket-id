package controller

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

// NewWebManifestController serves the web app manifest, which carries the configured application name when the app is added to a home screen
func NewWebManifestController(group *gin.RouterGroup, appConfigService *appconfig.AppConfigService) {
	wmc := &WebManifestController{appConfigService: appConfigService}

	group.GET("/app.webmanifest", httpserver.Handle(wmc.webManifestHandler).Handler)
}

type WebManifestController struct {
	appConfigService *appconfig.AppConfigService
}

func (wmc *WebManifestController) webManifestHandler(c *gin.Context) error {
	cfg, err := wmc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	manifest, err := json.Marshal(map[string]any{
		"name":        cfg.AppName.String(),
		"orientation": "any",
		"display":     "minimal-ui",
		"dir":         "auto",
		"start_url":   "/",
		"icons": []map[string]string{
			{"src": "img/static-logo-rounded-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
			{"src": "img/static-logo-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
			{"src": "img/static-logo.svg", "sizes": "any", "type": "image/svg+xml", "purpose": "maskable monochrome"},
		},
	})
	if err != nil {
		return err
	}

	c.Data(http.StatusOK, "application/manifest+json", manifest)
	return nil
}
