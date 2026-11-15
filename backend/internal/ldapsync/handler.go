package ldapsync

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
)

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) syncLdap(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	// The sync runs inline rather than through the actor, so the response reports whether it succeeded
	return h.service.SyncAll(c.Request.Context(), dbConfig)
}
