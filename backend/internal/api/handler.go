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

	var items []apiResponseDto
	if err := dto.MapStructList(apis, &items); err != nil {
		return err
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

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, responseDto)
	return nil
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

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}

	c.JSON(http.StatusCreated, responseDto)
	return nil
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

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, responseDto)
	return nil
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

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, responseDto)
	return nil
}

// updateCimdAccess godoc
// @Summary Update metadata document client access
// @Description Replace which permissions of an API every client registered through a Client ID Metadata Document may request
// @Tags APIs
// @Accept json
// @Produce json
// @Param id path string true "API ID"
// @Param access body apiCimdAccessUpdateDto true "Metadata document client access"
// @Success 200 {object} apiResponseDto "Updated API"
// @Router /api/apis/{id}/cimd-access [put]
func (h *handler) updateCimdAccess(c *gin.Context) error {
	var input apiCimdAccessUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	api, err := h.service.SetCIMDAccess(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var responseDto apiResponseDto
	if err := dto.MapStruct(api, &responseDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, responseDto)
	return nil
}

// listClients godoc
// @Summary List clients with access to an API
// @Description Get a paginated list of OIDC clients that may reach the API, with their permissions split into user-delegated and client (machine-to-machine) access
// @Tags APIs
// @Produce json
// @Param id path string true "API ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[apiClientAccessDto]
// @Router /api/apis/{id}/clients [get]
func (h *handler) listClients(c *gin.Context) error {
	clients, pagination, err := h.service.ListAPIClients(c.Request.Context(), c.Param("id"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return err
	}

	var items []apiClientAccessDto
	if err := dto.MapStructList(clients, &items); err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[apiClientAccessDto]{
		Data:       items,
		Pagination: pagination,
	})
	return nil
}

// listAssignableClients godoc
// @Summary List clients that can still be granted access to an API
// @Description Get a paginated list of OIDC clients that have no grant on the API yet
// @Tags APIs
// @Produce json
// @Param id path string true "API ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[apiClientDto]
// @Router /api/apis/{id}/assignable-clients [get]
func (h *handler) listAssignableClients(c *gin.Context) error {
	clients, pagination, err := h.service.ListAssignableClients(c.Request.Context(), c.Param("id"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return err
	}

	var items []apiClientDto
	if err := dto.MapStructList(clients, &items); err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[apiClientDto]{
		Data:       items,
		Pagination: pagination,
	})
	return nil
}

// listAssignableApis godoc
// @Summary List APIs a client can still be granted access to
// @Description Get a paginated list of APIs the OIDC client cannot already reach
// @Tags APIs
// @Produce json
// @Param clientId path string true "OIDC Client ID"
// @Param search query string false "Search term to filter APIs by name or resource"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[apiResponseDto]
// @Router /api/api-access/{clientId}/assignable-apis [get]
func (h *handler) listAssignableApis(c *gin.Context) error {
	apis, pagination, err := h.service.ListAssignableAPIs(c.Request.Context(), c.Param("clientId"), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return err
	}

	var items []apiResponseDto
	if err := dto.MapStructList(apis, &items); err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[apiResponseDto]{
		Data:       items,
		Pagination: pagination,
	})
	return nil
}

// updateClientAccessForApi godoc
// @Summary Update a client's access to an API
// @Description Replace the permissions of this API a single OIDC client may request, leaving its grants on other APIs untouched
// @Tags APIs
// @Accept json
// @Produce json
// @Param id path string true "API ID"
// @Param clientId path string true "OIDC Client ID"
// @Param access body apiClientGrantUpdateDto true "Access and allowed permission IDs per subject type"
// @Success 200 {object} apiClientGrantDto
// @Router /api/apis/{id}/clients/{clientId} [put]
func (h *handler) updateClientAccessForApi(c *gin.Context) error {
	var input apiClientGrantUpdateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	applied, err := h.service.SetAPIClientAccess(c.Request.Context(), c.Param("id"), c.Param("clientId"), APIClientGrant(input))
	if err != nil {
		return err
	}

	var grant apiClientGrantDto
	if err := dto.MapStruct(applied, &grant); err != nil {
		return err
	}

	c.JSON(http.StatusOK, grant)
	return nil
}

// removeClientAccessForApi godoc
// @Summary Revoke a client's access to an API
// @Description Remove every permission of this API a single OIDC client was allowed to request
// @Tags APIs
// @Param id path string true "API ID"
// @Param clientId path string true "OIDC Client ID"
// @Success 204 "No Content"
// @Router /api/apis/{id}/clients/{clientId} [delete]
func (h *handler) removeClientAccessForApi(c *gin.Context) error {
	err := h.service.RemoveAPIClientAccess(c.Request.Context(), c.Param("id"), c.Param("clientId"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// listClientApis godoc
// @Summary List APIs a client may access
// @Description Get every API the OIDC client may request tokens for, with its access and permissions split into user-delegated and client (machine-to-machine) access
// @Tags APIs
// @Produce json
// @Param clientId path string true "OIDC Client ID"
// @Success 200 {array} clientApiGrantDto
// @Router /api/api-access/{clientId}/apis [get]
func (h *handler) listClientApis(c *gin.Context) error {
	grants, err := h.service.ListClientAPIs(c.Request.Context(), c.Param("clientId"))
	if err != nil {
		return err
	}

	var items []clientApiGrantDto
	if err := dto.MapStructList(grants, &items); err != nil {
		return err
	}

	c.JSON(http.StatusOK, items)
	return nil
}
