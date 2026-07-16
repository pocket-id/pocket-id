package devicelogin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string) (string, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
	DeviceStringFromUserAgent(userAgent string) string
}

type AppConfigProvider interface {
	GetDbConfig() *model.AppConfig
}

type Dependencies struct {
	DB      *gorm.DB
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

func New(deps Dependencies) *Module {
	service := newService(deps)
	return &Module{
		service: service,
		handler: newHandler(service, deps.BaseURL, deps.AppConfig),
	}
}

// RegisterRoutes mounts the public exchange and authenticated verification endpoints
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, browserAuth, createRateLimit, verificationRateLimit gin.HandlerFunc) {
	apiGroup.POST("/device-login/requests", createRateLimit, m.handler.createRequest)
	apiGroup.POST("/device-login/requests/:id/exchange", m.handler.exchangeRequest)
	apiGroup.POST("/device-login/verification", verificationRateLimit, browserAuth, m.handler.inspectRequest)
	apiGroup.POST("/device-login/verification/decision", verificationRateLimit, browserAuth, m.handler.decideRequest)
}
