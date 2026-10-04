package logopreset

import (
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

// search godoc
// @Summary Search logo presets
// @Description Search the icon library for logos that can be used for OIDC clients
// @Tags OIDC
// @Produce json
// @Param search query string false "Search term matched against the icon name"
// @Success 200 {array} logoPresetDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/logo-presets [get]
func (h *handler) search(c *gin.Context) error {
	presets, err := h.service.Search(c.Request.Context(), c.Query("search"))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, presets)
	return nil
}
