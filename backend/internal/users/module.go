package users

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/customclaims"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

// CustomClaimUpdater applies the default custom claims to new users
type CustomClaimUpdater interface {
	UpdateCustomClaimsForUserInternal(ctx context.Context, userID string, claims []customclaims.CustomClaimCreateDto, tx *gorm.DB) ([]model.CustomClaim, error)
}

// AppAssetProvider serves the default profile picture an admin may have uploaded
type AppAssetProvider interface {
	IsDefaultProfilePictureSet() bool
	GetAsset(ctx context.Context, name string) (io.ReadCloser, int64, string, error)
}

// PasskeyManager lists and deletes the passkeys of any user for admins
type PasskeyManager interface {
	ListCredentials(ctx context.Context, userID string) ([]webauthn.WebauthnCredential, error)
	DeleteCredential(ctx context.Context, userID, credentialID string, origin auditlogs.Origin) error
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

// OwnAccountLimiter limits how often one key, the user's ID, may change their own username or email
type OwnAccountLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration, err error)
}

type AuditLogger interface {
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
	KeepUsername(ctx context.Context, userID, username string, tx *gorm.DB) error
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
	OwnAccountLimiter OwnAccountLimiter
	AuditLog          AuditLogger
	Actors            francishost.Host

	// CleanupDisabled skips registering the cron job that deletes unused default profile pictures, for example in tests
	CleanupDisabled bool
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register the cleanup job for the generated default profile pictures
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the users cleanup cron job")
		}

		cj, err := newCleanupJob(deps.DB, deps.FileStorage)
		if err != nil {
			return nil, err
		}

		err = deps.Actors.RegisterBuiltInActor(cj)
		if err != nil {
			return nil, fmt.Errorf("error registering users cleanup cron actor %q: %w", cj.ActorType(), err)
		}
	}

	service := newService(deps)
	return &Module{
		service: service,
		handler: newHandler(deps.AppConfig, service, deps.WebAuthn),
	}, nil
}

func (m *Module) RegisterRoutes(r *authz.Router, profilePictureSizeLimit gin.HandlerFunc) {
	r.GET("/users", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.listUsersHandler),
		option.Summary("List users"),
		option.Description("Get a paginated list of users with optional search and sorting"),
		option.Tags("Users"),
		option.Query("search", "Search term to filter users"),
		openapi.List[model.User](),
	)
	r.GET("/users/me", authz.AccountRead, httpserver.JSON(http.StatusOK, m.handler.getCurrentUserHandler),
		option.Summary("Get current user"),
		option.Description("Retrieve information about the currently authenticated user"),
		option.Tags("Users"),
	)
	r.GET("/users/:id", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.getUserHandler),
		option.Summary("Get user by ID"),
		option.Description("Retrieve detailed information about a specific user"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	r.POST("/users", authz.UsersWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.createUserHandler),
		option.Summary("Create user"),
		option.Description("Create a new user"),
		option.Tags("Users"),
	)
	updateUserDocs := option.Group(
		option.Summary("Update user"),
		option.Description("Update an existing user by ID, where PATCH changes only the fields in the body"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	r.PUT("/users/:id", authz.UsersWrite, httpserver.UpdateJSON(m.handler.updateUserHandler), updateUserDocs)
	r.PATCH("/users/:id", authz.UsersWrite, httpserver.UpdateJSON(m.handler.updateUserHandler), updateUserDocs)
	r.GET("/users/:id/groups", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.getUserGroupsHandler),
		option.Summary("Get user groups"),
		option.Description("Retrieve all groups a specific user belongs to"),
		option.Tags("Users", "User Groups"),
		option.Path("id", "User ID"),
	)
	r.GET("/users/:id/webauthn-credentials", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.listUserWebauthnCredentialsHandler),
		option.Summary("List user passkeys"),
		option.Description("Retrieve all WebAuthn credentials for a specific user"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	updateCurrentUserDocs := option.Group(
		option.Summary("Update current user"),
		option.Description("Update the currently authenticated user's information, where PATCH changes only the fields in the body"),
		option.Tags("Users"),
	)
	r.PUT("/users/me", authz.AccountWrite, httpserver.UpdateJSON(m.handler.updateCurrentUserHandler), updateCurrentUserDocs)
	r.PATCH("/users/me", authz.AccountWrite, httpserver.UpdateJSON(m.handler.updateCurrentUserHandler), updateCurrentUserDocs)
	r.DELETE("/users/:id", authz.UsersWrite, httpserver.NoContent(m.handler.deleteUserHandler),
		option.Summary("Delete user"),
		option.Description("Delete a specific user by ID"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	r.DELETE("/users/:id/webauthn-credentials/:credentialId", authz.UsersWrite, httpserver.NoContent(m.handler.deleteUserWebauthnCredentialHandler),
		option.Summary("Delete user passkey"),
		option.Description("Delete a specific WebAuthn credential for a user"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
		option.Path("credentialId", "Credential ID"),
	)

	r.PUT("/users/:id/user-groups", authz.UsersWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateUserGroups),
		option.Summary("Update user groups"),
		option.Description("Update the groups a specific user belongs to"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)

	r.Public().GET("/users/:id/profile-picture.png", httpserver.Handle(m.handler.getUserProfilePictureHandler),
		option.Summary("Get user profile picture"),
		option.Description("Retrieve a specific user's profile picture"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
		openapi.File("PNG image", "image/png"),
	)

	r.With(profilePictureSizeLimit).PUT("/users/:id/profile-picture", authz.UsersWrite, httpserver.NoContent(m.handler.updateUserProfilePictureHandler),
		option.Summary("Update user profile picture"),
		option.Description("Update a specific user's profile picture"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
		openapi.Upload("Profile picture image file (PNG, JPG, or JPEG)"),
	)
	r.With(profilePictureSizeLimit).PUT("/users/me/profile-picture", authz.AccountWrite, httpserver.NoContent(m.handler.updateCurrentUserProfilePictureHandler),
		option.Summary("Update current user's profile picture"),
		option.Description("Update the currently authenticated user's profile picture"),
		option.Tags("Users"),
		openapi.Upload("Profile picture image file (PNG, JPG, or JPEG)"),
	)

	r.DELETE("/users/:id/profile-picture", authz.UsersWrite, httpserver.NoContent(m.handler.resetUserProfilePictureHandler),
		option.Summary("Reset user profile picture"),
		option.Description("Reset a specific user's profile picture to the default"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	r.DELETE("/users/me/profile-picture", authz.AccountWrite, httpserver.NoContent(m.handler.resetCurrentUserProfilePictureHandler),
		option.Summary("Reset current user's profile picture"),
		option.Description("Reset the currently authenticated user's profile picture to the default"),
		option.Tags("Users"),
	)
}

// GetUser returns a user with their groups and custom claims
func (m *Module) GetUser(ctx context.Context, userID string) (model.User, error) {
	return m.service.GetUser(ctx, userID)
}

// DeleteUser deletes a user and their profile picture
func (m *Module) DeleteUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, allowLdapDelete bool, origin auditlogs.Origin) error {
	return m.service.DeleteUser(ctx, dbConfig, userID, allowLdapDelete, origin)
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
	return m.service.updateUserInternal(ctx, dbConfig, userID, input, updateOwnUser, isLdapSync, auditlogs.LDAPSync, tx)
}

// DisableUserInternal disables a user within an existing transaction
// It's exported for the LDAP sync, which soft-deletes users that are no longer in the directory
func (m *Module) DisableUserInternal(ctx context.Context, tx *gorm.DB, userID string) error {
	return m.service.disableUserInternal(ctx, tx, userID, auditlogs.LDAPSync)
}

// DeleteUserInternal deletes a user within an existing transaction
// It's exported for the LDAP sync, which deletes users that are no longer in the directory
// Note that the caller is responsible for removing the user's profile picture from the storage layer, which must happen outside of the transaction
func (m *Module) DeleteUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, tx *gorm.DB, userID string, allowLdapDelete bool) error {
	return m.service.deleteUserInternal(ctx, dbConfig, tx, userID, allowLdapDelete, auditlogs.LDAPSync)
}
