package scimsync

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) getServiceProviderByClient(c *gin.Context) (ScimServiceProviderDTO, error) {
	provider, err := h.service.GetServiceProviderByClient(c.Request.Context(), c.Param("id"))
	if err != nil {
		return ScimServiceProviderDTO{}, err
	}

	return serviceProviderDto(provider)
}

func (h *handler) syncServiceProvider(c *gin.Context) error {
	// The sync runs inline rather than through the actor so the response reports whether it succeeded
	err := h.service.SyncServiceProvider(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusOK)
	return nil
}

func (h *handler) createServiceProvider(c *gin.Context, input ScimServiceProviderCreateDTO) (ScimServiceProviderDTO, error) {
	provider, err := h.service.CreateServiceProvider(c.Request.Context(), &input)
	if err != nil {
		return ScimServiceProviderDTO{}, err
	}

	return serviceProviderDto(provider)
}

func (h *handler) updateServiceProvider(c *gin.Context, update dto.Update[ScimServiceProviderCreateDTO]) (ScimServiceProviderDTO, error) {
	provider, err := h.service.UpdateServiceProvider(c.Request.Context(), c.Param("id"), update)
	if err != nil {
		return ScimServiceProviderDTO{}, err
	}

	return serviceProviderDto(provider)
}

func (h *handler) deleteServiceProvider(c *gin.Context) error {
	return h.service.DeleteServiceProvider(c.Request.Context(), c.Param("id"))
}

func serviceProviderDto(provider ServiceProvider) (ScimServiceProviderDTO, error) {
	var output ScimServiceProviderDTO
	err := dto.MapStruct(provider, &output)
	if err != nil {
		return ScimServiceProviderDTO{}, err
	}
	output.HasToken = provider.Token != ""
	return output, nil
}
