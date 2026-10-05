// Package auditlogs owns audit records, their HTTP API, sign-in notifications, and retention cleanup
package auditlogs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
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

	// RetentionDays is how long audit logs are kept before the cleanup job deletes them
	RetentionDays int

	// CleanupDisabled skips registering cleanup cron jobs, for example in tests
	CleanupDisabled bool
}

type Module struct {
	service *service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register both cleanup jobs before the actor host starts
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

	service := newService(deps.DB, deps.EmailSender, deps.IPLocator, deps.AppConfig)
	return &Module{service: service, handler: newHandler(service)}, nil
}

// RegisterRoutes mounts audit-log queries with the existing admin and current-user permissions
func (m *Module) RegisterRoutes(group *gin.RouterGroup, adminAuth, userAuth gin.HandlerFunc) {
	group.GET("/audit-logs/all", adminAuth, httpserver.Handle(m.handler.listAllAuditLogsHandler))
	group.GET("/audit-logs", userAuth, httpserver.Handle(m.handler.listAuditLogsForUserHandler))
	group.GET("/audit-logs/filters/client-names", adminAuth, httpserver.Handle(m.handler.listClientNamesHandler))
	group.GET("/audit-logs/filters/users", adminAuth, httpserver.Handle(m.handler.listUserNamesWithIdsHandler))
}

// Create records an event within the caller's transaction
func (m *Module) Create(ctx context.Context, event Event, ipAddress, userAgent, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	return m.service.Create(ctx, event, ipAddress, userAgent, userID, data, tx)
}

// CreateSignIn prepares browser recognition and notification delivery within the caller's transaction
func (m *Module) CreateSignIn(ctx context.Context, event Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, emailLoginNotificationEnabled bool) SignInResult {
	return m.service.CreateSignIn(ctx, event, ipAddress, userAgent, userID, browserToken, tx, emailLoginNotificationEnabled)
}

// SendSignInNotification must be called only after the login commits successfully
func (m *Module) SendSignInNotification(ctx context.Context, result SignInResult) {
	m.service.SendSignInNotification(ctx, result)
}

// DeviceStringFromUserAgent describes a browser for audit records and device-login approval
func (m *Module) DeviceStringFromUserAgent(userAgent string) string {
	return m.service.DeviceStringFromUserAgent(userAgent)
}
