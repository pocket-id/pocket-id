package runtimecredential

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
	CreateNewSignInWithEmail(ctx context.Context, ipAddress, userAgent, userID string, tx *gorm.DB, emailLoginNotificationEnabled bool) model.AuditLog
}

type BootstrapTokenConsumer interface {
	ConsumeToken(ctx context.Context, token, deviceToken string) (onetimeaccess.TokenState, error)
	RestoreToken(ctx context.Context, token string, state onetimeaccess.TokenState)
}

type ReauthenticationTokenIssuer interface {
	CreateReauthenticationToken(ctx context.Context, tx *gorm.DB, userID string) (string, error)
}

type Dependencies struct {
	DB        *gorm.DB
	Signer    TokenService
	AuditLog  AuditLogger
	Bootstrap BootstrapTokenConsumer
	Reauth    ReauthenticationTokenIssuer
	AppConfig appconfig.AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps)
	return &Module{service: service, handler: newHandler(service, deps.AppConfig)}
}

// RegisterRoutes mounts the FCA04 public proof, authenticated reauthentication, self-service, and administrator lifecycle endpoints
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, userAuth, browserAuth, adminAuth, publicRateLimit, reauthRateLimit gin.HandlerFunc) {
	apiGroup.POST("/runtime-credentials/register/start", publicRateLimit, httpserver.Handle(m.handler.beginRegistration))
	apiGroup.POST("/runtime-credentials/register/finish", publicRateLimit, httpserver.Handle(m.handler.finishRegistration))
	apiGroup.POST("/runtime-credentials/login/start", publicRateLimit, httpserver.Handle(m.handler.beginLogin))
	apiGroup.POST("/runtime-credentials/login/finish", publicRateLimit, httpserver.Handle(m.handler.finishLogin))

	apiGroup.POST("/runtime-credentials/reauthenticate/start", browserAuth, reauthRateLimit, httpserver.Handle(m.handler.beginReauthentication))
	apiGroup.POST("/runtime-credentials/reauthenticate/finish", browserAuth, reauthRateLimit, httpserver.Handle(m.handler.finishReauthentication))

	apiGroup.GET("/runtime-credentials", userAuth, httpserver.Handle(m.handler.listOwnCredentials))
	apiGroup.PATCH("/runtime-credentials/:id", userAuth, httpserver.Handle(m.handler.updateOwnCredential))
	apiGroup.DELETE("/runtime-credentials/:id", userAuth, httpserver.Handle(m.handler.revokeOwnCredential))

	apiGroup.GET("/users/:id/runtime-credentials", adminAuth, httpserver.Handle(m.handler.listUserCredentials))
	apiGroup.DELETE("/users/:id/runtime-credentials/:credentialId", adminAuth, httpserver.Handle(m.handler.revokeUserCredential))
}
