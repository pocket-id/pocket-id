package api

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) list(c *gin.Context) (dto.Paginated[apiResponseDto], error) {
	search := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	apis, pagination, err := h.service.List(c.Request.Context(), search, listRequestOptions)
	if err != nil {
		return dto.Paginated[apiResponseDto]{}, err
	}

	var items []apiResponseDto
	if err := dto.MapStructList(apis, &items); err != nil {
		return dto.Paginated[apiResponseDto]{}, err
	}

	return dto.Paginated[apiResponseDto]{
		Data:       items,
		Pagination: pagination,
	}, nil
}

func (h *handler) get(c *gin.Context) (apiResponseDto, error) {
	api, err := h.service.Get(c.Request.Context(), nil, c.Param("id"))
	if err != nil {
		return apiResponseDto{}, err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return apiResponseDto{}, err
	}

	return responseDto, nil
}

func (h *handler) create(c *gin.Context, input apiCreateDto) (apiResponseDto, error) {
	api, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		return apiResponseDto{}, err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return apiResponseDto{}, err
	}

	return responseDto, nil
}

func (h *handler) update(c *gin.Context, update dto.Update[apiUpdateDto]) (apiResponseDto, error) {
	api, err := h.service.Update(c.Request.Context(), c.Param("id"), update)
	if err != nil {
		return apiResponseDto{}, err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return apiResponseDto{}, err
	}

	return responseDto, nil
}

func (h *handler) delete(c *gin.Context) error {
	return h.service.Delete(c.Request.Context(), c.Param("id"))
}

func (h *handler) updatePermissions(c *gin.Context, input apiPermissionsUpdateDto) (apiResponseDto, error) {
	api, err := h.service.UpdatePermissions(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return apiResponseDto{}, err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return apiResponseDto{}, err
	}

	return responseDto, nil
}

func (h *handler) updateCimdAccess(c *gin.Context, input apiCimdAccessUpdateDto) (apiResponseDto, error) {
	api, err := h.service.SetCIMDAccess(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return apiResponseDto{}, err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return apiResponseDto{}, err
	}

	return responseDto, nil
}

func (h *handler) listClients(c *gin.Context) (dto.Paginated[apiClientAccessDto], error) {
	clients, pagination, err := h.service.ListAPIClients(c.Request.Context(), c.Param("id"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return dto.Paginated[apiClientAccessDto]{}, err
	}

	var items []apiClientAccessDto
	if err := dto.MapStructList(clients, &items); err != nil {
		return dto.Paginated[apiClientAccessDto]{}, err
	}

	return dto.Paginated[apiClientAccessDto]{
		Data:       items,
		Pagination: pagination,
	}, nil
}

func (h *handler) listAssignableClients(c *gin.Context) (dto.Paginated[apiClientDto], error) {
	clients, pagination, err := h.service.ListAssignableClients(c.Request.Context(), c.Param("id"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return dto.Paginated[apiClientDto]{}, err
	}

	var items []apiClientDto
	if err := dto.MapStructList(clients, &items); err != nil {
		return dto.Paginated[apiClientDto]{}, err
	}

	return dto.Paginated[apiClientDto]{
		Data:       items,
		Pagination: pagination,
	}, nil
}

func (h *handler) listAssignableApis(c *gin.Context) (dto.Paginated[apiResponseDto], error) {
	apis, pagination, err := h.service.ListAssignableAPIs(c.Request.Context(), c.Param("clientId"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return dto.Paginated[apiResponseDto]{}, err
	}

	var items []apiResponseDto
	if err := dto.MapStructList(apis, &items); err != nil {
		return dto.Paginated[apiResponseDto]{}, err
	}

	return dto.Paginated[apiResponseDto]{
		Data:       items,
		Pagination: pagination,
	}, nil
}

func (h *handler) updateClientAccessForApi(c *gin.Context, input apiClientGrantUpdateDto) (apiClientGrantDto, error) {
	applied, err := h.service.SetAPIClientAccess(c.Request.Context(), c.Param("id"), c.Param("clientId"), APIClientGrant(input))
	if err != nil {
		return apiClientGrantDto{}, err
	}

	var grant apiClientGrantDto
	if err := dto.MapStruct(applied, &grant); err != nil {
		return apiClientGrantDto{}, err
	}

	return grant, nil
}

func (h *handler) removeClientAccessForApi(c *gin.Context) error {
	return h.service.RemoveAPIClientAccess(c.Request.Context(), c.Param("id"), c.Param("clientId"))
}

func (h *handler) listClientApis(c *gin.Context) ([]clientApiGrantDto, error) {
	grants, err := h.service.ListClientAPIs(c.Request.Context(), c.Param("clientId"))
	if err != nil {
		return nil, err
	}

	var items []clientApiGrantDto
	if err := dto.MapStructList(grants, &items); err != nil {
		return nil, err
	}

	return items, nil
}
