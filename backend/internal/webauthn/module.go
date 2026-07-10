package webauthn

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string) (string, error)
	VerifyAccessToken(tokenString string) (jwt.Token, error)
	GetAuthenticationMethod(token jwt.Token) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
	CreateNewSignInWithEmail(ctx context.Context, ipAddress, userAgent, userID string, tx *gorm.DB) model.AuditLog
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
func (m *Module) RegisterRoutes(api huma.API, userAuth func(*huma.Operation), loginRateLimit, reauthRateLimit func(huma.Context, func(huma.Context))) {
	beginRegistration := webauthnOperation("begin-webauthn-registration", http.MethodGet, "/api/webauthn/register/start", "Begin WebAuthn registration")
	userAuth(&beginRegistration)
	httpapi.Register(api, beginRegistration, m.handler.beginRegistration)

	verifyRegistration := webauthnOperation("finish-webauthn-registration", http.MethodPost, "/api/webauthn/register/finish", "Finish WebAuthn registration")
	userAuth(&verifyRegistration)
	httpapi.Register(api, verifyRegistration, m.handler.verifyRegistration)

	httpapi.Register(api, webauthnOperation("begin-webauthn-login", http.MethodGet, "/api/webauthn/login/start", "Begin WebAuthn login"), m.handler.beginLogin)

	verifyLogin := webauthnOperation("finish-webauthn-login", http.MethodPost, "/api/webauthn/login/finish", "Finish WebAuthn login")
	verifyLogin.Middlewares = append(verifyLogin.Middlewares, loginRateLimit)
	httpapi.Register(api, verifyLogin, m.handler.verifyLogin)

	logout := webauthnOperation("webauthn-logout", http.MethodPost, "/api/webauthn/logout", "Log out")
	logout.DefaultStatus = http.StatusNoContent
	userAuth(&logout)
	httpapi.Register(api, logout, m.handler.logout)

	reauthenticate := webauthnOperation("webauthn-reauthenticate", http.MethodPost, "/api/webauthn/reauthenticate", "Reauthenticate")
	reauthenticate.DefaultStatus = http.StatusNoContent
	userAuth(&reauthenticate)
	reauthenticate.Middlewares = append(reauthenticate.Middlewares, reauthRateLimit)
	httpapi.Register(api, reauthenticate, m.handler.reauthenticate)

	listCredentials := webauthnOperation("list-webauthn-credentials", http.MethodGet, "/api/webauthn/credentials", "List WebAuthn credentials")
	userAuth(&listCredentials)
	httpapi.Register(api, listCredentials, m.handler.listCredentials)

	updateCredential := webauthnOperation("update-webauthn-credential", http.MethodPatch, "/api/webauthn/credentials/{id}", "Update WebAuthn credential")
	userAuth(&updateCredential)
	httpapi.Register(api, updateCredential, m.handler.updateCredential)

	deleteCredential := webauthnOperation("delete-webauthn-credential", http.MethodDelete, "/api/webauthn/credentials/{id}", "Delete WebAuthn credential")
	deleteCredential.DefaultStatus = http.StatusNoContent
	userAuth(&deleteCredential)
	httpapi.Register(api, deleteCredential, m.handler.deleteCredential)
}

func webauthnOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"WebAuthn"}}
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
