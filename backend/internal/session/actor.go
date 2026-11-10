package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// Sessions are stored entirely in the actor state store, in one actor per user whose actor ID is the user ID
// Keeping all of a user's sessions in one actor makes listing their devices and signing them out everywhere a single state access
// The state's TTL follows the session that expires last, so it is purged once every session of the user has expired

// ActorType is the actor type for the actor that holds the sessions of one user
const ActorType = "UserSessions"

const (
	// MethodCreate stores a new session, replacing any existing session with the same ID
	MethodCreate = "create"

	methodValidate  = "validate"
	methodList      = "list"
	methodTouch     = "touch"
	methodRevoke    = "revoke"
	methodRevokeAll = "revokeAll"
	methodRemember  = "remember"
)

// Session is a browser in which a user is signed in to Pocket ID
type Session struct {
	ID string
	// TokenHash is the SHA-256 hash of the secret part of the session token, which is never stored itself
	TokenHash            string
	AuthenticationMethod string
	IPAddress            string
	UserAgent            string
	Country              string
	City                 string
	CreatedAt            time.Time
	LastActiveAt         time.Time
	ExpiresAt            time.Time
	// IdleTimeout ends the session once it has gone unused for this long, and is zero for a session that lasts until ExpiresAt
	IdleTimeout time.Duration
}

// endsAt is when the session stops being valid, which comes before ExpiresAt once a session with an idle timeout goes unused
func (s Session) endsAt() time.Time {
	idleEnd := s.LastActiveAt.Add(s.IdleTimeout)
	if s.IdleTimeout == 0 || s.ExpiresAt.Before(idleEnd) {
		return s.ExpiresAt
	}
	return idleEnd
}

type actorState struct {
	Sessions map[string]Session
}

type validateRequest struct {
	SessionID string
	TokenHash string
}

type validateResponse struct {
	Valid   bool
	Session Session
}

type sessionRequest struct {
	SessionID string
}

type revokeAllRequest struct {
	ExceptSessionID string
}

type sessionsActor struct {
	client actor.Client[actorState]
}

// newActor allocates the actor for the sessions of one user
// It satisfies actor.Factory
func newActor(actorID string, service *actor.Service) actor.Actor {
	return &sessionsActor{
		client: actor.NewActorClient[actorState](ActorType, actorID, service),
	}
}

// Invoke implements actor.ActorInvoke
func (a *sessionsActor) Invoke(ctx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case MethodCreate:
		var session Session
		if err := decode(data, method, &session); err != nil {
			return nil, err
		}
		return nil, a.create(ctx, session)
	case methodTouch:
		var req sessionRequest
		if err := decode(data, method, &req); err != nil {
			return nil, err
		}
		return nil, a.touch(ctx, req.SessionID)
	case methodRevoke:
		var req sessionRequest
		if err := decode(data, method, &req); err != nil {
			return nil, err
		}
		return a.revoke(ctx, req.SessionID)
	case methodRevokeAll:
		var req revokeAllRequest
		if err := decode(data, method, &req); err != nil {
			return nil, err
		}
		return a.revokeAll(ctx, req.ExceptSessionID)
	case methodRemember:
		var req sessionRequest
		if err := decode(data, method, &req); err != nil {
			return nil, err
		}
		return a.remember(ctx, req.SessionID)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

// Peek implements actor.ActorPeek
// Every authenticated request validates its session, so validation is a peek that runs concurrently and is served from the activation's state cache
func (a *sessionsActor) Peek(ctx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case methodValidate:
		var req validateRequest
		if err := decode(data, method, &req); err != nil {
			return nil, err
		}
		return a.validate(ctx, req)
	case methodList:
		return a.list(ctx)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

func (a *sessionsActor) create(ctx context.Context, session Session) error {
	state, err := a.loadForWrite(ctx)
	if err != nil {
		return err
	}

	state.Sessions[session.ID] = session
	return a.save(ctx, state)
}

func (a *sessionsActor) validate(ctx context.Context, req validateRequest) (validateResponse, error) {
	state, err := a.client.GetState(ctx)
	if err != nil {
		return validateResponse{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	session, ok := state.Sessions[req.SessionID]
	if !ok || !session.endsAt().After(time.Now()) {
		return validateResponse{}, nil
	}
	if subtle.ConstantTimeCompare([]byte(session.TokenHash), []byte(req.TokenHash)) != 1 {
		return validateResponse{}, nil
	}

	return validateResponse{Valid: true, Session: session}, nil
}

func (a *sessionsActor) list(ctx context.Context) ([]Session, error) {
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}

	now := time.Now()
	sessions := make([]Session, 0, len(state.Sessions))
	for _, session := range state.Sessions {
		if session.endsAt().After(now) {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (a *sessionsActor) touch(ctx context.Context, sessionID string) error {
	state, err := a.loadForWrite(ctx)
	if err != nil {
		return err
	}

	// The session may have been revoked since it was validated
	session, ok := state.Sessions[sessionID]
	if !ok {
		return nil
	}

	session.LastActiveAt = time.Now()
	state.Sessions[sessionID] = session
	return a.save(ctx, state)
}

// revoke returns the revoked session, or the zero Session when it does not exist
func (a *sessionsActor) revoke(ctx context.Context, sessionID string) (Session, error) {
	state, err := a.loadForWrite(ctx)
	if err != nil {
		return Session{}, err
	}

	session, ok := state.Sessions[sessionID]
	if !ok {
		return Session{}, nil
	}

	delete(state.Sessions, sessionID)
	return session, a.save(ctx, state)
}

// revokeAll returns how many sessions it revoked
func (a *sessionsActor) revokeAll(ctx context.Context, exceptSessionID string) (int, error) {
	state, err := a.loadForWrite(ctx)
	if err != nil {
		return 0, err
	}

	count := len(state.Sessions)
	maps.DeleteFunc(state.Sessions, func(id string, _ Session) bool {
		return id != exceptSessionID
	})
	return count - len(state.Sessions), a.save(ctx, state)
}

// remember drops the idle timeout of a session, so it lasts until it expires
// It returns the zero Session when the session does not exist
func (a *sessionsActor) remember(ctx context.Context, sessionID string) (Session, error) {
	state, err := a.loadForWrite(ctx)
	if err != nil {
		return Session{}, err
	}

	session, ok := state.Sessions[sessionID]
	if !ok {
		return Session{}, nil
	}

	session.IdleTimeout = 0
	state.Sessions[sessionID] = session
	return session, a.save(ctx, state)
}

// loadForWrite returns a copy of the state that can be modified without touching the client's state cache, which must keep the old state if saving fails
func (a *sessionsActor) loadForWrite(ctx context.Context) (actorState, error) {
	state, err := a.client.GetState(ctx)
	if err != nil {
		return actorState{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	sessions := make(map[string]Session, len(state.Sessions)+1)
	maps.Copy(sessions, state.Sessions)
	return actorState{Sessions: sessions}, nil
}

// save drops ended sessions and persists the rest until the last of them ends
func (a *sessionsActor) save(ctx context.Context, state actorState) error {
	now := time.Now()
	var lastExpiry time.Time
	maps.DeleteFunc(state.Sessions, func(_ string, session Session) bool {
		return !session.endsAt().After(now)
	})
	for _, session := range state.Sessions {
		if session.endsAt().After(lastExpiry) {
			lastExpiry = session.endsAt()
		}
	}

	if len(state.Sessions) == 0 {
		err := a.client.DeleteState(ctx)
		if err != nil && !errors.Is(err, actor.ErrStateNotFound) {
			return fmt.Errorf("error deleting actor state: %w", err)
		}
		return nil
	}

	err := a.client.SetState(ctx, state, &actor.SetStateOpts{TTL: lastExpiry.Sub(now)})
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}
	return nil
}

func decode(data actor.Envelope, method string, target any) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", method)
	}
	err := data.Decode(target)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", method, err)
	}
	return nil
}
