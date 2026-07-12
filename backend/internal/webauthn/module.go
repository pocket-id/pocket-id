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
	httpapi.Register(api, huma.Operation{
		OperationID: "begin-webauthn-registration",
		Method:      http.MethodGet,
		Path:        "/api/webauthn/register/start",
		Summary:     "Begin WebAuthn registration",
		Tags:        []string{"WebAuthn"},
	}, m.handler.beginRegistration, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "finish-webauthn-registration",
		Method:      http.MethodPost,
		Path:        "/api/webauthn/register/finish",
		Summary:     "Finish WebAuthn registration",
		Tags:        []string{"WebAuthn"},
	}, m.handler.verifyRegistration, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "begin-webauthn-login",
		Method:      http.MethodGet,
		Path:        "/api/webauthn/login/start",
		Summary:     "Begin WebAuthn login",
		Tags:        []string{"WebAuthn"},
	}, m.handler.beginLogin)

	httpapi.Register(api, huma.Operation{
		OperationID: "finish-webauthn-login",
		Method:      http.MethodPost,
		Path:        "/api/webauthn/login/finish",
		Summary:     "Finish WebAuthn login",
		Tags:        []string{"WebAuthn"},
	}, m.handler.verifyLogin, httpapi.WithMiddleware(loginRateLimit))

	httpapi.Register(api, huma.Operation{
		OperationID:   "webauthn-logout",
		Method:        http.MethodPost,
		Path:          "/api/webauthn/logout",
		Summary:       "Log out",
		Tags:          []string{"WebAuthn"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.logout, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "webauthn-reauthenticate",
		Method:        http.MethodPost,
		Path:          "/api/webauthn/reauthenticate",
		Summary:       "Reauthenticate",
		Tags:          []string{"WebAuthn"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.reauthenticate, userAuth, httpapi.WithMiddleware(reauthRateLimit))

	httpapi.Register(api, huma.Operation{
		OperationID: "list-webauthn-credentials",
		Method:      http.MethodGet,
		Path:        "/api/webauthn/credentials",
		Summary:     "List WebAuthn credentials",
		Tags:        []string{"WebAuthn"},
	}, m.handler.listCredentials, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-webauthn-credential",
		Method:      http.MethodPatch,
		Path:        "/api/webauthn/credentials/{id}",
		Summary:     "Update WebAuthn credential",
		Tags:        []string{"WebAuthn"},
	}, m.handler.updateCredential, userAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-webauthn-credential",
		Method:        http.MethodDelete,
		Path:          "/api/webauthn/credentials/{id}",
		Summary:       "Delete WebAuthn credential",
		Tags:          []string{"WebAuthn"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.deleteCredential, userAuth)
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
