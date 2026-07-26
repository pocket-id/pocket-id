package usersignup

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type UserCreator interface {
	CreateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
}

// AppConfigResolver loads the current application configuration, so handlers can pass it explicitly to the service methods that need it
type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors *local.Host

	Signer      TokenService
	AuditLog    AuditLogger
	UserCreator UserCreator
	AppConfig   AppConfigResolver
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
// adminAuth guards the admin token-management routes; signupRateLimit throttles public self-signup
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, adminAuth, signupRateLimit gin.HandlerFunc) {
	apiGroup.POST("/signup-tokens", adminAuth, m.handler.createSignupToken)
	apiGroup.GET("/signup-tokens", adminAuth, m.handler.listSignupTokens)
	apiGroup.DELETE("/signup-tokens/:id", adminAuth, m.handler.deleteSignupToken)
	apiGroup.POST("/signup", signupRateLimit, m.handler.signup)
	apiGroup.GET("/signup/setup", m.handler.checkInitialAdminSetupAvailable)
	apiGroup.POST("/signup/setup", m.handler.signUpInitialAdmin)
}
