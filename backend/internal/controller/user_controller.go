package controller

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

const defaultOneTimeAccessTokenDuration = 15 * time.Minute

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

type oneTimeAccessOwnInput struct {
	Body dto.OneTimeAccessTokenCreateDto
}

type oneTimeAccessAdminInput struct {
	ID   string `path:"id"`
	Body dto.OneTimeAccessTokenCreateDto
}

type oneTimeAccessEmailAdminInput struct {
	ID   string `path:"id"`
	Body dto.OneTimeAccessEmailAsAdminDto
}

type oneTimeAccessEmailInput struct {
	Body dto.OneTimeAccessEmailAsUnauthenticatedUserDto
}

type oneTimeAccessExchangeInput struct {
	Token string `path:"token"`
}

type emailVerificationInput struct {
	Body dto.EmailVerificationDto
}

type userCookieOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      dto.UserDto
}

type userPictureOutput struct {
	ContentType   string `header:"Content-Type"`
	ContentLength int64  `header:"Content-Length"`
	CacheControl  string `header:"Cache-Control"`
	Body          func(huma.Context)
}

// NewUserController registers user management endpoints
func NewUserController(api huma.API, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, userService *service.UserService, oneTimeAccessService *service.OneTimeAccessService, webAuthnService *webauthn.Module, appConfigService *service.AppConfigService) {
	controller := &UserController{userService: userService, oneTimeAccessService: oneTimeAccessService, webAuthnService: webAuthnService, appConfigService: appConfigService}
	adminAuth := authMiddleware.Huma(api)
	userAuth := authMiddleware.WithAdminNotRequired().Huma(api)

	listUsers := userOperation("list-users", http.MethodGet, "/api/users", "List users")
	adminAuth(&listUsers)
	httpapi.Register(api, listUsers, controller.listUsersHandler)

	getCurrentUser := userOperation("get-current-user", http.MethodGet, "/api/users/me", "Get current user")
	userAuth(&getCurrentUser)
	httpapi.Register(api, getCurrentUser, controller.getCurrentUserHandler)

	getUser := userOperation("get-user", http.MethodGet, "/api/users/{id}", "Get user by ID")
	adminAuth(&getUser)
	httpapi.Register(api, getUser, controller.getUserHandler)

	createUser := userOperation("create-user", http.MethodPost, "/api/users", "Create user")
	createUser.DefaultStatus = http.StatusCreated
	adminAuth(&createUser)
	httpapi.Register(api, createUser, controller.createUserHandler)

	updateUser := userOperation("update-user", http.MethodPut, "/api/users/{id}", "Update user")
	adminAuth(&updateUser)
	httpapi.Register(api, updateUser, controller.updateUserHandler)

	updateCurrentUser := userOperation("update-current-user", http.MethodPut, "/api/users/me", "Update current user")
	userAuth(&updateCurrentUser)
	httpapi.Register(api, updateCurrentUser, controller.updateCurrentUserHandler)

	getGroups := userOperation("get-user-groups", http.MethodGet, "/api/users/{id}/groups", "Get user groups")
	adminAuth(&getGroups)
	httpapi.Register(api, getGroups, controller.getUserGroupsHandler)

	listCredentials := userOperation("list-user-webauthn-credentials", http.MethodGet, "/api/users/{id}/webauthn-credentials", "List user passkeys")
	adminAuth(&listCredentials)
	httpapi.Register(api, listCredentials, controller.listUserWebauthnCredentialsHandler)

	deleteUser := userOperation("delete-user", http.MethodDelete, "/api/users/{id}", "Delete user")
	deleteUser.DefaultStatus = http.StatusNoContent
	adminAuth(&deleteUser)
	httpapi.Register(api, deleteUser, controller.deleteUserHandler)

	deleteCredential := userOperation("delete-user-webauthn-credential", http.MethodDelete, "/api/users/{id}/webauthn-credentials/{credentialId}", "Delete user passkey")
	deleteCredential.DefaultStatus = http.StatusNoContent
	adminAuth(&deleteCredential)
	httpapi.Register(api, deleteCredential, controller.deleteUserWebauthnCredentialHandler)

	updateGroups := userOperation("update-user-groups", http.MethodPut, "/api/users/{id}/user-groups", "Update user groups")
	adminAuth(&updateGroups)
	httpapi.Register(api, updateGroups, controller.updateUserGroups)

	httpapi.Register(api, userOperation("get-user-profile-picture", http.MethodGet, "/api/users/{id}/profile-picture.png", "Get user profile picture"), controller.getUserProfilePictureHandler)

	updatePicture := userOperation("update-user-profile-picture", http.MethodPut, "/api/users/{id}/profile-picture", "Update user profile picture")
	updatePicture.DefaultStatus = http.StatusNoContent
	adminAuth(&updatePicture)
	httpapi.Register(api, updatePicture, controller.updateUserProfilePictureHandler)

	updateCurrentPicture := userOperation("update-current-user-profile-picture", http.MethodPut, "/api/users/me/profile-picture", "Update current user profile picture")
	updateCurrentPicture.DefaultStatus = http.StatusNoContent
	userAuth(&updateCurrentPicture)
	httpapi.Register(api, updateCurrentPicture, controller.updateCurrentUserProfilePictureHandler)

	createOwnToken := userOperation("create-own-one-time-access-token", http.MethodPost, "/api/users/me/one-time-access-token", "Create one-time access token for current user")
	createOwnToken.DefaultStatus = http.StatusCreated
	userAuth(&createOwnToken)
	httpapi.Register(api, createOwnToken, controller.createOwnOneTimeAccessTokenHandler)

	createAdminToken := userOperation("create-user-one-time-access-token", http.MethodPost, "/api/users/{id}/one-time-access-token", "Create one-time access token for user")
	createAdminToken.DefaultStatus = http.StatusCreated
	adminAuth(&createAdminToken)
	httpapi.Register(api, createAdminToken, controller.createAdminOneTimeAccessTokenHandler)

	adminEmail := userOperation("request-user-one-time-access-email", http.MethodPost, "/api/users/{id}/one-time-access-email", "Request one-time access email for user")
	adminEmail.DefaultStatus = http.StatusNoContent
	adminAuth(&adminEmail)
	httpapi.Register(api, adminEmail, controller.requestOneTimeAccessEmailAsAdminHandler)

	exchangeToken := userOperation("exchange-one-time-access-token", http.MethodPost, "/api/one-time-access-token/{token}", "Exchange one-time access token")
	exchangeToken.Middlewares = append(exchangeToken.Middlewares, rateLimitMiddleware.Huma(api, middleware.RateLimitOneTimeAccessToken))
	httpapi.Register(api, exchangeToken, controller.exchangeOneTimeAccessTokenHandler)

	requestEmail := userOperation("request-one-time-access-email", http.MethodPost, "/api/one-time-access-email", "Request one-time access email")
	requestEmail.DefaultStatus = http.StatusNoContent
	requestEmail.Middlewares = append(requestEmail.Middlewares, rateLimitMiddleware.Huma(api, middleware.RateLimitOneTimeAccessEmail))
	httpapi.Register(api, requestEmail, controller.requestOneTimeAccessEmailAsUnauthenticatedUserHandler)

	resetPicture := userOperation("reset-user-profile-picture", http.MethodDelete, "/api/users/{id}/profile-picture", "Reset user profile picture")
	resetPicture.DefaultStatus = http.StatusNoContent
	adminAuth(&resetPicture)
	httpapi.Register(api, resetPicture, controller.resetUserProfilePictureHandler)

	resetCurrentPicture := userOperation("reset-current-user-profile-picture", http.MethodDelete, "/api/users/me/profile-picture", "Reset current user profile picture")
	resetCurrentPicture.DefaultStatus = http.StatusNoContent
	userAuth(&resetCurrentPicture)
	httpapi.Register(api, resetCurrentPicture, controller.resetCurrentUserProfilePictureHandler)

	sendVerification := userOperation("send-email-verification", http.MethodPost, "/api/users/me/send-email-verification", "Send email verification")
	sendVerification.DefaultStatus = http.StatusNoContent
	sendVerification.Middlewares = append(sendVerification.Middlewares, rateLimitMiddleware.Huma(api, middleware.RateLimitSendEmailVerification))
	userAuth(&sendVerification)
	httpapi.Register(api, sendVerification, controller.sendEmailVerificationHandler)

	verifyEmail := userOperation("verify-email", http.MethodPost, "/api/users/me/verify-email", "Verify email")
	verifyEmail.DefaultStatus = http.StatusNoContent
	verifyEmail.Middlewares = append(verifyEmail.Middlewares, rateLimitMiddleware.Huma(api, middleware.RateLimitVerifyEmail))
	userAuth(&verifyEmail)
	httpapi.Register(api, verifyEmail, controller.verifyEmailHandler)
}

func userOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"Users"}}
}

type UserController struct {
	userService          *service.UserService
	oneTimeAccessService *service.OneTimeAccessService
	webAuthnService      *webauthn.Module
	appConfigService     *service.AppConfigService
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
	if err := uc.userService.DeleteUser(ctx, input.ID, false); err != nil {
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
	user, err := uc.userService.CreateUser(ctx, input.Body)
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
	user, err := uc.userService.UpdateUser(ctx, userID, input, ownUser, false)
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

func (uc *UserController) createOwnOneTimeAccessTokenHandler(ctx context.Context, _ *oneTimeAccessOwnInput) (*httpapi.BodyOutput[map[string]string], error) {
	return uc.createOneTimeAccessToken(ctx, httpapi.UserID(ctx), defaultOneTimeAccessTokenDuration)
}

func (uc *UserController) createAdminOneTimeAccessTokenHandler(ctx context.Context, input *oneTimeAccessAdminInput) (*httpapi.BodyOutput[map[string]string], error) {
	ttl := input.Body.TTL.Duration
	if ttl <= 0 {
		ttl = defaultOneTimeAccessTokenDuration
	}
	return uc.createOneTimeAccessToken(ctx, input.ID, ttl)
}

func (uc *UserController) createOneTimeAccessToken(ctx context.Context, userID string, ttl time.Duration) (*httpapi.BodyOutput[map[string]string], error) {
	if userID == "" {
		return nil, &common.UserIdNotProvidedError{}
	}
	token, err := uc.oneTimeAccessService.CreateOneTimeAccessToken(ctx, userID, ttl)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[map[string]string]{Body: map[string]string{"token": token}}, nil
}

func (uc *UserController) requestOneTimeAccessEmailAsUnauthenticatedUserHandler(ctx context.Context, input *oneTimeAccessEmailInput) (*emptyOutputWithCookie, error) {
	deviceToken, err := uc.oneTimeAccessService.RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx, input.Body.Email, input.Body.RedirectPath)
	if err != nil {
		return nil, err
	}
	return &emptyOutputWithCookie{SetCookie: []http.Cookie{*cookie.NewDeviceTokenCookie(deviceToken)}}, nil
}

func (uc *UserController) requestOneTimeAccessEmailAsAdminHandler(ctx context.Context, input *oneTimeAccessEmailAdminInput) (*httpapi.EmptyOutput, error) {
	ttl := input.Body.TTL.Duration
	if ttl <= 0 {
		ttl = defaultOneTimeAccessTokenDuration
	}
	if err := uc.oneTimeAccessService.RequestOneTimeAccessEmailAsAdmin(ctx, input.ID, ttl); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (uc *UserController) exchangeOneTimeAccessTokenHandler(ctx context.Context, input *oneTimeAccessExchangeInput) (*userCookieOutput, error) {
	if len(input.Token) != 6 && len(input.Token) != 16 {
		return nil, &common.TokenInvalidOrExpiredError{}
	}
	deviceToken := ""
	if requestCookie, err := httpapi.Cookie(ctx, cookie.DeviceTokenCookieName); err == nil {
		deviceToken = requestCookie.Value
	}
	user, token, err := uc.oneTimeAccessService.ExchangeOneTimeAccessToken(ctx, input.Token, deviceToken, httpapi.ClientIP(ctx), httpapi.UserAgent(ctx))
	if err != nil {
		return nil, err
	}
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	maxAge := int(uc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	return &userCookieOutput{SetCookie: []http.Cookie{*cookie.NewAccessTokenCookie(maxAge, token)}, Body: output}, nil
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
	if err := uc.userService.SendEmailVerification(ctx, httpapi.UserID(ctx)); err != nil {
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

type emptyOutputWithCookie struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

func mapUser(user model.User) (*httpapi.BodyOutput[dto.UserDto], error) {
	var output dto.UserDto
	if err := dto.MapStruct(user, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.UserDto]{Body: output}, nil
}
