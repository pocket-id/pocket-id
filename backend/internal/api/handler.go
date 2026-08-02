package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

// list godoc
// @Summary List APIs
// @Description Get a paginated list of APIs with optional search and sorting
// @Tags APIs
// @Produce json
// @Param search query string false "Search term to filter APIs by name or resource"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[apiResponseDto]
// @Router /api/apis [get]
func (h *handler) list(c *gin.Context) error {
	search := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	apis, pagination, err := h.service.List(c.Request.Context(), search, listRequestOptions)
	if err != nil {
		return err
	}

	items := make([]apiResponseDto, len(apis))
	for i, api := range apis {
		var item apiResponseDto
		if err := dto.MapStruct(api, &item); err != nil {
			return err
		}
		item.Resource = api.Audience
		items[i] = item
	}

	c.JSON(http.StatusOK, dto.Paginated[apiResponseDto]{
		Data:       items,
		Pagination: pagination,
	})
	return nil
}

// get godoc
// @Summary Get API by ID
// @Description Retrieve a single API including its permissions
// @Tags APIs
// @Produce json
// @Param id path string true "API ID"
// @Success 200 {object} apiResponseDto
// @Router /api/apis/{id} [get]
func (h *handler) get(c *gin.Context) error {
	api, err := h.service.Get(c.Request.Context(), nil, c.Param("id"))
	if err != nil {
		return err
	}

	return h.respond(c, http.StatusOK, api)
}

// create godoc
// @Summary Create API
// @Description Create a new API resource server
// @Tags APIs
// @Accept json
// @Produce json
// @Param api body apiCreateDto true "API information"
// @Success 201 {object} apiResponseDto "Created API"
// @Router /api/apis [post]
func (h *handler) create(c *gin.Context) error {
	var input apiCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	api, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		return err
	}

	return h.respond(c, http.StatusCreated, api)
}

// update godoc
// @Summary Update API
// @Description Update an existing API by ID
// @Tags APIs
// @Accept json
// @Produce json
// @Param id path string true "API ID"
// @Param api body apiUpdateDto true "API information"
// @Success 200 {object} apiResponseDto "Updated API"
// @Router /api/apis/{id} [put]
func (h *handler) update(c *gin.Context) error {
	var input apiUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	api, err := h.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	return h.respond(c, http.StatusOK, api)
}

// delete godoc
// @Summary Delete API
// @Description Delete an API by ID
// @Tags APIs
// @Param id path string true "API ID"
// @Success 204 "No Content"
// @Router /api/apis/{id} [delete]
func (h *handler) delete(c *gin.Context) error {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// updatePermissions godoc
// @Summary Update API permissions
// @Description Replace the full set of permissions for an API
// @Tags APIs
// @Accept json
// @Produce json
// @Param id path string true "API ID"
// @Param permissions body apiPermissionsUpdateDto true "Permissions to set"
// @Success 200 {object} apiResponseDto "Updated API"
// @Router /api/apis/{id}/permissions [put]
func (h *handler) updatePermissions(c *gin.Context) error {
	var input apiPermissionsUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	api, err := h.service.UpdatePermissions(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	return h.respond(c, http.StatusOK, api)
}

// getClientAccess godoc
// @Summary Get client API access
// @Description Get the API permissions an OIDC client is allowed to request, split into user-delegated and client (machine-to-machine) access
// @Tags APIs
// @Produce json
// @Param clientId path string true "OIDC Client ID"
// @Success 200 {object} clientApiAccessDto
// @Router /api/api-access/{clientId} [get]
func (h *handler) getClientAccess(c *gin.Context) error {
	access, err := h.service.GetClientAPIAccess(c.Request.Context(), c.Param("clientId"))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, newClientApiAccessDto(access))
	return nil
}

// updateClientAccess godoc
// @Summary Update client API access
// @Description Replace the API permissions an OIDC client is allowed to request, split into user-delegated and client (machine-to-machine) access
// @Tags APIs
// @Accept json
// @Produce json
// @Param clientId path string true "OIDC Client ID"
// @Param access body clientApiAccessUpdateDto true "Allowed permission IDs per subject type"
// @Success 200 {object} clientApiAccessDto
// @Router /api/api-access/{clientId} [put]
func (h *handler) updateClientAccess(c *gin.Context) error {
	var input clientApiAccessUpdateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	applied, err := h.service.SetClientAPIAccess(c.Request.Context(), c.Param("clientId"), ClientAPIAccess(input))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, newClientApiAccessDto(applied))
	return nil
}

// newClientApiAccessDto always serializes both permission lists as arrays rather than null
func newClientApiAccessDto(access ClientAPIAccess) clientApiAccessDto {
	dto := clientApiAccessDto(access)
	if dto.UserDelegatedPermissionIDs == nil {
		dto.UserDelegatedPermissionIDs = []string{}
	}
	if dto.ClientPermissionIDs == nil {
		dto.ClientPermissionIDs = []string{}
	}
	return dto
}

func (h *handler) respond(c *gin.Context, status int, api API) error {
	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}
	responseDto.Resource = api.Audience
	c.JSON(status, responseDto)
	return nil
}
