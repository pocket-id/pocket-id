// Package auditlogs owns audit records, their HTTP API, sign-in notifications, and retention cleanup
package auditlogs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
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
	admin := r.Group("/admin")
	admin.GET("/audit-logs", authz.AuditLogsRead, httpserver.JSON(http.StatusOK, m.handler.listAllAuditLogsHandler),
		option.Summary("List all audit logs"),
		option.Description("Get a paginated list of all audit logs (admin only)"),
		option.Tags("Audit Logs"),
		openapi.List[AuditLog](),
		option.Query("from", "Only entries created at or after this RFC 3339 time"),
		option.Query("to", "Only entries created before this RFC 3339 time"),
		option.Query("filters[userID]", "Only entries of this user, including the actions they took"),
		option.QueryArray("filters[clientName]", "Only entries of the OIDC clients with these names", reflect.String),
		option.QueryArray("filters[location]", "Only entries from the internal or external network", reflect.String),
	)
	r.GET("/audit-logs", authz.AccountAuditLogs, httpserver.JSON(http.StatusOK, m.handler.listAuditLogsForUserHandler),
		option.Summary("List audit logs"),
		option.Description("Get a paginated list of the audit logs about the current user"),
		option.Tags("Audit Logs"),
		openapi.List[AuditLog](),
		option.Query("from", "Only entries created at or after this RFC 3339 time"),
		option.Query("to", "Only entries created before this RFC 3339 time"),
	)
	admin.GET("/audit-logs/filters/client-names", authz.AuditLogsRead, httpserver.JSON(http.StatusOK, m.handler.listClientNamesHandler),
		option.Summary("List client names"),
		option.Description("Get a list of all client names for audit log filtering"),
		option.Tags("Audit Logs"),
	)
	admin.GET("/audit-logs/filters/users", authz.AuditLogsRead, httpserver.JSON(http.StatusOK, m.handler.listUserNamesWithIdsHandler),
		option.Summary("List users with IDs"),
		option.Description("Get a list of all usernames with their IDs for audit log filtering"),
		option.Tags("Audit Logs"),
	)
}

// Create records an event within the caller's transaction
func (m *Module) Create(ctx context.Context, event Event, ipAddress, userAgent, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	return m.service.Create(ctx, event, ipAddress, userAgent, userID, data, tx)
}

// Record creates an entry about userID for an action that origin performed
func (m *Module) Record(ctx context.Context, event Event, origin Origin, userID string, data Data, tx *gorm.DB) {
	m.service.Record(ctx, event, origin, userID, data, tx)
}

// KeepUsername must be called within the transaction that deletes the user, before the delete
func (m *Module) KeepUsername(ctx context.Context, userID, username string, tx *gorm.DB) error {
	return m.service.KeepUsername(ctx, userID, username, tx)
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
