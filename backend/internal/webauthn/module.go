package webauthn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type SessionStore interface {
	Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration, remember bool) (string, error)
	Revoke(ctx context.Context, userID, sessionID string, origin auditlogs.Origin) error
}

type AuditLogger interface {
	CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) auditlogs.SignInResult
	SendSignInNotification(ctx context.Context, result auditlogs.SignInResult)
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host
	AppURL string

	// HTTPClient downloads the FIDO metadata that attestations are verified against
	HTTPClient *http.Client
	// FidoMetadataURL is where the FIDO metadata is downloaded from, and empty never downloads it
	FidoMetadataURL string

	Sessions  SessionStore
	AuditLog  AuditLogger
	AppConfig appconfig.AppConfigResolver

	// CleanupDisabled skips registering the cron jobs that delete expired rows from the database, for example in tests
	CleanupDisabled bool
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

	// Register the cleanup jobs for expired WebAuthn rows
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the WebAuthn cleanup cron jobs")
		}

		jobs, err := newCleanupJobs(deps.DB)
		if err != nil {
			return nil, err
		}

		for _, cj := range jobs {
			err = deps.Actors.RegisterBuiltInActor(cj)
			if err != nil {
				return nil, fmt.Errorf("error registering WebAuthn cleanup cron actor %q: %w", cj.ActorType(), err)
			}
		}
	}

	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the WebAuthn registration, login and reauthentication endpoints
func (m *Module) RegisterRoutes(r *authz.Router, loginRateLimit, reauthRateLimit gin.HandlerFunc) {
	r.GET("/webauthn/register/start", authz.AccountPasskeysEnroll, httpserver.Handle(m.handler.beginRegistration))
	r.POST("/webauthn/register/finish", authz.AccountPasskeysEnroll, httpserver.Handle(m.handler.verifyRegistration))

	r.Public().GET("/webauthn/login/start", httpserver.Handle(m.handler.beginLogin))
	r.With(loginRateLimit).Public().POST("/webauthn/login/finish", httpserver.Handle(m.handler.verifyLogin))

	r.POST("/webauthn/logout", authz.AccountSession, httpserver.Handle(m.handler.logout))

	r.With(reauthRateLimit).POST("/webauthn/reauthenticate", authz.AccountSession, httpserver.Handle(m.handler.reauthenticate))

	r.GET("/webauthn/credentials", authz.AccountPasskeys, httpserver.Handle(m.handler.listCredentials))
	r.PATCH("/webauthn/credentials/:id", authz.AccountPasskeys, httpserver.Handle(m.handler.updateCredential))
	r.DELETE("/webauthn/credentials/:id", authz.AccountPasskeys, httpserver.Handle(m.handler.deleteCredential))

	r.Public().GET("/webauthn/authenticator-icons/:aaguid", httpserver.Handle(m.handler.getThemedAuthenticatorIcon))
}

// ConsumeReauthenticationToken implements the OIDC module's ReauthenticationTokenConsumer interface
func (m *Module) ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error) {
	return m.service.ConsumeReauthenticationToken(ctx, tx, token, userID)
}

// ListCredentials returns the passkeys registered for the given user
// It is consumed by the users module for the admin "manage passkeys" view
func (m *Module) ListCredentials(ctx context.Context, userID string) ([]WebauthnCredential, error) {
	return m.service.ListCredentials(ctx, userID)
}

// DeleteCredential removes a passkey, optionally on behalf of an admin or a client acting for another user
// It is consumed by the users module for the admin "manage passkeys" view
func (m *Module) DeleteCredential(ctx context.Context, userID, credentialID string, origin auditlogs.Origin) error {
	return m.service.DeleteCredential(ctx, userID, credentialID, origin)
}
