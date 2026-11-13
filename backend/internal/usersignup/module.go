package usersignup

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type SessionCreator interface {
	Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration, remember bool) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
}

type UserCreator interface {
	CreateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
}

type EmailVerificationSender interface {
	Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error
}

// ScimSyncScheduler schedules SCIM after the signup transaction has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	Sessions    SessionCreator
	AuditLog    AuditLogger
	UserCreator UserCreator
	AppConfig   appconfig.AppConfigResolver
	ScimSync    ScimSyncScheduler
	UserDeleter UserDeleter

	EmailVerification EmailVerificationSender

	// CleanupDisabled skips registering the cron job that deletes signups that missed their email verification deadline, for example in tests
	CleanupDisabled bool
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register the actor that manages a signup token
	// Each token is its own actor, whose actor ID is the token's value
	err := deps.Actors.RegisterActor(SignupTokenActorType, NewSignupTokenActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", SignupTokenActorType, err)
	}

	// Register the job that deletes self-signed-up accounts that did not verify their email in time
	if !deps.CleanupDisabled {
		job, err := newCleanupJob(deps.DB, deps.AppConfig, deps.UserDeleter)
		if err != nil {
			return nil, err
		}

		err = deps.Actors.RegisterBuiltInActor(job)
		if err != nil {
			return nil, fmt.Errorf("error registering unverified signup cleanup cron actor: %w", err)
		}
	}

	service := newService(deps, deps.Actors.Service())
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RunSignupTokenMigration performs the one-time migration of the pre-actor signup tokens, then blocks until the context is canceled.
// It's meant to be started as a background service gated on the actor host being ready, since the migration needs the actor state store.
// Note that it must not return before the context is canceled, as the service runner stops the application as soon as any of its services returns.
func (m *Module) RunSignupTokenMigration(ctx context.Context) error {
	err := m.service.migrateSignupTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to migrate signup tokens: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}

// RegisterRoutes mounts the signup and signup-token management endpoints
// signupRateLimit throttles public self-signup
func (m *Module) RegisterRoutes(r *authz.Router, signupRateLimit gin.HandlerFunc) {
	admin := r.Group("/admin")
	admin.POST("/signup-tokens", authz.UsersWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.createSignupToken),
		option.Summary("Create signup token"),
		option.Description("Create a new signup token that allows user registration"),
		option.Tags("Users"),
	)
	// Listing returns working token values, which can create accounts
	admin.GET("/signup-tokens", authz.UsersWrite, httpserver.JSON(http.StatusOK, m.handler.listSignupTokens),
		option.Summary("List signup tokens"),
		option.Description("Get a paginated list of signup tokens"),
		option.Tags("Users"),
		openapi.List[SignupToken](),
	)
	admin.DELETE("/signup-tokens/:id", authz.UsersWrite, httpserver.NoContent(m.handler.deleteSignupToken),
		option.Summary("Delete signup token"),
		option.Description("Delete a signup token by ID"),
		option.Tags("Users"),
		option.Path("id", "Token ID"),
	)
	r.With(signupRateLimit).Public().POST("/signup", httpserver.BodyJSON(http.StatusCreated, m.handler.signup),
		option.Summary("Sign up"),
		option.Description("Create a new user account"),
		option.Tags("Users"),
	)
	r.Public().GET("/signup/setup", httpserver.NoContent(m.handler.checkInitialAdminSetupAvailable),
		option.Summary("Check initial admin setup"),
		option.Description("Succeeds while no admin has been set up yet and fails with setup_not_available afterwards"),
		option.Tags("Users"),
	)
	r.Public().POST("/signup/setup", httpserver.BodyJSON(http.StatusOK, m.handler.signUpInitialAdmin),
		option.Summary("Sign up initial admin user"),
		option.Description("Sign up and generate setup access token for initial admin user"),
		option.Tags("Users"),
	)
}
