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

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	// Load the signup tokens currently stored in the database, so the singleton actor can be seeded (migrated) from them on first startup
	existing, err := loadExistingSignupTokens(ctx, deps.DB)
	if err != nil {
		return nil, err
	}

	// Register the singleton actor that holds all signup tokens
	bootstrapData := &signupTokenBootstrap{
		Tokens: existing,
	}
	err = deps.Actors.RegisterSingletonActor(
		SignupTokenActorType, NewSignupTokenActor,
		local.WithBootstrapData(bootstrapData),
		local.WithIdleTimeout(-1), // Disable idle timeout for this actor
	)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", SignupTokenActorType, err)
	}

	service := newService(deps, deps.Actors.Service())
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
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
