package users

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	appConfig appconfig.AppConfigResolver
	service   *Service
	webAuthn  PasskeyManager
}

func newHandler(appConfig appconfig.AppConfigResolver, service *Service, webAuthn PasskeyManager) *handler {
	return &handler{appConfig: appConfig, service: service, webAuthn: webAuthn}
}

// getUserGroupsHandler godoc
// @Summary Get user groups
// @Description Retrieve all groups a specific user belongs to
// @Tags Users,User Groups
// @Param id path string true "User ID"
// @Success 200 {array} dto.UserGroupDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/groups [get]
func (h *handler) getUserGroupsHandler(c *gin.Context) error {
	userID := c.Param("id")
	groups, err := h.service.GetUserGroups(c.Request.Context(), userID)
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/webauthn-credentials [get]
func (h *handler) listUserWebauthnCredentialsHandler(c *gin.Context) error {
	userID := c.Param("id")

	if _, err := h.service.GetUser(c.Request.Context(), userID); err != nil {
		return err
	}

	credentials, err := h.webAuthn.ListCredentials(c.Request.Context(), userID)
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users [get]
func (h *handler) listUsersHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	users, pagination, err := h.service.ListUsers(c.Request.Context(), searchTerm, listRequestOptions)
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id} [get]
func (h *handler) getUserHandler(c *gin.Context) error {
	user, err := h.service.GetUser(c.Request.Context(), c.Param("id"))
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/me [get]
func (h *handler) getCurrentUserHandler(c *gin.Context) error {
	user, err := h.service.GetUser(c.Request.Context(), authz.PrincipalFrom(c).UserID)
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id} [delete]
func (h *handler) deleteUserHandler(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := h.service.DeleteUser(c.Request.Context(), dbConfig, c.Param("id"), false, auditlogs.OriginFrom(c)); err != nil {
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/webauthn-credentials/{credentialId} [delete]
func (h *handler) deleteUserWebauthnCredentialHandler(c *gin.Context) error {
	err := h.webAuthn.DeleteCredential(
		c.Request.Context(),
		c.Param("id"),
		c.Param("credentialId"),
		auditlogs.OriginFrom(c),
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users [post]
func (h *handler) createUserHandler(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input dto.UserCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	user, err := h.service.CreateUser(c.Request.Context(), dbConfig, input, auditlogs.OriginFrom(c))
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
// @Description Update an existing user by ID, where PATCH changes only the fields in the body
// @Tags Users
// @Param id path string true "User ID"
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id} [put]
// @Router /api/users/{id} [patch]
func (h *handler) updateUserHandler(c *gin.Context) error {
	return h.updateUser(c, false)
}

// updateCurrentUserHandler godoc
// @Summary Update current user
// @Description Update the currently authenticated user's information, where PATCH changes only the fields in the body
// @Tags Users
// @Param user body dto.UserCreateDto true "User information"
// @Success 200 {object} dto.UserDto
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/me [put]
// @Router /api/users/me [patch]
func (h *handler) updateCurrentUserHandler(c *gin.Context) error {
	return h.updateUser(c, true)
}

// getUserProfilePictureHandler godoc
// @Summary Get user profile picture
// @Description Retrieve a specific user's profile picture
// @Tags Users
// @Produce image/png
// @Param id path string true "User ID"
// @Success 200 {file} binary "PNG image"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/profile-picture.png [get]
func (h *handler) getUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")

	picture, size, err := h.service.GetProfilePicture(c.Request.Context(), userID)
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/profile-picture [put]
func (h *handler) updateUserProfilePictureHandler(c *gin.Context) error {
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

	if err := h.service.UpdateProfilePicture(c.Request.Context(), userID, file); err != nil {
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/me/profile-picture [put]
func (h *handler) updateCurrentUserProfilePictureHandler(c *gin.Context) error {
	userID := authz.PrincipalFrom(c).UserID
	fileHeader, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	if err := h.service.UpdateProfilePicture(c.Request.Context(), userID, file); err != nil {
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/user-groups [put]
func (h *handler) updateUserGroups(c *gin.Context) error {
	var input dto.UserUpdateUserGroupDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	user, err := h.service.UpdateUserGroups(c.Request.Context(), c.Param("id"), input.UserGroupIds, auditlogs.OriginFrom(c))
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
func (h *handler) updateUser(c *gin.Context, updateOwnUser bool) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var userID string
	if updateOwnUser {
		userID = authz.PrincipalFrom(c).UserID
	} else {
		userID = c.Param("id")
	}

	update, err := httpserver.BindUpdate[dto.UserCreateDto](c)
	if err != nil {
		return err
	}

	user, err := h.service.UpdateUser(c.Request.Context(), dbConfig, userID, update, updateOwnUser, false, auditlogs.OriginFrom(c))
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/{id}/profile-picture [delete]
func (h *handler) resetUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")

	if err := h.service.ResetProfilePicture(c.Request.Context(), userID); err != nil {
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/users/me/profile-picture [delete]
func (h *handler) resetCurrentUserProfilePictureHandler(c *gin.Context) error {
	userID := authz.PrincipalFrom(c).UserID

	if err := h.service.ResetProfilePicture(c.Request.Context(), userID); err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
