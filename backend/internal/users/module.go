package users

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

// CustomClaimUpdater applies the default custom claims to new users
type CustomClaimUpdater interface {
	UpdateCustomClaimsForUserInternal(ctx context.Context, userID string, claims []dto.CustomClaimCreateDto, tx *gorm.DB) ([]model.CustomClaim, error)
}

// AppAssetProvider serves the default profile picture an admin may have uploaded
type AppAssetProvider interface {
	IsDefaultProfilePictureSet() bool
	GetAsset(ctx context.Context, name string) (io.ReadCloser, int64, string, error)
}

// PasskeyManager lists and deletes the passkeys of any user for admins
type PasskeyManager interface {
	ListCredentials(ctx context.Context, userID string) ([]model.WebauthnCredential, error)
	DeleteCredential(ctx context.Context, userID, credentialID, ipAddress, userAgent string, actor authz.Principal) error
}

// ScimSyncScheduler schedules SCIM after a user change has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

// EmailVerificationSender emails verification links for a user's current email or for an email they want to change to
type EmailVerificationSender interface {
	AllowSend(ctx context.Context, userID string) error
	Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error
	RequestEmailChange(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, newEmail string) error
}

type Dependencies struct {
	DB                *gorm.DB
	AppConfig         appconfig.AppConfigResolver
	FileStorage       storage.FileStorage
	CustomClaims      CustomClaimUpdater
	AppAssets         AppAssetProvider
	WebAuthn          PasskeyManager
	ScimSync          ScimSyncScheduler
	BackchannelLogout *backchannellogout.Service
	EmailVerification EmailVerificationSender
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps)
	return &Module{
		service: service,
		handler: newHandler(deps.AppConfig, service, deps.WebAuthn),
	}
}

func (m *Module) RegisterRoutes(r *authz.Router, profilePictureSizeLimit gin.HandlerFunc) {
	r.GET("/users", authz.UsersRead, httpserver.Handle(m.handler.listUsersHandler))
	r.GET("/users/me", authz.AccountRead, httpserver.Handle(m.handler.getCurrentUserHandler))
	r.GET("/users/:id", authz.UsersRead, httpserver.Handle(m.handler.getUserHandler))
	r.POST("/users", authz.UsersWrite, httpserver.Handle(m.handler.createUserHandler))
	r.PUT("/users/:id", authz.UsersWrite, httpserver.Handle(m.handler.updateUserHandler))
	r.GET("/users/:id/groups", authz.UsersRead, httpserver.Handle(m.handler.getUserGroupsHandler))
	r.GET("/users/:id/webauthn-credentials", authz.UsersRead, httpserver.Handle(m.handler.listUserWebauthnCredentialsHandler))
	r.PUT("/users/me", authz.AccountWrite, httpserver.Handle(m.handler.updateCurrentUserHandler))
	r.DELETE("/users/:id", authz.UsersWrite, httpserver.Handle(m.handler.deleteUserHandler))
	r.DELETE("/users/:id/webauthn-credentials/:credentialId", authz.UsersWrite, httpserver.Handle(m.handler.deleteUserWebauthnCredentialHandler))

	r.PUT("/users/:id/user-groups", authz.UsersWrite, httpserver.Handle(m.handler.updateUserGroups))

	r.Public().GET("/users/:id/profile-picture.png", httpserver.Handle(m.handler.getUserProfilePictureHandler))

	r.PUT("/users/:id/profile-picture", authz.UsersWrite, profilePictureSizeLimit, httpserver.Handle(m.handler.updateUserProfilePictureHandler))
	r.PUT("/users/me/profile-picture", authz.AccountWrite, profilePictureSizeLimit, httpserver.Handle(m.handler.updateCurrentUserProfilePictureHandler))

	r.DELETE("/users/:id/profile-picture", authz.UsersWrite, httpserver.Handle(m.handler.resetUserProfilePictureHandler))
	r.DELETE("/users/me/profile-picture", authz.AccountWrite, httpserver.Handle(m.handler.resetCurrentUserProfilePictureHandler))
}

// GetUser returns a user with their groups and custom claims
func (m *Module) GetUser(ctx context.Context, userID string) (model.User, error) {
	return m.service.GetUser(ctx, userID)
}

// DeleteUser deletes a user and their profile picture
func (m *Module) DeleteUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, allowLdapDelete bool) error {
	return m.service.DeleteUser(ctx, dbConfig, userID, allowLdapDelete)
}

// UpdateProfilePicture converts the image into the user's square profile picture
func (m *Module) UpdateProfilePicture(ctx context.Context, userID string, file io.ReadSeeker) error {
	return m.service.UpdateProfilePicture(ctx, userID, file)
}

// CreateUserInternal creates a user within an existing transaction
// It's exported for the signup and the LDAP sync, which create users in transactions of their own
func (m *Module) CreateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error) {
	return m.service.createUserInternal(ctx, input, isLdapSync, tx, dbConfig)
}

// UpdateUserInternal updates a user within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) UpdateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, input dto.UserCreateDto, updateOwnUser bool, isLdapSync bool, tx *gorm.DB) (model.User, error) {
	return m.service.updateUserInternal(ctx, dbConfig, userID, input, updateOwnUser, isLdapSync, tx)
}

// DisableUserInternal disables a user within an existing transaction
// It's exported for the LDAP sync, which soft-deletes users that are no longer in the directory
func (m *Module) DisableUserInternal(ctx context.Context, tx *gorm.DB, userID string) error {
	return m.service.disableUserInternal(ctx, tx, userID)
}

// DeleteUserInternal deletes a user within an existing transaction
// It's exported for the LDAP sync, which deletes users that are no longer in the directory
// Note that the caller is responsible for removing the user's profile picture from the storage layer, which must happen outside of the transaction
func (m *Module) DeleteUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, tx *gorm.DB, userID string, allowLdapDelete bool) error {
	return m.service.deleteUserInternal(ctx, dbConfig, tx, userID, allowLdapDelete)
}
