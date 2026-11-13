package devicelogin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type SessionCreator interface {
	Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration, remember bool) (string, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

type AuditLogger interface {
	CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) auditlogs.SignInResult
	SendSignInNotification(ctx context.Context, result auditlogs.SignInResult)
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
	DeviceStringFromUserAgent(userAgent string) string
}

type Dependencies struct {
	DB      *gorm.DB
	Actors  francishost.Host
	BaseURL string

	Sessions  SessionCreator
	Reauth    ReauthenticationTokenConsumer
	AuditLog  AuditLogger
	IPLocator iplocation.Resolver
	AppConfig appconfig.AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := NewService(deps.Actors.Service(), deps.DB, deps.Sessions, deps.Reauth, deps.AuditLog, deps.IPLocator)
	module := &Module{
		service: service,
		handler: newHandler(service, deps.BaseURL, deps.AppConfig),
	}

	// Register the durable request actor before the host starts
	err := deps.Actors.RegisterActor(
		requestActorType,
		newRequestActor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register device login actor: %w", err)
	}

	return module, nil
}

// RegisterRoutes mounts the public exchange and authenticated verification endpoints
func (m *Module) RegisterRoutes(r *authz.Router, createRateLimit, exchangeRateLimit, verificationRateLimit gin.HandlerFunc) {
	r.With(createRateLimit).Public().POST("/device-login/requests", httpserver.JSON(http.StatusCreated, m.handler.createRequest),
		option.Summary("Create device login request"),
		option.Description("Create a short-lived request that can be approved from another authenticated device"),
		option.Tags("Device Login"),
	)
	r.With(exchangeRateLimit).Public().POST("/device-login/requests/:id/exchange", httpserver.Handle(m.handler.exchangeRequest),
		option.Summary("Exchange device login request"),
		option.Description("Wait for a device login decision and create a browser session after it has been approved"),
		option.Tags("Device Login"),
		option.Path("id", "Device login request ID"),
		option.QueryBool("remember", "Set to false for a session that ends when the browser closes or after 20 minutes without activity", param.Default(true)),
		openapi.Response[dto.UserDto](http.StatusOK, "Approved request exchanged for a user session"),
		openapi.Status(http.StatusAccepted, "Authorization pending"),
	)
	r.With(verificationRateLimit).POST("/device-login/verification", authz.AccountSession, httpserver.BodyJSON(http.StatusOK, m.handler.inspectRequest),
		option.Summary("Inspect device login request"),
		option.Description("Retrieve the requesting device details for an authenticated user before approval or denial"),
		option.Tags("Device Login"),
	)
	r.With(verificationRateLimit).POST("/device-login/verification/decision", authz.AccountSession, httpserver.BodyNoContent(m.handler.decideRequest),
		option.Summary("Decide device login request"),
		option.Description("Approve or deny a device login request; approval requires fresh passkey reauthentication"),
		option.Tags("Device Login"),
	)
}
