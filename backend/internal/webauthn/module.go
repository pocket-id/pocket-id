package webauthn

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
	VerifyAccessToken(tokenString string) (jwt.Token, error)
	GetAuthenticationMethod(token jwt.Token) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
	CreateNewSignInWithEmail(ctx context.Context, ipAddress, userAgent, userID string, tx *gorm.DB, dbConfig *appconfig.AppConfigModel) model.AuditLog
}

type AppConfigProvider interface {
	GetDbConfig() *model.AppConfig
}

type Dependencies struct {
	DB     *gorm.DB
	AppURL string

	Signer    TokenService
	AuditLog  AuditLogger
	AppConfig AppConfigProvider
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service, err := newService(deps)
	if err != nil {
		return nil, err
	}

	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the WebAuthn registration, login and reauthentication endpoints
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, userAuth, loginRateLimit, reauthRateLimit gin.HandlerFunc) {
	apiGroup.GET("/webauthn/register/start", userAuth, m.handler.beginRegistration)
	apiGroup.POST("/webauthn/register/finish", userAuth, m.handler.verifyRegistration)

	apiGroup.GET("/webauthn/login/start", m.handler.beginLogin)
	apiGroup.POST("/webauthn/login/finish", loginRateLimit, m.handler.verifyLogin)

	apiGroup.POST("/webauthn/logout", userAuth, m.handler.logout)

	apiGroup.POST("/webauthn/reauthenticate", userAuth, reauthRateLimit, m.handler.reauthenticate)

	apiGroup.GET("/webauthn/credentials", userAuth, m.handler.listCredentials)
	apiGroup.PATCH("/webauthn/credentials/:id", userAuth, m.handler.updateCredential)
	apiGroup.DELETE("/webauthn/credentials/:id", userAuth, m.handler.deleteCredential)
}

// ConsumeReauthenticationToken implements the OIDC module's ReauthenticationTokenConsumer interface
func (m *Module) ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error) {
	return m.service.ConsumeReauthenticationToken(ctx, tx, token, userID)
}

// ListCredentials returns the passkeys registered for the given user
// It is consumed by the user controller for the admin "manage passkeys" view
func (m *Module) ListCredentials(ctx context.Context, userID string) ([]model.WebauthnCredential, error) {
	return m.service.ListCredentials(ctx, userID)
}

// DeleteCredential removes a passkey, optionally on behalf of an admin acting for another user
// It is consumed by the user controller for the admin "manage passkeys" view
func (m *Module) DeleteCredential(ctx context.Context, userID, credentialID, ipAddress, userAgent, actorUserID string) error {
	return m.service.DeleteCredential(ctx, userID, credentialID, ipAddress, userAgent, actorUserID)
}
