package netbird

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type SessionCreator interface {
	Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration) (string, error)
}

type AuditLogger interface {
	CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) auditlogs.SignInResult
	SendSignInNotification(ctx context.Context, result auditlogs.SignInResult)
}

type Dependencies struct {
	DB *gorm.DB

	Sessions  SessionCreator
	AuditLog  AuditLogger
	AppConfig appconfig.AppConfigResolver
}

type Module struct {
	handler *handler
}

func New(deps Dependencies) *Module {
	service := &Service{
		db:       deps.DB,
		sessions: deps.Sessions,
		auditLog: deps.AuditLog,
	}
	return &Module{handler: newHandler(service, deps.AppConfig)}
}

// RegisterRoutes mounts the public NetBird identity and sign-in endpoints
func (m *Module) RegisterRoutes(r *authz.Router, rateLimit gin.HandlerFunc) {
	r.Public().GET("/netbird/identity", rateLimit, httpserver.Handle(m.handler.getIdentity))
	r.Public().POST("/netbird/login", rateLimit, httpserver.Handle(m.handler.login))
}
