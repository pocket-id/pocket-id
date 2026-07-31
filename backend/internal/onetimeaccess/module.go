package onetimeaccess

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// EmailSender sends the one-time access email
type EmailSender interface {
	SendOneTimeAccessEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, code, loginLink, loginLinkWithCode, expirationString string) error
}

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type UserProvider interface {
	GetUser(ctx context.Context, userID string) (model.User, error)
}

// AppConfigResolver loads the current application configuration, so handlers can pass it explicitly to the service methods that need it
type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors *local.Host

	Signer       TokenService
	AuditLog     AuditLogger
	UserProvider UserProvider
	EmailSender  EmailSender
	AppConfig    AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register the actor that manages a one-time access token
	// Each token is its own actor, whose actor ID is the token's value
	err := deps.Actors.RegisterActor(TokenActorType, NewTokenActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", TokenActorType, err)
	}

	service := newService(deps, deps.Actors.Service())
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the one-time access token endpoints
// auth guards the admin routes, while the rate limiters throttle the public exchange and email endpoints
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, auth, exchangeRateLimit, emailRateLimit gin.HandlerFunc) {
	apiGroup.POST("/users/:id/one-time-access-token", auth, m.handler.createTokenForUser)
	apiGroup.POST("/users/:id/one-time-access-email", auth, m.handler.requestEmailAsAdmin)
	apiGroup.POST("/one-time-access-token/:token", exchangeRateLimit, m.handler.exchangeToken)
	apiGroup.POST("/one-time-access-email", emailRateLimit, m.handler.requestEmailAsUnauthenticatedUser)
}
