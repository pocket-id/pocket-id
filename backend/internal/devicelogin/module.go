package devicelogin

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

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
	DeviceStringFromUserAgent(userAgent string) string
}

type AppConfigProvider interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB      *gorm.DB
	Actors  *local.Host
	BaseURL string

	Signer    TokenService
	Reauth    ReauthenticationTokenConsumer
	AuditLog  AuditLogger
	AppConfig AppConfigProvider
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := NewService(deps.Actors.Service(), deps.DB, deps.Signer, deps.Reauth, deps.AuditLog)
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
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, browserAuth, createRateLimit, exchangeRateLimit, verificationRateLimit gin.HandlerFunc) {
	apiGroup.POST("/device-login/requests", createRateLimit, m.handler.createRequest)
	apiGroup.POST("/device-login/requests/:id/exchange", exchangeRateLimit, m.handler.exchangeRequest)
	apiGroup.POST("/device-login/verification", verificationRateLimit, browserAuth, m.handler.inspectRequest)
	apiGroup.POST("/device-login/verification/decision", verificationRateLimit, browserAuth, m.handler.decideRequest)
}
