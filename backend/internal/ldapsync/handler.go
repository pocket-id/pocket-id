package ldapsync

import (
	"fmt"
	"net/http"

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

// syncLdap godoc
// @Summary Synchronize LDAP
// @Description Manually trigger LDAP synchronization
// @Tags Application Configuration
// @Success 204 "No Content"
// @Router /api/application-configuration/sync-ldap [post]
func (h *handler) syncLdap(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	// The sync runs inline rather than through the actor, so the response reports whether it succeeded
	err = h.service.SyncAll(c.Request.Context(), dbConfig)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
