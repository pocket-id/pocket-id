package usergroups

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	appConfig appconfig.AppConfigResolver
	service   *Service
}

func newHandler(appConfig appconfig.AppConfigResolver, service *Service) *handler {
	return &handler{appConfig: appConfig, service: service}
}

func (h *handler) list(c *gin.Context) (dto.Paginated[dto.UserGroupMinimalDto], error) {
	return h.listGroups(c.Request.Context(), c.Query("search"), utils.ParseListRequestOptions(c))
}

func (h *handler) listGroups(ctx context.Context, searchTerm string, listRequestOptions utils.ListRequestOptions) (dto.Paginated[dto.UserGroupMinimalDto], error) {
	groups, pagination, err := h.service.List(ctx, searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[dto.UserGroupMinimalDto]{}, err
	}

	// Map the user groups to DTOs
	var groupsDto = make([]dto.UserGroupMinimalDto, len(groups))
	for i, group := range groups {
		var groupDto dto.UserGroupMinimalDto
		if err := dto.MapStruct(group, &groupDto); err != nil {
			return dto.Paginated[dto.UserGroupMinimalDto]{}, err
		}
		groupDto.UserCount, err = h.service.GetUserCountOfGroup(ctx, group.ID)
		if err != nil {
			return dto.Paginated[dto.UserGroupMinimalDto]{}, err
		}
		groupsDto[i] = groupDto
	}

	return dto.Paginated[dto.UserGroupMinimalDto]{Data: groupsDto, Pagination: pagination}, nil
}

func (h *handler) get(c *gin.Context) (dto.UserGroupDto, error) {
	group, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return dto.UserGroupDto{}, err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return dto.UserGroupDto{}, err
	}

	return groupDto, nil
}

func (h *handler) create(c *gin.Context, input dto.UserGroupCreateDto) (dto.UserGroupDto, error) {
	group, err := h.service.Create(c.Request.Context(), input, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserGroupDto{}, err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return dto.UserGroupDto{}, err
	}

	return groupDto, nil
}

func (h *handler) update(c *gin.Context, update dto.Update[dto.UserGroupCreateDto]) (dto.UserGroupDto, error) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserGroupDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	group, err := h.service.Update(c.Request.Context(), dbConfig, c.Param("id"), update)
	if err != nil {
		return dto.UserGroupDto{}, err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return dto.UserGroupDto{}, err
	}

	return groupDto, nil
}

func (h *handler) delete(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	return h.service.Delete(c.Request.Context(), dbConfig, c.Param("id"), auditlogs.OriginFrom(c))
}

func (h *handler) updateUsers(c *gin.Context, input dto.UserGroupUpdateUsersDto) (dto.UserGroupDto, error) {
	group, err := h.service.UpdateUsers(c.Request.Context(), c.Param("id"), input.UserIDs, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserGroupDto{}, err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return dto.UserGroupDto{}, err
	}

	return groupDto, nil
}

func (h *handler) updateAllowedOidcClients(c *gin.Context, input dto.UserGroupUpdateAllowedOidcClientsDto) (dto.UserGroupDto, error) {
	userGroup, err := h.service.UpdateAllowedOidcClient(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserGroupDto{}, err
	}

	var userGroupDto dto.UserGroupDto
	if err := dto.MapStruct(userGroup, &userGroupDto); err != nil {
		return dto.UserGroupDto{}, err
	}

	return userGroupDto, nil
}
