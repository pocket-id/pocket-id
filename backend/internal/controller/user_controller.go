package controller

import (
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/time/rate"
)

// NewUserController creates a new controller for user management endpoints
// @Summary User management controller
// @Description Initializes all user-related API endpoints
// @Tags Users
func NewUserController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, userService *service.UserService, appConfigService *service.AppConfigService) {
	uc := UserController{
		userService:      userService,
		appConfigService: appConfigService,
	}

	group.GET("/users", authMiddleware.Add(), uc.listUsersHandler)
	group.GET("/users/me", authMiddleware.WithAdminNotRequired().Add(), uc.getCurrentUserHandler)
	group.GET("/users/:id", authMiddleware.Add(), uc.getUserHandler)
	group.POST("/users", authMiddleware.Add(), uc.createUserHandler)
	group.PUT("/users/:id", authMiddleware.Add(), uc.updateUserHandler)
	group.GET("/users/:id/groups", authMiddleware.Add(), uc.getUserGroupsHandler)
	group.PUT("/users/me", authMiddleware.WithAdminNotRequired().Add(), uc.updateCurrentUserHandler)
	group.DELETE("/users/:id", authMiddleware.Add(), uc.deleteUserHandler)

	group.PUT("/users/:id/user-groups", authMiddleware.Add(), uc.updateUserGroups)

	group.GET("/users/:id/profile-picture.png", uc.getUserProfilePictureHandler)

	group.PUT("/users/:id/profile-picture", authMiddleware.Add(), uc.updateUserProfilePictureHandler)
	group.PUT("/users/me/profile-picture", authMiddleware.WithAdminNotRequired().Add(), uc.updateCurrentUserProfilePictureHandler)

	group.POST("/users/me/one-time-access-token", authMiddleware.WithAdminNotRequired().Add(), uc.createOwnOneTimeAccessTokenHandler)
	group.POST("/users/:id/one-time-access-token", authMiddleware.Add(), uc.createAdminOneTimeAccessTokenHandler)
	group.POST("/users/:id/one-time-access-email", authMiddleware.Add(), uc.RequestOneTimeAccessEmailAsAdminHandler)
	group.POST("/one-time-access-token/:token", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), uc.exchangeOneTimeAccessTokenHandler)
	group.POST("/one-time-access-token/setup", uc.getSetupAccessTokenHandler)
	group.POST("/one-time-access-email", rateLimitMiddleware.Add(rate.Every(10*time.Minute), 3), uc.RequestOneTimeAccessEmailAsUnauthenticatedUserHandler)

	group.DELETE("/users/:id/profile-picture", authMiddleware.Add(), uc.resetUserProfilePictureHandler)
	group.DELETE("/users/me/profile-picture", authMiddleware.WithAdminNotRequired().Add(), uc.resetCurrentUserProfilePictureHandler)
}

type UserController struct {
	userService      *service.UserService
	appConfigService *service.AppConfigService
}

// getUserGroupsHandler godoc
// @Summary Get user groups
// @Description Retrieve all groups a specific user belongs to
// @Tags Users,User Groups
// @Param id path string true "User ID"
// @Success 200 {array} dto.UserGroupDtoWithUsers
// @Router /api/users/{id}/groups [get]
func (uc *UserController) getUserGroupsHandler(c *gin.Context) {
	userID := c.Param("id")
	groups, err := uc.userService.GetUserGroups(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var groupsDto []dto.UserGroupDtoWithUsers
	if err := dto.MapStructList(groups, &groupsDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groupsDto)
}

// listUsersHandler godoc
// @Summary List users
// @Description Get a paginated list of users with optional search and sorting
// @Tags Users
// @Param search query string false "Search term to filter users"
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("created_at")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("desc")
// @Success 200 {object} dto.Paginated[dto.UserDto]
// @Router /api/users [get]
func (uc *UserController) listUsersHandler(c *gin.Context) {
	searchTerm := c.Query("search")
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		_ = c.Error(err)
		return
	}

	users, pagination, err := uc.userService.ListUsers(c.Request.Context(), searchTerm, sortedPaginationRequest)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var usersDto []dto.UserDto
	if err := dto.MapStructList(users, &usersDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.UserDto]{
		Data:       usersDto,
		Pagination: pagination,
	})
}

// getUserHandler godoc
// @Summary Get user by ID
// @Description Retrieve detailed information about a specific user
// @Tags Users
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id} [get]
func (uc *UserController) getUserHandler(c *gin.Context) {
	user, err := uc.userService.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

// getCurrentUserHandler godoc
// @Summary Get current user
// @Description Retrieve information about the currently authenticated user
// @Tags Users
// @Success 200 {object} dto.UserDto
// @Router /api/users/me [get]
func (uc *UserController) getCurrentUserHandler(c *gin.Context) {
	user, err := uc.userService.GetUser(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

// deleteUserHandler godoc
// @Summary Delete user
// @Description Delete a specific user by ID
// @Tags Users
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Router /api/users/{id} [delete]
func (uc *UserController) deleteUserHandler(c *gin.Context) {
	if err := uc.userService.DeleteUser(c.Request.Context(), c.Param("id"), false); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// createUserHandler godoc
// @Summary Create user
// @Description Create a new user
// @Tags Users
// @Param user body dto.UserCreateDto true "User information"
// @Success 201 {object} dto.UserDto
// @Router /api/users [post]
func (uc *UserController) createUserHandler(c *gin.Context) {
	var input dto.UserCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userDto)
}

// updateUserHandler godoc
// @Summary Update user
// @Description Update an existing user by ID
// @Tags Users
// @Param id path string true "User ID"
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id} [put]
func (uc *UserController) updateUserHandler(c *gin.Context) {
	uc.updateUser(c, false)
}

// updateCurrentUserHandler godoc
// @Summary Update current user
// @Description Update the currently authenticated user's information
// @Tags Users
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/users/me [put]
func (uc *UserController) updateCurrentUserHandler(c *gin.Context) {
	if !uc.appConfigService.GetDbConfig().AllowOwnAccountEdit.IsTrue() {
		_ = c.Error(&common.AccountEditNotAllowedError{})
		return
	}
	uc.updateUser(c, true)
}

// getUserProfilePictureHandler godoc
// @Summary Get user profile picture
// @Description Retrieve a specific user's profile picture
// @Tags Users
// @Produce image/png
// @Param id path string true "User ID"
// @Success 200 {file} binary "PNG image"
// @Router /api/users/{id}/profile-picture.png [get]
func (uc *UserController) getUserProfilePictureHandler(c *gin.Context) {
	userID := c.Param("id")

	picture, size, err := uc.userService.GetProfilePicture(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if picture != nil {
		defer picture.Close()
	}

	_, ok := c.GetQuery("skipCache")
	if !ok {
		c.Header("Cache-Control", "public, max-age=900")
	}

	c.DataFromReader(http.StatusOK, size, "image/png", picture, nil)
}

// updateUserProfilePictureHandler godoc
// @Summary Update user profile picture
// @Description Update a specific user's profile picture
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "User ID"
// @Param file formData file true "Profile picture image file (PNG, JPG, or JPEG)"
// @Success 204 "No Content"
// @Router /api/users/{id}/profile-picture [put]
func (uc *UserController) updateUserProfilePictureHandler(c *gin.Context) {
	userID := c.Param("id")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(err)
		return
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(userID, file); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateCurrentUserProfilePictureHandler godoc
// @Summary Update current user's profile picture
// @Description Update the currently authenticated user's profile picture
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Profile picture image file (PNG, JPG, or JPEG)"
// @Success 204 "No Content"
// @Router /api/users/me/profile-picture [put]
func (uc *UserController) updateCurrentUserProfilePictureHandler(c *gin.Context) {
	userID := c.GetString("userID")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(err)
		return
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(userID, file); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (uc *UserController) createOneTimeAccessTokenHandler(c *gin.Context, own bool) {
	var input dto.OneTimeAccessTokenCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	if own {
		input.UserID = c.GetString("userID")
	}
	token, err := uc.userService.CreateOneTimeAccessToken(c.Request.Context(), input.UserID, input.ExpiresAt)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// createOwnOneTimeAccessTokenHandler godoc
// @Summary Create one-time access token for current user
// @Description Generate a one-time access token for the currently authenticated user
// @Tags Users
// @Param id path string true "User ID"
// @Param body body dto.OneTimeAccessTokenCreateDto true "Token options"
// @Success 201 {object} object "{ \"token\": \"string\" }"
// @Router /api/users/{id}/one-time-access-token [post]
func (uc *UserController) createOwnOneTimeAccessTokenHandler(c *gin.Context) {
	uc.createOneTimeAccessTokenHandler(c, true)
}

// createAdminOneTimeAccessTokenHandler godoc
// @Summary Create one-time access token for user (admin)
// @Description Generate a one-time access token for a specific user (admin only)
// @Tags Users
// @Param id path string true "User ID"
// @Param body body dto.OneTimeAccessTokenCreateDto true "Token options"
// @Success 201 {object} object "{ \"token\": \"string\" }"
// @Router /api/users/{id}/one-time-access-token [post]
func (uc *UserController) createAdminOneTimeAccessTokenHandler(c *gin.Context) {
	uc.createOneTimeAccessTokenHandler(c, false)
}

// RequestOneTimeAccessEmailAsUnauthenticatedUserHandler godoc
// @Summary Request one-time access email
// @Description Request a one-time access email for unauthenticated users
// @Tags Users
// @Accept json
// @Produce json
// @Param body body dto.OneTimeAccessEmailAsUnauthenticatedUserDto true "Email request information"
// @Success 204 "No Content"
// @Router /api/one-time-access-email [post]
func (uc *UserController) RequestOneTimeAccessEmailAsUnauthenticatedUserHandler(c *gin.Context) {
	var input dto.OneTimeAccessEmailAsUnauthenticatedUserDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	err := uc.userService.RequestOneTimeAccessEmailAsUnauthenticatedUser(c.Request.Context(), input.Email, input.RedirectPath)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RequestOneTimeAccessEmailAsAdminHandler godoc
// @Summary Request one-time access email (admin)
// @Description Request a one-time access email for a specific user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body dto.OneTimeAccessEmailAsAdminDto true "Email request options"
// @Success 204 "No Content"
// @Router /api/users/{id}/one-time-access-email [post]
func (uc *UserController) RequestOneTimeAccessEmailAsAdminHandler(c *gin.Context) {
	var input dto.OneTimeAccessEmailAsAdminDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	userID := c.Param("id")

	err := uc.userService.RequestOneTimeAccessEmailAsAdmin(c.Request.Context(), userID, input.ExpiresAt)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// exchangeOneTimeAccessTokenHandler godoc
// @Summary Exchange one-time access token
// @Description Exchange a one-time access token for a session token
// @Tags Users
// @Param token path string true "One-time access token"
// @Success 200 {object} dto.UserDto
// @Router /api/one-time-access-token/{token} [post]
func (uc *UserController) exchangeOneTimeAccessTokenHandler(c *gin.Context) {
	user, token, err := uc.userService.ExchangeOneTimeAccessToken(c.Request.Context(), c.Param("token"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(uc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

// getSetupAccessTokenHandler godoc
// @Summary Setup initial admin
// @Description Generate setup access token for initial admin user configuration
// @Tags Users
// @Success 200 {object} dto.UserDto
// @Router /api/one-time-access-token/setup [post]
func (uc *UserController) getSetupAccessTokenHandler(c *gin.Context) {
	user, token, err := uc.userService.SetupInitialAdmin(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(uc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

// updateUserGroups godoc
// @Summary Update user groups
// @Description Update the groups a specific user belongs to
// @Tags Users
// @Param id path string true "User ID"
// @Param groups body dto.UserUpdateUserGroupDto true "User group IDs"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id}/user-groups [put]
func (uc *UserController) updateUserGroups(c *gin.Context) {
	var input dto.UserUpdateUserGroupDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	user, err := uc.userService.UpdateUserGroups(c.Request.Context(), c.Param("id"), input.UserGroupIds)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

// updateUser is an internal helper method, not exposed as an API endpoint
func (uc *UserController) updateUser(c *gin.Context, updateOwnUser bool) {
	var input dto.UserCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	var userID string
	if updateOwnUser {
		userID = c.GetString("userID")
	} else {
		userID = c.Param("id")
	}

	user, err := uc.userService.UpdateUser(c.Request.Context(), userID, input, updateOwnUser, false)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

// resetUserProfilePictureHandler godoc
// @Summary Reset user profile picture
// @Description Reset a specific user's profile picture to the default
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Router /api/users/{id}/profile-picture [delete]
func (uc *UserController) resetUserProfilePictureHandler(c *gin.Context) {
	userID := c.Param("id")

	if err := uc.userService.ResetProfilePicture(userID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// resetCurrentUserProfilePictureHandler godoc
// @Summary Reset current user's profile picture
// @Description Reset the currently authenticated user's profile picture to the default
// @Tags Users
// @Produce json
// @Success 204 "No Content"
// @Router /api/users/me/profile-picture [delete]
func (uc *UserController) resetCurrentUserProfilePictureHandler(c *gin.Context) {
	userID := c.GetString("userID")

	if err := uc.userService.ResetProfilePicture(userID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
