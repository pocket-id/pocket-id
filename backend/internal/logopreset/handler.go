package logopreset

import (
	"github.com/gin-gonic/gin"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) search(c *gin.Context) ([]logoPresetDto, error) {
	return h.service.Search(c.Request.Context(), c.Query("search"))
}
