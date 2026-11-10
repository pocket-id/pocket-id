package usergroups

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	appConfig appconfig.AppConfigResolver
	service   *Service
}

func newHandler(appConfig appconfig.AppConfigResolver, service *Service) *handler {
	return &handler{appConfig: appConfig, service: service}
}

// list godoc
// @Summary List user groups
// @Description Get a paginated list of user groups with optional search and sorting
// @Tags User Groups
// @Param search query string false "Search term to filter user groups by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.UserGroupMinimalDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups [get]
func (h *handler) list(c *gin.Context) error {
	groups, err := h.listGroups(c.Request.Context(), c.Query("search"), utils.ParseListRequestOptions(c))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, groups)
	return nil
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

// get godoc
// @Summary Get user group by ID
// @Description Retrieve detailed information about a specific user group including its users
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Success 200 {object} dto.UserGroupDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups/{id} [get]
func (h *handler) get(c *gin.Context) error {
	group, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, groupDto)
	return nil
}

// create godoc
// @Summary Create user group
// @Description Create a new user group
// @Tags User Groups
// @Accept json
// @Produce json
// @Param userGroup body dto.UserGroupCreateDto true "User group information"
// @Success 201 {object} dto.UserGroupDto "Created user group"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups [post]
func (h *handler) create(c *gin.Context) error {
	var input dto.UserGroupCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := h.service.Create(c.Request.Context(), input, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return err
	}

	c.JSON(http.StatusCreated, groupDto)
	return nil
}

// update godoc
// @Summary Update user group
// @Description Update an existing user group by ID
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Param userGroup body dto.UserGroupCreateDto true "User group information"
// @Success 200 {object} dto.UserGroupDto "Updated user group"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups/{id} [put]
func (h *handler) update(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input dto.UserGroupCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := h.service.Update(c.Request.Context(), dbConfig, c.Param("id"), input)
	if err != nil {
		return err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, groupDto)
	return nil
}

// delete godoc
// @Summary Delete user group
// @Description Delete a specific user group by ID
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups/{id} [delete]
func (h *handler) delete(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := h.service.Delete(c.Request.Context(), dbConfig, c.Param("id"), auditlogs.OriginFrom(c)); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// updateUsers godoc
// @Summary Update users in a group
// @Description Update the list of users belonging to a specific user group
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Param users body dto.UserGroupUpdateUsersDto true "List of user IDs to assign to this group"
// @Success 200 {object} dto.UserGroupDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups/{id}/users [put]
func (h *handler) updateUsers(c *gin.Context) error {
	var input dto.UserGroupUpdateUsersDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := h.service.UpdateUsers(c.Request.Context(), c.Param("id"), input.UserIDs, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	var groupDto dto.UserGroupDto
	if err := dto.MapStruct(group, &groupDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, groupDto)
	return nil
}

// updateAllowedOidcClients godoc
// @Summary Update allowed OIDC clients
// @Description Update the OIDC clients allowed for a specific user group
// @Tags OIDC
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Param groups body dto.UserGroupUpdateAllowedOidcClientsDto true "OIDC client IDs to allow"
// @Success 200 {object} dto.UserGroupDto "Updated user group"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/user-groups/{id}/allowed-oidc-clients [put]
func (h *handler) updateAllowedOidcClients(c *gin.Context) error {
	var input dto.UserGroupUpdateAllowedOidcClientsDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	userGroup, err := h.service.UpdateAllowedOidcClient(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	var userGroupDto dto.UserGroupDto
	if err := dto.MapStruct(userGroup, &userGroupDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, userGroupDto)
	return nil
}
