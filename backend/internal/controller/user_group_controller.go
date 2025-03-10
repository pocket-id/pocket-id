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
// @Accept json
// @Produce json
// @Param search query string false "Search term to filter user groups by name"
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("name")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} object "{ \"data\": []dto.UserGroupDtoWithUserCount, \"pagination\": utils.Pagination }"
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups [get]
func (ugc *UserGroupController) list(c *gin.Context) {
	searchTerm := c.Query("search")
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		c.Error(err)
		return
	}

	groups, pagination, err := ugc.UserGroupService.List(searchTerm, sortedPaginationRequest)
	if err != nil {
		c.Error(err)
		return
	}

	// Map the user groups to DTOs
	var groupsDto = make([]dto.UserGroupDtoWithUserCount, len(groups))
	for i, group := range groups {
		var groupDto dto.UserGroupDtoWithUserCount
		if err := dto.MapStruct(group, &groupDto); err != nil {
			c.Error(err)
			return
		}
		groupDto.UserCount, err = ugc.UserGroupService.GetUserCountOfGroup(group.ID)
		if err != nil {
			c.Error(err)
			return
		}
		groupsDto[i] = groupDto
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       groupsDto,
		"pagination": pagination,
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
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "User group not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups/{id} [get]
func (ugc *UserGroupController) get(c *gin.Context) {
	group, err := ugc.UserGroupService.Get(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		c.Error(err)
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
// @Failure 400 {object} object "Bad request or validation error"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 409 {object} object "Conflict - group name already exists"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups [post]
func (ugc *UserGroupController) create(c *gin.Context) {
	var input dto.UserGroupCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.Create(input)
	if err != nil {
		c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		c.Error(err)
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
// @Failure 400 {object} object "Bad request or validation error"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "User group not found"
// @Failure 409 {object} object "Conflict - group name already exists"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups/{id} [put]
func (ugc *UserGroupController) update(c *gin.Context) {
	var input dto.UserGroupCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.Update(c.Param("id"), input, false)
	if err != nil {
		c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		c.Error(err)
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
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "User group not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups/{id} [delete]
func (ugc *UserGroupController) delete(c *gin.Context) {
	if err := ugc.UserGroupService.Delete(c.Param("id")); err != nil {
		c.Error(err)
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
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "User group not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /user-groups/{id}/users [put]
func (ugc *UserGroupController) updateUsers(c *gin.Context) {
	var input dto.UserGroupUpdateUsersDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	group, err := ugc.UserGroupService.UpdateUsers(c.Param("id"), input.UserIDs)
	if err != nil {
		c.Error(err)
		return
	}

	var groupDto dto.UserGroupDtoWithUsers
	if err := dto.MapStruct(group, &groupDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groupDto)
}
