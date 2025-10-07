package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewUserGroupController creates a new controller for user group management
// @Summary User group management controller
// @Description Initializes all user group-related API endpoints
// @Tags User Groups
func NewUserGroupController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, userGroupService *service.UserGroupService) {
	ugc := UserGroupController{
		UserGroupService: userGroupService,
	}

	userGroupsGroup := group.Group("/user-groups")
	userGroupsGroup.Use(authMiddleware.Add())
	{
		userGroupsGroup.GET("", ugc.list)
		userGroupsGroup.GET("/:id", ugc.get)
		userGroupsGroup.POST("", ugc.create)
		userGroupsGroup.PUT("/:id", ugc.update)
		userGroupsGroup.DELETE("/:id", ugc.delete)
		userGroupsGroup.PUT("/:id/users", ugc.updateUsers)
	}
}

type UserGroupController struct {
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
// @Success 200 {object} dto.Paginated[dto.UserGroupDtoWithUserCount]
// @Router /api/user-groups [get]
func (ugc *UserGroupController) list(c *gin.Context) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	groups, pagination, err := ugc.UserGroupService.List(c, searchTerm, listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Map the user groups to DTOs
	var groupsDto = make([]dto.UserGroupDtoWithUserCount, len(groups))
	for i, group := range groups {
		var groupDto dto.UserGroupDtoWithUserCount
		if err := dto.MapStruct(group, &groupDto); err != nil {
			_ = c.Error(err)
			return
		}
		groupDto.UserCount, err = ugc.UserGroupService.GetUserCountOfGroup(ctx, group.ID)
		if err != nil {
			_ = c.Error(err)
			return
		}
		groupsDto[i] = groupDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.UserGroupDtoWithUserCount]{
		Data:       groupsDto,
		Pagination: pagination,
	})
}

// get godoc
// @Summary Get user group by ID
// @Description Retrieve detailed information about a specific user group including its users
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Success 200 {object} dto.UserGroupDtoWithUsers
// @Router /api/user-groups/{id} [get]
func (ugc *UserGroupController) get(c *gin.Context) {
	group, err := ugc.UserGroupService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groupDto)
}

// create godoc
// @Summary Create user group
// @Description Create a new user group
// @Tags User Groups
// @Accept json
// @Produce json
// @Param userGroup body dto.UserGroupCreateDto true "User group information"
// @Success 201 {object} dto.UserGroupDtoWithUsers "Created user group"
// @Router /api/user-groups [post]
func (ugc *UserGroupController) create(c *gin.Context) {
	var input dto.UserGroupCreateDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.Create(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, groupDto)
}

// update godoc
// @Summary Update user group
// @Description Update an existing user group by ID
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Param userGroup body dto.UserGroupCreateDto true "User group information"
// @Success 200 {object} dto.UserGroupDtoWithUsers "Updated user group"
// @Router /api/user-groups/{id} [put]
func (ugc *UserGroupController) update(c *gin.Context) {
	var input dto.UserGroupCreateDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groupDto)
}

// delete godoc
// @Summary Delete user group
// @Description Delete a specific user group by ID
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Success 204 "No Content"
// @Router /api/user-groups/{id} [delete]
func (ugc *UserGroupController) delete(c *gin.Context) {
	if err := ugc.UserGroupService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateUsers godoc
// @Summary Update users in a group
// @Description Update the list of users belonging to a specific user group
// @Tags User Groups
// @Accept json
// @Produce json
// @Param id path string true "User Group ID"
// @Param users body dto.UserGroupUpdateUsersDto true "List of user IDs to assign to this group"
// @Success 200 {object} dto.UserGroupDtoWithUsers
// @Router /api/user-groups/{id}/users [put]
func (ugc *UserGroupController) updateUsers(c *gin.Context) {
	var input dto.UserGroupUpdateUsersDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.UpdateUsers(c.Request.Context(), c.Param("id"), input.UserIDs)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groupDto)
}
