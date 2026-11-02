package claimmappingpolicy

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// listCustomScopes godoc
// @Summary List custom OIDC scopes
// @Description Get a paginated list of the custom OIDC scopes with optional search and sorting
// @Tags OIDC Custom Scopes
// @Param search query string false "Search term to filter scopes by key or name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[CustomScopeDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/custom-scopes [get]
func (h *handler) listCustomScopes(c *gin.Context) error {
	scopes, pagination, err := h.service.ListCustomScopes(c.Request.Context(), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return err
	}

	dtos := make([]CustomScopeDto, len(scopes))
	for i, scope := range scopes {
		dtos[i] = customScopeToDto(scope)
	}
	c.JSON(http.StatusOK, dto.Paginated[CustomScopeDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// createCustomScope godoc
// @Summary Create a custom OIDC scope
// @Description Create a custom OIDC scope, which claim mappings can then be released with
// @Tags OIDC Custom Scopes
// @Accept json
// @Produce json
// @Param scope body CustomScopeDto true "Custom scope"
// @Success 201 {object} CustomScopeDto "Created custom scope"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/custom-scopes [post]
func (h *handler) createCustomScope(c *gin.Context) error {
	var input CustomScopeDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	scope, err := h.service.CreateCustomScope(c.Request.Context(), input)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, customScopeToDto(scope))
	return nil
}

// updateCustomScope godoc
// @Summary Update a custom OIDC scope
// @Description Update the name and description of a custom OIDC scope, whose key can't change
// @Tags OIDC Custom Scopes
// @Accept json
// @Produce json
// @Param id path string true "Custom scope ID"
// @Param scope body CustomScopeDto true "Custom scope"
// @Success 200 {object} CustomScopeDto "Updated custom scope"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/custom-scopes/{id} [put]
func (h *handler) updateCustomScope(c *gin.Context) error {
	var input CustomScopeDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	scope, err := h.service.UpdateCustomScope(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, customScopeToDto(scope))
	return nil
}

// deleteCustomScope godoc
// @Summary Delete a custom OIDC scope
// @Description Delete a custom OIDC scope that no claim mapping is released with
// @Tags OIDC Custom Scopes
// @Param id path string true "Custom scope ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/custom-scopes/{id} [delete]
func (h *handler) deleteCustomScope(c *gin.Context) error {
	err := h.service.DeleteCustomScope(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// listCustomScopeKeys godoc
// @Summary List the keys of all custom OIDC scopes
// @Description Get the keys of every custom OIDC scope, sorted, for pickers that offer all of them at once
// @Tags OIDC Custom Scopes
// @Produce json
// @Success 200 {array} string "Custom scope keys"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/custom-scopes/keys [get]
func (h *handler) listCustomScopeKeys(c *gin.Context) error {
	scopes, err := CustomScopesByKey(c.Request.Context(), h.service.db, nil)
	if err != nil {
		return err
	}

	keys := make([]string, len(scopes))
	for i, scope := range scopes {
		keys[i] = scope.Key
	}
	c.JSON(http.StatusOK, keys)
	return nil
}
