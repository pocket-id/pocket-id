package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

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

	// Map the user groups to DTOs. The user count can't be mapped directly, so we have to do it manually.
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

func (ugc *UserGroupController) delete(c *gin.Context) {
	if err := ugc.UserGroupService.Delete(c.Param("id")); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

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
