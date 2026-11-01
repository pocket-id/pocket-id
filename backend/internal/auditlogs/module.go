// Package auditlogs owns audit records, their HTTP API, sign-in notifications, and retention cleanup
package auditlogs

import (
	"context"
	"errors"
	"fmt"
	"time"

	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
)

type NewLoginEmailSender interface {
	SendNewLogin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, ipAddress, country, city, device, method string, dateTime time.Time) error
}

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	EmailSender NewLoginEmailSender
	IPLocator   iplocation.Resolver
	AppConfig   appconfig.AppConfigResolver
	EnvConfig   *common.EnvConfigSchema
	InstanceID  string
	AppURL      string

	// RetentionDays is how long audit logs are kept before the cleanup job deletes them
	RetentionDays int

	// CleanupDisabled skips registering cleanup cron jobs, for example in tests
	CleanupDisabled bool
}

type Module struct {
	service       *service
	handler       *handler
	browserTokens *browserTokens
}

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	// Register audit retention cleanup before the actor host starts
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the audit log cleanup cron job")
		}

		jobs, err := newCleanupJobs(deps.DB, deps.RetentionDays)
		if err != nil {
			return nil, err
		}

		for _, cj := range jobs {
			if err := deps.Actors.RegisterBuiltInActor(cj); err != nil {
				return nil, fmt.Errorf("error registering audit log cleanup cron actor %q: %w", cj.ActorType(), err)
			}
		}
	}

	browserTokens := &browserTokens{db: deps.DB, envConfig: deps.EnvConfig, appURL: deps.AppURL}
	err := browserTokens.loadKey(ctx, deps.InstanceID)
	if err != nil {
		return nil, err
	}

	service := newService(deps.DB, deps.EmailSender, deps.IPLocator, deps.AppConfig, browserTokens)
	return &Module{service: service, handler: newHandler(service), browserTokens: browserTokens}, nil
}

// ReloadBrowserKey replaces the known-browser key with the one stored for the instance, which the E2E reset needs after switching to a fixed instance ID
func (m *Module) ReloadBrowserKey(ctx context.Context, instanceID string) error {
	return m.browserTokens.loadKey(ctx, instanceID)
}

// RegisterRoutes mounts the audit-log queries for the caller's own events and for all events
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/audit-logs/all", authz.AuditLogsRead, httpserver.Handle(m.handler.listAllAuditLogsHandler))
	r.GET("/audit-logs", authz.AccountAuditLogs, httpserver.Handle(m.handler.listAuditLogsForUserHandler))
	r.GET("/audit-logs/filters/client-names", authz.AuditLogsRead, httpserver.Handle(m.handler.listClientNamesHandler))
	r.GET("/audit-logs/filters/users", authz.AuditLogsRead, httpserver.Handle(m.handler.listUserNamesWithIdsHandler))
}

// Create records an event within the caller's transaction
func (m *Module) Create(ctx context.Context, event Event, ipAddress, userAgent, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	return m.service.Create(ctx, event, ipAddress, userAgent, userID, data, tx)
}

// CreateSignIn prepares browser recognition and notification delivery within the caller's transaction
func (m *Module) CreateSignIn(ctx context.Context, event Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) SignInResult {
	return m.service.CreateSignIn(ctx, event, ipAddress, userAgent, userID, browserToken, tx, notificationMode)
}

// SendSignInNotification must be called only after the login commits successfully
func (m *Module) SendSignInNotification(ctx context.Context, result SignInResult) {
	m.service.SendSignInNotification(ctx, result)
}

// DeviceStringFromUserAgent describes a browser for audit records and device-login approval
func (m *Module) DeviceStringFromUserAgent(userAgent string) string {
	return m.service.DeviceStringFromUserAgent(userAgent)
}
