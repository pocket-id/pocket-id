// Package session owns the browser sessions users sign in to Pocket ID with, and lets users and admins review and revoke them
package session

import (
	"context"
	"fmt"
	"time"

	francishost "github.com/italypaleale/francis/host"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
)

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

	service := newService(deps.Actors.Service(), deps.IPLocator, deps.EndNotifier)
	return &Module{service: service, handler: &handler{service: service}}, nil
}

// RegisterRoutes mounts the endpoints to review and revoke the caller's own sessions and those of any user
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/sessions", authz.AccountSession, httpserver.Handle(m.handler.listOwn))
	r.DELETE("/sessions", authz.AccountSession, httpserver.Handle(m.handler.revokeOtherOwn))
	r.DELETE("/sessions/:id", authz.AccountSession, httpserver.Handle(m.handler.revokeOwn))

	r.GET("/users/:id/sessions", authz.UsersRead, httpserver.Handle(m.handler.listForUser))
	r.DELETE("/users/:id/sessions", authz.UsersWrite, httpserver.Handle(m.handler.revokeAllForUser))
	r.DELETE("/users/:id/sessions/:sessionId", authz.UsersWrite, httpserver.Handle(m.handler.revokeForUser))
}

// Create starts a session for the user and returns its token
// It invokes an actor, so it must not be called while a database transaction is open
func (m *Module) Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration) (string, error) {
	return m.service.Create(ctx, userID, authenticationMethod, ipAddress, userAgent, duration)
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

// Revoke signs the user out of one session
func (m *Module) Revoke(ctx context.Context, userID, sessionID string) error {
	return m.service.Revoke(ctx, userID, sessionID)
}
