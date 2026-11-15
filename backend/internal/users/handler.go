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
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

type handler struct {
	appConfig appconfig.AppConfigResolver
	service   *Service
	webAuthn  PasskeyManager
}

func newHandler(appConfig appconfig.AppConfigResolver, service *Service, webAuthn PasskeyManager) *handler {
	return &handler{appConfig: appConfig, service: service, webAuthn: webAuthn}
}

func (h *handler) getUserGroupsHandler(c *gin.Context) ([]dto.UserGroupDto, error) {
	userID := c.Param("id")
	groups, err := h.service.GetUserGroups(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}

	var groupsDto []dto.UserGroupDto
	if err := dto.MapStructList(groups, &groupsDto); err != nil {
		return nil, err
	}

	return groupsDto, nil
}

func (h *handler) listUserWebauthnCredentialsHandler(c *gin.Context) ([]webauthn.WebauthnCredentialDto, error) {
	userID := c.Param("id")

	if _, err := h.service.GetUser(c.Request.Context(), userID); err != nil {
		return nil, err
	}

	credentials, err := h.webAuthn.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}

	var credentialDtos []webauthn.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &credentialDtos); err != nil {
		return nil, err
	}

	return credentialDtos, nil
}

func (h *handler) listUsersHandler(c *gin.Context) (dto.Paginated[dto.UserDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	users, pagination, err := h.service.ListUsers(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[dto.UserDto]{}, err
	}

	var usersDto []dto.UserDto
	if err := dto.MapStructList(users, &usersDto); err != nil {
		return dto.Paginated[dto.UserDto]{}, err
	}

	return dto.Paginated[dto.UserDto]{
		Data:       usersDto,
		Pagination: pagination,
	}, nil
}

func (h *handler) getUserHandler(c *gin.Context) (dto.UserDto, error) {
	user, err := h.service.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return dto.UserDto{}, err
	}

	return userDto, nil
}

func (h *handler) getCurrentUserHandler(c *gin.Context) (dto.UserDto, error) {
	user, err := h.service.GetUser(c.Request.Context(), authz.PrincipalFrom(c).UserID)
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return dto.UserDto{}, err
	}

	return userDto, nil
}

func (h *handler) deleteUserHandler(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	return h.service.DeleteUser(c.Request.Context(), dbConfig, c.Param("id"), false, auditlogs.OriginFrom(c))
}

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

	return nil
}

func (h *handler) createUserHandler(c *gin.Context, input dto.UserCreateDto) (dto.UserDto, error) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	user, err := h.service.CreateUser(c.Request.Context(), dbConfig, input, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return dto.UserDto{}, err
	}

	return userDto, nil
}

func (h *handler) updateUserHandler(c *gin.Context, update dto.Update[dto.UserCreateDto]) (dto.UserDto, error) {
	return h.updateUser(c, update, false)
}

func (h *handler) updateCurrentUserHandler(c *gin.Context, update dto.Update[dto.UserCreateDto]) (dto.UserDto, error) {
	return h.updateUser(c, update, true)
}

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

	return h.service.UpdateProfilePicture(c.Request.Context(), userID, file)
}

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

	return h.service.UpdateProfilePicture(c.Request.Context(), userID, file)
}

func (h *handler) updateUserGroups(c *gin.Context, input dto.UserUpdateUserGroupDto) (dto.UserDto, error) {
	user, err := h.service.UpdateUserGroups(c.Request.Context(), c.Param("id"), input.UserGroupIds, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return dto.UserDto{}, err
	}

	return userDto, nil
}

// updateUser is an internal helper method, not exposed as an API endpoint
func (h *handler) updateUser(c *gin.Context, update dto.Update[dto.UserCreateDto], updateOwnUser bool) (dto.UserDto, error) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.UserDto{}, fmt.Errorf("error loading app configuration: %w", err)
	}

	var userID string
	if updateOwnUser {
		userID = authz.PrincipalFrom(c).UserID
	} else {
		userID = c.Param("id")
	}

	user, err := h.service.UpdateUser(c.Request.Context(), dbConfig, userID, update, updateOwnUser, false, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.UserDto{}, err
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	return userDto, err
}

func (h *handler) resetUserProfilePictureHandler(c *gin.Context) error {
	userID := c.Param("id")

	return h.service.ResetProfilePicture(c.Request.Context(), userID)
}

func (h *handler) resetCurrentUserProfilePictureHandler(c *gin.Context) error {
	userID := authz.PrincipalFrom(c).UserID

	return h.service.ResetProfilePicture(c.Request.Context(), userID)
}
