package devicelogin

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

type AuditLogger interface {
	CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) auditlogs.SignInResult
	SendSignInNotification(ctx context.Context, result auditlogs.SignInResult)
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
	DeviceStringFromUserAgent(userAgent string) string
}

type Dependencies struct {
	DB      *gorm.DB
	Actors  francishost.Host
	BaseURL string

	Signer    TokenService
	Reauth    ReauthenticationTokenConsumer
	AuditLog  AuditLogger
	IPLocator iplocation.Resolver
	AppConfig appconfig.AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := NewService(deps.Actors.Service(), deps.DB, deps.Signer, deps.Reauth, deps.AuditLog, deps.IPLocator)
	module := &Module{
		service: service,
		handler: newHandler(service, deps.BaseURL, deps.AppConfig),
	}

	// Register the durable request actor before the host starts
	err := deps.Actors.RegisterActor(
		requestActorType,
		newRequestActor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register device login actor: %w", err)
	}

	return module, nil
}

// RegisterRoutes mounts the public exchange and authenticated verification endpoints
func (m *Module) RegisterRoutes(r *authz.Router, createRateLimit, exchangeRateLimit, verificationRateLimit gin.HandlerFunc) {
	r.Public().POST("/device-login/requests", createRateLimit, httpserver.Handle(m.handler.createRequest))
	r.Public().POST("/device-login/requests/:id/exchange", exchangeRateLimit, httpserver.Handle(m.handler.exchangeRequest))
	r.POST("/device-login/verification", authz.AccountSession, verificationRateLimit, httpserver.Handle(m.handler.inspectRequest))
	r.POST("/device-login/verification/decision", authz.AccountSession, verificationRateLimit, httpserver.Handle(m.handler.decideRequest))
}
