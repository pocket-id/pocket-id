package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

type userListInput struct {
	utils.ListRequestOptions
	Search string `query:"search" required:"false"`
}

type userIDInput struct {
	ID string `path:"id"`
}

type userCredentialIDInput struct {
	ID           string `path:"id"`
	CredentialID string `path:"credentialId"`
}

type userCreateInput struct {
	Body dto.UserCreateDto
}

type userUpdateInput struct {
	ID   string `path:"id"`
	Body dto.UserCreateDto
}

type userGroupsInput struct {
	ID   string `path:"id"`
	Body dto.UserUpdateUserGroupDto
}

type userPictureUploadForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type userPictureUploadInput struct {
	ID      string `path:"id"`
	RawBody huma.MultipartFormFiles[userPictureUploadForm]
}

type currentUserPictureUploadInput struct {
	RawBody huma.MultipartFormFiles[userPictureUploadForm]
}

type emailVerificationInput struct {
	Body dto.EmailVerificationDto
}

type userPictureOutput struct {
	ContentType   string `header:"Content-Type"`
	ContentLength int64  `header:"Content-Length"`
	CacheControl  string `header:"Cache-Control"`
	Body          func(huma.Context)
}

// NewUserController registers user management endpoints
func NewUserController(api huma.API, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, appConfigService *appconfig.AppConfigService, userService *service.UserService, webAuthnService *webauthn.Module) {
	controller := &UserController{appConfigService: appConfigService, userService: userService, webAuthnService: webAuthnService}
	adminAuth := authMiddleware.Huma(api)
	userAuth := authMiddleware.WithAdminNotRequired().Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Summary:     "List users",
		Tags:        []string{"Users"},
	}, controller.listUsersHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-current-user",
		Method:      http.MethodGet,
		Path:        "/api/users/me",
		Summary:     "Get current user",
		Tags:        []string{"Users"},
	}, controller.getCurrentUserHandler, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/api/users/{id}",
		Summary:     "Get user by ID",
		Tags:        []string{"Users"},
	}, controller.getUserHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/api/users",
		Summary:       "Create user",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
	}, controller.createUserHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        "/api/users/{id}",
		Summary:     "Update user",
		Tags:        []string{"Users"},
	}, controller.updateUserHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-current-user",
		Method:      http.MethodPut,
		Path:        "/api/users/me",
		Summary:     "Update current user",
		Tags:        []string{"Users"},
	}, controller.updateCurrentUserHandler, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-user-groups",
		Method:      http.MethodGet,
		Path:        "/api/users/{id}/groups",
		Summary:     "Get user groups",
		Tags:        []string{"Users"},
	}, controller.getUserGroupsHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-user-webauthn-credentials",
		Method:      http.MethodGet,
		Path:        "/api/users/{id}/webauthn-credentials",
		Summary:     "List user passkeys",
		Tags:        []string{"Users"},
	}, controller.listUserWebauthnCredentialsHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-user",
		Method:        http.MethodDelete,
		Path:          "/api/users/{id}",
		Summary:       "Delete user",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.deleteUserHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-user-webauthn-credential",
		Method:        http.MethodDelete,
		Path:          "/api/users/{id}/webauthn-credentials/{credentialId}",
		Summary:       "Delete user passkey",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.deleteUserWebauthnCredentialHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-groups",
		Method:      http.MethodPut,
		Path:        "/api/users/{id}/user-groups",
		Summary:     "Update user groups",
		Tags:        []string{"Users"},
	}, controller.updateUserGroups, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-user-profile-picture",
		Method:      http.MethodGet,
		Path:        "/api/users/{id}/profile-picture.png",
		Summary:     "Get user profile picture",
		Tags:        []string{"Users"},
	}, controller.getUserProfilePictureHandler)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-user-profile-picture",
		Method:        http.MethodPut,
		Path:          "/api/users/{id}/profile-picture",
		Summary:       "Update user profile picture",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateUserProfilePictureHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "update-current-user-profile-picture",
		Method:        http.MethodPut,
		Path:          "/api/users/me/profile-picture",
		Summary:       "Update current user profile picture",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.updateCurrentUserProfilePictureHandler, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "reset-user-profile-picture",
		Method:        http.MethodDelete,
		Path:          "/api/users/{id}/profile-picture",
		Summary:       "Reset user profile picture",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.resetUserProfilePictureHandler, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "reset-current-user-profile-picture",
		Method:        http.MethodDelete,
		Path:          "/api/users/me/profile-picture",
		Summary:       "Reset current user profile picture",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.resetCurrentUserProfilePictureHandler, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "send-email-verification",
		Method:        http.MethodPost,
		Path:          "/api/users/me/send-email-verification",
		Summary:       "Send email verification",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.sendEmailVerificationHandler, httpapi.WithMiddleware(rateLimitMiddleware.Huma(api, middleware.RateLimitSendEmailVerification)), userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "verify-email",
		Method:        http.MethodPost,
		Path:          "/api/users/me/verify-email",
		Summary:       "Verify email",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, controller.verifyEmailHandler, httpapi.WithMiddleware(rateLimitMiddleware.Huma(api, middleware.RateLimitVerifyEmail)), userAuth)
}

type UserController struct {
	appConfigService *appconfig.AppConfigService
	userService      *service.UserService
	webAuthnService  *webauthn.Module
}

func (uc *UserController) getUserGroupsHandler(ctx context.Context, input *userIDInput) (*httpapi.BodyOutput[[]dto.UserGroupDto], error) {
	groups, err := uc.userService.GetUserGroups(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	var output []dto.UserGroupDto
	if err := dto.MapStructList(groups, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.UserGroupDto]{Body: output}, nil
}

func (uc *UserController) listUserWebauthnCredentialsHandler(ctx context.Context, input *userIDInput) (*httpapi.BodyOutput[[]dto.WebauthnCredentialDto], error) {
	if _, err := uc.userService.GetUser(ctx, input.ID); err != nil {
		return nil, err
	}
	credentials, err := uc.webAuthnService.ListCredentials(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	var output []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.WebauthnCredentialDto]{Body: output}, nil
}

func (uc *UserController) listUsersHandler(ctx context.Context, input *userListInput) (*httpapi.BodyOutput[dto.Paginated[dto.UserDto]], error) {
	users, pagination, err := uc.userService.ListUsers(ctx, input.Search, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}
	var output []dto.UserDto
	if err := dto.MapStructList(users, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.UserDto]]{Body: dto.Paginated[dto.UserDto]{Data: output, Pagination: pagination}}, nil
}

func (uc *UserController) getUserHandler(ctx context.Context, input *userIDInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	user, err := uc.userService.GetUser(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return mapUser(user)
}

func (uc *UserController) getCurrentUserHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	user, err := uc.userService.GetUser(ctx, httpapi.UserID(ctx))
	if err != nil {
		return nil, err
	}
	return mapUser(user)
}

func (uc *UserController) deleteUserHandler(ctx context.Context, input *userIDInput) (*httpapi.EmptyOutput, error) {
	dbConfig, err := uc.appConfigService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := uc.userService.DeleteUser(ctx, dbConfig, input.ID, false); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) deleteUserWebauthnCredentialHandler(ctx context.Context, input *userCredentialIDInput) (*httpapi.EmptyOutput, error) {
	if err := uc.webAuthnService.DeleteCredential(ctx, input.ID, input.CredentialID, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx), httpapi.UserID(ctx)); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) createUserHandler(ctx context.Context, input *userCreateInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	dbConfig, err := uc.appConfigService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	user, err := uc.userService.CreateUser(ctx, dbConfig, input.Body)
	if err != nil {
		return nil, err
	}
	return mapUser(user)
}

func (uc *UserController) updateUserHandler(ctx context.Context, input *userUpdateInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	return uc.updateUser(ctx, input.ID, input.Body, false)
}

func (uc *UserController) updateCurrentUserHandler(ctx context.Context, input *userCreateInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	return uc.updateUser(ctx, httpapi.UserID(ctx), input.Body, true)
}

func (uc *UserController) updateUser(ctx context.Context, userID string, input dto.UserCreateDto, ownUser bool) (*httpapi.BodyOutput[dto.UserDto], error) {
	dbConfig, err := uc.appConfigService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	user, err := uc.userService.UpdateUser(ctx, dbConfig, userID, input, ownUser, false)
	if err != nil {
		return nil, err
	}
	return mapUser(user)
}

func (uc *UserController) getUserProfilePictureHandler(ctx context.Context, input *userIDInput) (*userPictureOutput, error) {
	picture, size, err := uc.userService.GetProfilePicture(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	cacheControl := ""
	if !httpapi.QueryPresent(ctx, "skipCache") {
		cacheControl = utils.CacheControlValue(15*time.Minute, time.Hour)
	}
	return &userPictureOutput{
		ContentType:   "image/png",
		ContentLength: size,
		CacheControl:  cacheControl,
		Body: func(streamCtx huma.Context) {
			if picture != nil {
				defer picture.Close()
				_, _ = io.Copy(streamCtx.BodyWriter(), picture)
			}
		},
	}, nil
}

func (uc *UserController) updateUserProfilePictureHandler(ctx context.Context, input *userPictureUploadInput) (*httpapi.EmptyOutput, error) {
	return uc.updateProfilePicture(ctx, input.ID, input.RawBody.Data().File)
}

func (uc *UserController) updateCurrentUserProfilePictureHandler(ctx context.Context, input *currentUserPictureUploadInput) (*httpapi.EmptyOutput, error) {
	return uc.updateProfilePicture(ctx, httpapi.UserID(ctx), input.RawBody.Data().File)
}

func (uc *UserController) updateProfilePicture(ctx context.Context, userID string, file huma.FormFile) (*httpapi.EmptyOutput, error) {
	defer file.Close()
	if err := uc.userService.UpdateProfilePicture(ctx, userID, file); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) updateUserGroups(ctx context.Context, input *userGroupsInput) (*httpapi.BodyOutput[dto.UserDto], error) {
	user, err := uc.userService.UpdateUserGroups(ctx, input.ID, input.Body.UserGroupIds)
	if err != nil {
		return nil, err
	}
	return mapUser(user)
}

func (uc *UserController) resetUserProfilePictureHandler(ctx context.Context, input *userIDInput) (*httpapi.EmptyOutput, error) {
	if err := uc.userService.ResetProfilePicture(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) resetCurrentUserProfilePictureHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	if err := uc.userService.ResetProfilePicture(ctx, httpapi.UserID(ctx)); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) sendEmailVerificationHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.EmptyOutput, error) {
	dbConfig, err := uc.appConfigService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := uc.userService.SendEmailVerification(ctx, dbConfig, httpapi.UserID(ctx)); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) verifyEmailHandler(ctx context.Context, input *emailVerificationInput) (*httpapi.EmptyOutput, error) {
	if err := uc.userService.VerifyEmail(ctx, httpapi.UserID(ctx), input.Body.Token); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func mapUser(user model.User) (*httpapi.BodyOutput[dto.UserDto], error) {
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.UserDto]{Body: output}, nil
}
