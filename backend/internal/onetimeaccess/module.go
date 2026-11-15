package onetimeaccess

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
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// EmailSender sends the one-time access email
type EmailSender interface {
	SendOneTimeAccessEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, code, loginLink, loginLinkWithCode, expirationString string) error
}

type SessionCreator interface {
	Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration, remember bool) (string, error)
}

type AuditLogger interface {
	CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) auditlogs.SignInResult
	SendSignInNotification(ctx context.Context, result auditlogs.SignInResult)
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

type UserProvider interface {
	GetUser(ctx context.Context, userID string) (model.User, error)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	Sessions     SessionCreator
	AuditLog     AuditLogger
	UserProvider UserProvider
	EmailSender  EmailSender
	AppConfig    appconfig.AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register the actor that manages a one-time access token
	// Each token is its own actor, whose actor ID is the token's value
	err := deps.Actors.RegisterActor(TokenActorType, NewTokenActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", TokenActorType, err)
	}

	service := newService(deps, deps.Actors.Service())
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the one-time access token endpoints
// The rate limiters throttle the public exchange and email endpoints
func (m *Module) RegisterRoutes(r *authz.Router, exchangeRateLimit, emailRateLimit gin.HandlerFunc) {
	admin := r.Group("/admin")
	admin.POST("/users/:id/one-time-access-token", authz.UsersWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.createTokenForUser),
		option.Summary("Create one-time access token for user (admin)"),
		option.Description("Generate a one-time access token for a specific user (admin only)"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	admin.POST("/users/:id/one-time-access-email", authz.UsersWrite, httpserver.BodyNoContent(m.handler.requestEmailAsAdmin),
		option.Summary("Request one-time access email (admin)"),
		option.Description("Request a one-time access email for a specific user (admin only)"),
		option.Tags("Users"),
		option.Path("id", "User ID"),
	)
	r.With(exchangeRateLimit).Public().POST("/one-time-access-token", httpserver.BodyJSON(http.StatusOK, m.handler.exchangeToken),
		option.Summary("Exchange one-time access token"),
		option.Description("Exchange a one-time access token for a session token"),
		option.Tags("Users"),
		option.QueryBool("remember", "Set to false for a session that ends when the browser closes or after 20 minutes without activity", param.Default(true)),
	)
	r.With(emailRateLimit).Public().POST("/one-time-access-email", httpserver.BodyNoContent(m.handler.requestEmailAsUnauthenticatedUser),
		option.Summary("Request one-time access email"),
		option.Description("Request a one-time access email for unauthenticated users"),
		option.Tags("Users"),
	)
}
