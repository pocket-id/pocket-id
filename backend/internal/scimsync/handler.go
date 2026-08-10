package scimsync

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

// getServiceProviderByClient godoc
// @Summary Get SCIM service provider
// @Description Get the SCIM service provider configuration for an OIDC client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} ScimServiceProviderDTO "SCIM service provider configuration"
// @Router /api/oidc/clients/{id}/scim-service-provider [get]
func (h *handler) getServiceProviderByClient(c *gin.Context) error {
	provider, err := h.service.GetServiceProviderByClient(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return respondWithServiceProvider(c, http.StatusOK, provider)
}

// syncServiceProvider godoc
// @Summary Sync SCIM service provider
// @Description Trigger synchronization for a SCIM service provider
// @Tags SCIM
// @Param id path string true "Service Provider ID"
// @Success 200 "OK"
// @Router /api/scim/service-provider/{id}/sync [post]
func (h *handler) syncServiceProvider(c *gin.Context) error {
	// The sync runs inline rather than through the actor so the response reports whether it succeeded
	err := h.service.SyncServiceProvider(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusOK)
	return nil
}

// createServiceProvider godoc
// @Summary Create SCIM service provider
// @Description Create a new SCIM service provider
// @Tags SCIM
// @Accept json
// @Produce json
// @Param serviceProvider body ScimServiceProviderCreateDTO true "SCIM service provider information"
// @Success 201 {object} ScimServiceProviderDTO "Created SCIM service provider"
// @Router /api/scim/service-provider [post]
func (h *handler) createServiceProvider(c *gin.Context) error {
	var input ScimServiceProviderCreateDTO
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	provider, err := h.service.CreateServiceProvider(c.Request.Context(), &input)
	if err != nil {
		return err
	}

	return respondWithServiceProvider(c, http.StatusCreated, provider)
}

// updateServiceProvider godoc
// @Summary Update SCIM service provider
// @Description Update an existing SCIM service provider
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "Service Provider ID"
// @Param serviceProvider body ScimServiceProviderCreateDTO true "SCIM service provider information"
// @Success 200 {object} ScimServiceProviderDTO "Updated SCIM service provider"
// @Router /api/scim/service-provider/{id} [put]
func (h *handler) updateServiceProvider(c *gin.Context) error {
	var input ScimServiceProviderCreateDTO
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	provider, err := h.service.UpdateServiceProvider(c.Request.Context(), c.Param("id"), &input)
	if err != nil {
		return err
	}

	return respondWithServiceProvider(c, http.StatusOK, provider)
}

// deleteServiceProvider godoc
// @Summary Delete SCIM service provider
// @Description Delete a SCIM service provider by ID
// @Tags SCIM
// @Param id path string true "Service Provider ID"
// @Success 204 "No Content"
// @Router /api/scim/service-provider/{id} [delete]
func (h *handler) deleteServiceProvider(c *gin.Context) error {
	err := h.service.DeleteServiceProvider(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

func respondWithServiceProvider(c *gin.Context, status int, provider ServiceProvider) error {
	var output ScimServiceProviderDTO
	err := dto.MapStruct(provider, &output)
	if err != nil {
		return err
	}

	c.JSON(status, output)
	return nil
}
