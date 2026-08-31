package onetimeaccess

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
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

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	Signer       TokenService
	AuditLog     AuditLogger
	UserProvider UserProvider
	EmailSender  EmailSender
	AppConfig    appconfig.AppConfigResolver
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
	apiGroup.POST("/users/:id/one-time-access-token", auth, httpserver.Handle(m.handler.createTokenForUser))
	apiGroup.POST("/users/:id/one-time-access-email", auth, httpserver.Handle(m.handler.requestEmailAsAdmin))
	apiGroup.POST("/one-time-access-token/:token", exchangeRateLimit, httpserver.Handle(m.handler.exchangeToken))
	apiGroup.POST("/one-time-access-email", emailRateLimit, httpserver.Handle(m.handler.requestEmailAsUnauthenticatedUser))
}
