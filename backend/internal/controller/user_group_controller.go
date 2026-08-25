package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewUserGroupController creates a new controller for user group management
// @Summary User group management controller
// @Description Initializes all user group-related API endpoints
// @Tags User Groups
func NewUserGroupController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, appConfigService *appconfig.AppConfigService, userGroupService *service.UserGroupService) {
	ugc := UserGroupController{
		appConfigService: appConfigService,
		UserGroupService: userGroupService,
	}

	userGroupsGroup := group.Group("/user-groups")
	userGroupsGroup.Use(authMiddleware.Add())
	{
		userGroupsGroup.GET("", httpserver.Handle(ugc.list))
		userGroupsGroup.GET("/:id", httpserver.Handle(ugc.get))
		userGroupsGroup.POST("", httpserver.Handle(ugc.create))
		userGroupsGroup.PUT("/:id", httpserver.Handle(ugc.update))
		userGroupsGroup.DELETE("/:id", httpserver.Handle(ugc.delete))
		userGroupsGroup.PUT("/:id/users", httpserver.Handle(ugc.updateUsers))
		userGroupsGroup.PUT("/:id/allowed-oidc-clients", httpserver.Handle(ugc.updateAllowedOidcClients))
	}
}

type UserGroupController struct {
	appConfigService *appconfig.AppConfigService
	UserGroupService *service.UserGroupService
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
func (ugc *UserGroupController) list(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	groups, pagination, err := ugc.UserGroupService.List(c, searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	// Map the user groups to DTOs
	var groupsDto = make([]dto.UserGroupMinimalDto, len(groups))
	for i, group := range groups {
		var groupDto dto.UserGroupMinimalDto
		if err := dto.MapStruct(group, &groupDto); err != nil {
			return err
		}
		groupDto.UserCount, err = ugc.UserGroupService.GetUserCountOfGroup(c.Request.Context(), group.ID)
		if err != nil {
			return err
		}
		groupsDto[i] = groupDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.UserGroupMinimalDto]{
		Data:       groupsDto,
		Pagination: pagination,
	})
	return nil
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
func (ugc *UserGroupController) get(c *gin.Context) error {
	group, err := ugc.UserGroupService.Get(c.Request.Context(), c.Param("id"))
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
func (ugc *UserGroupController) create(c *gin.Context) error {
	var input dto.UserGroupCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := ugc.UserGroupService.Create(c.Request.Context(), input)
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
func (ugc *UserGroupController) update(c *gin.Context) error {
	dbConfig, err := ugc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input dto.UserGroupCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := ugc.UserGroupService.Update(c.Request.Context(), dbConfig, c.Param("id"), input)
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
func (ugc *UserGroupController) delete(c *gin.Context) error {
	dbConfig, err := ugc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := ugc.UserGroupService.Delete(c.Request.Context(), dbConfig, c.Param("id")); err != nil {
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
func (ugc *UserGroupController) updateUsers(c *gin.Context) error {
	var input dto.UserGroupUpdateUsersDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	group, err := ugc.UserGroupService.UpdateUsers(c.Request.Context(), c.Param("id"), input.UserIDs)
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
func (ugc *UserGroupController) updateAllowedOidcClients(c *gin.Context) error {
	var input dto.UserGroupUpdateAllowedOidcClientsDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	userGroup, err := ugc.UserGroupService.UpdateAllowedOidcClient(c.Request.Context(), c.Param("id"), input)
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
