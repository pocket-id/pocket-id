// Package session owns the browser sessions users sign in to Pocket ID with, and lets users and admins review and revoke them
package session

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
)

type AuditLogger interface {
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
	DeviceStringFromUserAgent(userAgent string) string
}

// EndNotifier is told about sessions that ended, so it can end what the apps were granted through them
type EndNotifier interface {
	// SessionsEnded is called for sessions the user signed out of or that were revoked
	SessionsEnded(ctx context.Context, userID string, sessionIDs []string)
	// SessionsExpired is called for sessions that ran out, and an error makes the expiry alarm retry
	SessionsExpired(ctx context.Context, userID string, sessionIDs []string) error
	// SignedOutEverywhere is called after an admin signed the user out of every session, after SessionsEnded for those sessions
	SignedOutEverywhere(ctx context.Context, userID string) error
}

type Dependencies struct {
	Actors      francishost.Host
	IPLocator   iplocation.Resolver
	AuditLog    AuditLogger
	EndNotifier EndNotifier
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	err := deps.Actors.RegisterActor(ActorType, newActorFactory(deps.EndNotifier))
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", ActorType, err)
	}

	service := newService(deps.Actors.Service(), deps.IPLocator, deps.AuditLog, deps.EndNotifier)
	return &Module{service: service, handler: &handler{service: service}}, nil
}

// RegisterRoutes mounts the endpoints to review, remember and revoke the caller's own sessions and those of any user
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/sessions", authz.AccountSession, httpserver.JSON(http.StatusOK, m.handler.listOwn),
		option.Summary("List own sessions"),
		option.Description("Get the browsers in which the current user is signed in, most recently active first"),
		option.Tags("Sessions"),
	)
	r.DELETE("/sessions", authz.AccountSession, httpserver.NoContent(m.handler.revokeOtherOwn),
		option.Summary("Revoke other own sessions"),
		option.Description("Sign the current user out of every session except the current one"),
		option.Tags("Sessions"),
	)
	r.DELETE("/sessions/:id", authz.AccountSession, httpserver.NoContent(m.handler.revokeOwn),
		option.Summary("Revoke own session"),
		option.Description("Sign the current user out of one of their sessions"),
		option.Tags("Sessions"),
		option.Path("id", "Session ID"),
	)
	r.POST("/sessions/current/remember", authz.AccountSession, httpserver.NoContent(m.handler.rememberCurrent),
		option.Summary("Remember current session"),
		option.Description("Keep the current session signed in after the browser closes and while it goes unused, for a sign-in that was not remembered"),
		option.Tags("Sessions"),
	)

	admin := r.Group("/admin")
	admin.GET("/users/:id/sessions", authz.UsersRead, httpserver.JSON(http.StatusOK, m.handler.listForUser),
		option.Summary("List sessions of a user"),
		option.Description("Get the browsers in which a user is signed in, most recently active first"),
		option.Tags("Sessions"),
		option.Path("id", "User ID"),
	)
	admin.DELETE("/users/:id/sessions", authz.UsersWrite, httpserver.NoContent(m.handler.revokeAllForUser),
		option.Summary("Revoke all sessions of a user"),
		option.Description("Sign a user out of every session and revoke every grant their apps hold, offline_access ones included"),
		option.Tags("Sessions"),
		option.Path("id", "User ID"),
	)
	admin.DELETE("/users/:id/sessions/:sessionId", authz.UsersWrite, httpserver.NoContent(m.handler.revokeForUser),
		option.Summary("Revoke a session of a user"),
		option.Description("Sign a user out of one of their sessions"),
		option.Tags("Sessions"),
		option.Path("id", "User ID"),
		option.Path("sessionId", "Session ID"),
	)
}

// Create starts a session for the user and returns its token
// It invokes an actor, so it must not be called while a database transaction is open
func (m *Module) Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration, remember bool) (string, error) {
	return m.service.Create(ctx, userID, authenticationMethod, ipAddress, userAgent, duration, remember)
}

// Validate returns the ID of the user and the active session the token belongs to
func (m *Module) Validate(ctx context.Context, token string) (string, Session, error) {
	return m.service.Validate(ctx, token)
}

// IsActive reports whether the user is still signed in with the session
// It invokes an actor, so it must not be called while a database transaction is open
func (m *Module) IsActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return m.service.IsActive(ctx, userID, sessionID)
}

// Revoke signs the user out of one session on behalf of origin
func (m *Module) Revoke(ctx context.Context, userID, sessionID string, origin auditlogs.Origin) error {
	return m.service.Revoke(ctx, userID, sessionID, origin)
}
