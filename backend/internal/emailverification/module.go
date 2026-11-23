package emailverification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type AuditLogger interface {
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	EmailSender EmailSender
	ScimSync    ScimSyncScheduler
	// SendLimiter is nil when rate limiting is disabled
	SendLimiter SendLimiter
	AppConfig   appconfig.AppConfigResolver
	AuditLog    AuditLogger
	AppURL      string
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	err := deps.Actors.RegisterActor(ActorType, NewActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", ActorType, err)
	}

	service := newService(deps.DB, deps.Actors.Service(), deps.EmailSender, deps.ScimSync, deps.SendLimiter, deps.AuditLog, deps.AppURL)
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// Send issues a new verification token for the user's current email and emails the verification link
func (m *Module) Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error {
	return m.service.Send(ctx, dbConfig, userID)
}

// AllowSend consumes one of the user's verification emails, or returns a rate-limit error once they're used up
func (m *Module) AllowSend(ctx context.Context, userID string) error {
	return m.service.AllowSend(ctx, userID)
}

// RequestEmailChange emails a verification link to a new address, which replaces the user's current email once the link is used
func (m *Module) RequestEmailChange(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, newEmail string) error {
	return m.service.RequestEmailChange(ctx, dbConfig, userID, newEmail)
}

// RegisterRoutes mounts the email verification endpoints
func (m *Module) RegisterRoutes(r *authz.Router, verifyRateLimit gin.HandlerFunc) {
	r.POST("/users/me/send-email-verification", authz.AccountWrite, httpserver.NoContent(m.handler.send),
		option.Summary("Send email verification"),
		option.Description("Send an email verification to the currently authenticated user"),
		option.Tags("Users"),
	)
	r.With(verifyRateLimit).POST("/users/me/verify-email", authz.AccountWrite, httpserver.BodyNoContent(m.handler.verify),
		option.Summary("Verify email"),
		option.Description("Verify the currently authenticated user's email using a verification token"),
		option.Tags("Users"),
	)
	r.GET("/users/me/email-change", authz.AccountRead, httpserver.Handle(m.handler.pendingEmailChange),
		option.Summary("Get pending email change"),
		option.Description("Get the email change of the currently authenticated user that still has to be confirmed through the emailed link"),
		option.Tags("Users"),
		openapi.Response[emailChangeDto](http.StatusOK, ""),
		openapi.Status(http.StatusNoContent, "No pending email change"),
	)
	r.DELETE("/users/me/email-change", authz.AccountWrite, httpserver.NoContent(m.handler.cancelEmailChange),
		option.Summary("Cancel pending email change"),
		option.Description("Cancel the pending email change of the currently authenticated user, so its verification link stops working"),
		option.Tags("Users"),
	)
}
