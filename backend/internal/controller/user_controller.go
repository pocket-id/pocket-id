package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

// NewUserController creates a new controller for user management endpoints
// @Summary User management controller
// @Description Initializes all user-related API endpoints
// @Tags Users
func NewUserController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, appConfigService *appconfig.AppConfigService, userService *service.UserService, webAuthnService *webauthn.Module) {
	uc := UserController{
		appConfigService: appConfigService,
		userService:      userService,
		webAuthnService:  webAuthnService,
	}

	group.GET("/users", authMiddleware.Add(), httpserver.Handle(uc.listUsersHandler))
	group.GET("/users/me", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(uc.getCurrentUserHandler))
	group.GET("/users/:id", authMiddleware.Add(), httpserver.Handle(uc.getUserHandler))
	group.POST("/users", authMiddleware.Add(), httpserver.Handle(uc.createUserHandler))
	group.PUT("/users/:id", authMiddleware.Add(), httpserver.Handle(uc.updateUserHandler))
	group.GET("/users/:id/groups", authMiddleware.Add(), httpserver.Handle(uc.getUserGroupsHandler))
	group.GET("/users/:id/webauthn-credentials", authMiddleware.Add(), httpserver.Handle(uc.listUserWebauthnCredentialsHandler))
	group.PUT("/users/me", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(uc.updateCurrentUserHandler))
	group.DELETE("/users/:id", authMiddleware.Add(), httpserver.Handle(uc.deleteUserHandler))
	group.DELETE("/users/:id/webauthn-credentials/:credentialId", authMiddleware.Add(), httpserver.Handle(uc.deleteUserWebauthnCredentialHandler))

	group.PUT("/users/:id/user-groups", authMiddleware.Add(), httpserver.Handle(uc.updateUserGroups))

	group.GET("/users/:id/profile-picture.png", httpserver.Handle(uc.getUserProfilePictureHandler))

	group.PUT("/users/:id/profile-picture", authMiddleware.Add(), httpserver.Handle(uc.updateUserProfilePictureHandler))
	group.PUT("/users/me/profile-picture", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(uc.updateCurrentUserProfilePictureHandler))

	group.DELETE("/users/:id/profile-picture", authMiddleware.Add(), httpserver.Handle(uc.resetUserProfilePictureHandler))
	group.DELETE("/users/me/profile-picture", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(uc.resetCurrentUserProfilePictureHandler))
}

type UserController struct {
	appConfigService *appconfig.AppConfigService
	userService      *service.UserService
	webAuthnService  *webauthn.Module
}

// getUserGroupsHandler godoc
// @Summary Get user groups
// @Description Retrieve all groups a specific user belongs to
// @Tags Users,User Groups
// @Param id path string true "User ID"
// @Success 200 {array} dto.UserGroupDto
// @Router /api/users/{id}/groups [get]
func (uc *UserController) getUserGroupsHandler(c *gin.Context) error {
	userID := c.Param("id")
	groups, err := uc.userService.GetUserGroups(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	var groupsDto []dto.UserGroupDto
	if err := dto.MapStructList(groups, &groupsDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, groupsDto)
	return nil
}

// listUserWebauthnCredentialsHandler godoc
// @Summary List user passkeys
// @Description Retrieve all WebAuthn credentials for a specific user
// @Tags Users
// @Param id path string true "User ID"
// @Success 200 {array} dto.WebauthnCredentialDto
// @Router /api/users/{id}/webauthn-credentials [get]
func (uc *UserController) listUserWebauthnCredentialsHandler(c *gin.Context) error {
	userID := c.Param("id")

	if _, err := uc.userService.GetUser(c.Request.Context(), userID); err != nil {
		return err
	}

	credentials, err := uc.webAuthnService.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	var credentialDtos []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &credentialDtos); err != nil {
		return err
	}

	c.JSON(http.StatusOK, credentialDtos)
	return nil
}

// listUsersHandler godoc
// @Summary List users
// @Description Get a paginated list of users with optional search and sorting
// @Tags Users
// @Param search query string false "Search term to filter users"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.UserDto]
// @Router /api/users [get]
func (uc *UserController) listUsersHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	users, pagination, err := uc.userService.ListUsers(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	var usersDto []dto.UserDto
	if err := dto.MapStructList(users, &usersDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.UserDto]{
		Data:       usersDto,
		Pagination: pagination,
	})
	return nil
}

// getUserHandler godoc
// @Summary Get user by ID
// @Description Retrieve detailed information about a specific user
// @Tags Users
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id} [get]
func (uc *UserController) getUserHandler(c *gin.Context) error {
	user, err := uc.userService.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, userDto)
	return nil
}

// getCurrentUserHandler godoc
// @Summary Get current user
// @Description Retrieve information about the currently authenticated user
// @Tags Users
// @Success 200 {object} dto.UserDto
// @Router /api/users/me [get]
func (uc *UserController) getCurrentUserHandler(c *gin.Context) error {
	user, err := uc.userService.GetUser(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, userDto)
	return nil
}

// deleteUserHandler godoc
// @Summary Delete user
// @Description Delete a specific user by ID
// @Tags Users
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Router /api/users/{id} [delete]
func (uc *UserController) deleteUserHandler(c *gin.Context) error {
	dbConfig, err := uc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := uc.userService.DeleteUser(c.Request.Context(), dbConfig, c.Param("id"), false); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// deleteUserWebauthnCredentialHandler godoc
// @Summary Delete user passkey
// @Description Delete a specific WebAuthn credential for a user
// @Tags Users
// @Param id path string true "User ID"
// @Param credentialId path string true "Credential ID"
// @Success 204 "No Content"
// @Router /api/users/{id}/webauthn-credentials/{credentialId} [delete]
func (uc *UserController) deleteUserWebauthnCredentialHandler(c *gin.Context) error {
	err := uc.webAuthnService.DeleteCredential(
		c.Request.Context(),
		c.Param("id"),
		c.Param("credentialId"),
		c.ClientIP(),
		c.Request.UserAgent(),
		c.GetString("userID"),
	)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// createUserHandler godoc
// @Summary Create user
// @Description Create a new user
// @Tags Users
// @Param user body dto.UserCreateDto true "User information"
// @Success 201 {object} dto.UserDto
// @Router /api/users [post]
func (uc *UserController) createUserHandler(c *gin.Context) error {
	dbConfig, err := uc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input dto.UserCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), dbConfig, input)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	c.JSON(http.StatusCreated, userDto)
	return nil
}

// updateUserHandler godoc
// @Summary Update user
// @Description Update an existing user by ID
// @Tags Users
// @Param id path string true "User ID"
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id} [put]
func (uc *UserController) updateUserHandler(c *gin.Context) error {
	return uc.updateUser(c, false)
}

// updateCurrentUserHandler godoc
// @Summary Update current user
// @Description Update the currently authenticated user's information
// @Tags Users
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/users/me [put]
func (uc *UserController) updateCurrentUserHandler(c *gin.Context) error {
	return uc.updateUser(c, true)
}

// getUserProfilePictureHandler godoc
// @Summary Get user profile picture
// @Description Retrieve a specific user's profile picture
// @Tags Users
// @Produce image/png
// @Param id path string true "User ID"
// @Success 200 {file} binary "PNG image"
// @Router /api/users/{id}/profile-picture.png [get]
func (uc *UserController) getUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")

	picture, size, err := uc.userService.GetProfilePicture(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	if picture != nil {
		defer picture.Close()
	}

	utils.SetCacheControlHeader(c, 15*time.Minute, 1*time.Hour)

	c.DataFromReader(http.StatusOK, size, "image/png", picture, nil)
	return nil
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
func (uc *UserController) updateUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")
	fileHeader, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(c.Request.Context(), userID, file); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
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
func (uc *UserController) updateCurrentUserProfilePictureHandler(c *gin.Context) error {
	userID := c.GetString("userID")
	fileHeader, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(c.Request.Context(), userID, file); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// updateUserGroups godoc
// @Summary Update user groups
// @Description Update the groups a specific user belongs to
// @Tags Users
// @Param id path string true "User ID"
// @Param groups body dto.UserUpdateUserGroupDto true "User group IDs"
// @Success 200 {object} dto.UserDto
// @Router /api/users/{id}/user-groups [put]
func (uc *UserController) updateUserGroups(c *gin.Context) error {
	var input dto.UserUpdateUserGroupDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	user, err := uc.userService.UpdateUserGroups(c.Request.Context(), c.Param("id"), input.UserGroupIds)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, userDto)
	return nil
}

// updateUser is an internal helper method, not exposed as an API endpoint
func (uc *UserController) updateUser(c *gin.Context, updateOwnUser bool) error {
	dbConfig, err := uc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input dto.UserCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	var userID string
	if updateOwnUser {
		userID = c.GetString("userID")
	} else {
		userID = c.Param("id")
	}

	user, err := uc.userService.UpdateUser(c.Request.Context(), dbConfig, userID, input, updateOwnUser, false)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, userDto)
	return nil
}

// resetUserProfilePictureHandler godoc
// @Summary Reset user profile picture
// @Description Reset a specific user's profile picture to the default
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Router /api/users/{id}/profile-picture [delete]
func (uc *UserController) resetUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")

	if err := uc.userService.ResetProfilePicture(c.Request.Context(), userID); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// resetCurrentUserProfilePictureHandler godoc
// @Summary Reset current user's profile picture
// @Description Reset the currently authenticated user's profile picture to the default
// @Tags Users
// @Produce json
// @Success 204 "No Content"
// @Router /api/users/me/profile-picture [delete]
func (uc *UserController) resetCurrentUserProfilePictureHandler(c *gin.Context) error {
	userID := c.GetString("userID")

	if err := uc.userService.ResetProfilePicture(c.Request.Context(), userID); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
