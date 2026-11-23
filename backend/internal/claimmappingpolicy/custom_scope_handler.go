package claimmappingpolicy

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func (h *handler) listCustomScopes(c *gin.Context) (dto.Paginated[CustomScopeDto], error) {
	scopes, pagination, err := h.service.ListCustomScopes(c.Request.Context(), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return dto.Paginated[CustomScopeDto]{}, err
	}

	dtos := make([]CustomScopeDto, len(scopes))
	for i, scope := range scopes {
		dtos[i] = customScopeToDto(scope)
	}
	return dto.Paginated[CustomScopeDto]{
		Data:       dtos,
		Pagination: pagination,
	}, nil
}

func (h *handler) createCustomScope(c *gin.Context, input CustomScopeDto) (CustomScopeDto, error) {
	scope, err := h.service.CreateCustomScope(c.Request.Context(), input)
	if err != nil {
		return CustomScopeDto{}, err
	}

	return customScopeToDto(scope), nil
}

func (h *handler) updateCustomScope(c *gin.Context, update dto.Update[CustomScopeDto]) (CustomScopeDto, error) {
	scope, err := h.service.UpdateCustomScope(c.Request.Context(), c.Param("id"), update)
	if err != nil {
		return CustomScopeDto{}, err
	}

	return customScopeToDto(scope), nil
}

func (h *handler) deleteCustomScope(c *gin.Context) error {
	return h.service.DeleteCustomScope(c.Request.Context(), c.Param("id"))
}

func (h *handler) listCustomScopeKeys(c *gin.Context) ([]string, error) {
	scopes, err := CustomScopesByKey(c.Request.Context(), h.service.db, nil)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(scopes))
	for i, scope := range scopes {
		keys[i] = scope.Key
	}
	return keys, nil
}
