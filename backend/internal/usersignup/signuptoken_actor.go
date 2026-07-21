package usersignup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// Signup tokens are stored in a single singleton actor that holds all of them in its state.
// The state itself has no TTL: the actor keeps a single alarm scheduled for the moment the earliest-expiring token expires. When that alarm fires, the actor purges every expired token and, if any tokens remain, reschedules the alarm for the next earliest expiration. This replaces the periodic cleanup job.
// Because it's a singleton, read-only operations (such as listing tokens) are served via Peek, while mutations (create, delete, consume, release) go through Invoke.
//
// Consuming a token must happen outside of a DB transaction (invoking an actor while a transaction is open would deadlock on SQLite): the caller invokes the actor to atomically increment the usage count, performs the rest of its work in a transaction, and, on failure, compensates by invoking the actor again to decrement the usage count (best-effort).

// SignupTokenActorType is the actor type for the signup token singleton actor
const SignupTokenActorType = "SignupToken"

// cleanupAlarmName is the name of the alarm used to purge expired tokens
const cleanupAlarmName = "cleanup"

// Methods exposed by the signup token actor
const (
	signupTokenMethodCreate  = "create"
	signupTokenMethodDelete  = "delete"
	signupTokenMethodConsume = "consume"
	signupTokenMethodRelease = "release"
	signupTokenMethodReplace = "replace"
	signupTokenMethodList    = "list"
)

// signupTokenConsumeStatus is the outcome of a "consume" invocation.
// It's returned as part of the response payload rather than as a Go error, because errors lose their concrete type when they cross the actor invocation boundary (which may be a different host in a cluster).
type signupTokenConsumeStatus string

const (
	signupTokenConsumeOK           signupTokenConsumeStatus = "ok"
	signupTokenConsumeNotFound     signupTokenConsumeStatus = "not_found"
	signupTokenConsumeExpired      signupTokenConsumeStatus = "expired"
	signupTokenConsumeLimitReached signupTokenConsumeStatus = "limit_reached"
)

// storedSignupToken is a single signup token as held in the actor state
type storedSignupToken struct {
	ID           string    `msgpack:"id"`
	Token        string    `msgpack:"token"`
	ExpiresAt    time.Time `msgpack:"expiresAt"`
	UsageLimit   int       `msgpack:"usageLimit"`
	UsageCount   int       `msgpack:"usageCount"`
	UserGroupIDs []string  `msgpack:"userGroupIds"`
	CreatedAt    time.Time `msgpack:"createdAt"`
}

func (t storedSignupToken) isExpired(now time.Time) bool {
	return t.ExpiresAt.Before(now)
}

func (t storedSignupToken) isUsageLimitReached() bool {
	return t.UsageCount >= t.UsageLimit
}

// signupTokenActorState is the persisted state of the signup token singleton actor.
// Tokens are keyed by their token value.
type signupTokenActorState struct {
	Tokens map[string]storedSignupToken `msgpack:"tokens"`
}

// removeExpired deletes every token that has expired and returns the number removed.
func (s *signupTokenActorState) removeExpired(now time.Time) (removed int) {
	for k, t := range s.Tokens {
		if t.isExpired(now) {
			delete(s.Tokens, k)
			removed++
		}
	}

	return removed
}

// earliestExpiration returns the earliest expiration time among all tokens, and whether there's at least one token.
func (s *signupTokenActorState) earliestExpiration() (earliest time.Time, found bool) {
	for _, t := range s.Tokens {
		if !found || t.ExpiresAt.Before(earliest) {
			earliest = t.ExpiresAt
			found = true
		}
	}

	return earliest, found
}

// Payloads for the actor methods

type signupTokenBootstrap struct {
	Tokens []storedSignupToken `msgpack:"tokens"`
}

type signupTokenDeleteRequest struct {
	ID string `msgpack:"id"`
}

type signupTokenConsumeRequest struct {
	Token string `msgpack:"token"`
}

type signupTokenConsumeResponse struct {
	Status       signupTokenConsumeStatus `msgpack:"status"`
	UserGroupIDs []string                 `msgpack:"userGroupIds"`
}

type signupTokenReleaseRequest struct {
	Token string `msgpack:"token"`
}

type signupTokenReplaceRequest struct {
	Tokens []storedSignupToken `msgpack:"tokens"`
}

type signupTokenListResponse struct {
	Tokens []storedSignupToken `msgpack:"tokens"`
}

// signupTokenActor is the singleton actor that manages all signup tokens
type signupTokenActor struct {
	log    *slog.Logger
	client actor.Client[*signupTokenActorState]
}

// NewSignupTokenActor allocates a new signup token actor.
// It satisfies actor.Factory.
func NewSignupTokenActor(actorID string, service *actor.Service) actor.Actor {
	return &signupTokenActor{
		log: slog.With(
			slog.String("scope", "actor"),
			slog.String("actorType", SignupTokenActorType),
			slog.String("actorID", actorID),
		),
		client: actor.NewActorClient[*signupTokenActorState](SignupTokenActorType, actorID, service),
	}
}

// Bootstrap implements actor.ActorBootstrapper for the singleton actor.
// On first startup it seeds the state from the tokens migrated from the database; on subsequent startups it just makes sure the cleanup alarm is scheduled.
func (a *signupTokenActor) Bootstrap(parentCtx context.Context, data actor.Envelope) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}

	// If we already have a state, just make sure the cleanup alarm is scheduled and we're done
	if state != nil {
		return a.scheduleCleanup(parentCtx, state)
	}

	// Initialize the state, seeding it from the migrated tokens (if any)
	state = &signupTokenActorState{
		Tokens: map[string]storedSignupToken{},
	}
	if data != nil {
		payload := signupTokenBootstrap{}
		err = data.Decode(&payload)
		if err != nil {
			return fmt.Errorf("request body is not valid for bootstrap: %w", err)
		}
		for _, t := range payload.Tokens {
			state.Tokens[t.Token] = t
		}
	}

	// Don't carry over tokens that have already expired
	state.removeExpired(time.Now())

	ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err = a.client.SetState(ctx, state, nil)
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}

	return a.scheduleCleanup(parentCtx, state)
}

// Peek implements actor.ActorPeek for read-only operations
func (a *signupTokenActor) Peek(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	if method != signupTokenMethodList {
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}

	return signupTokenListResponse{Tokens: collectTokens(state)}, nil
}

// Invoke implements actor.ActorInvoke for mutating operations
func (a *signupTokenActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case signupTokenMethodCreate:
		return a.create(parentCtx, data)
	case signupTokenMethodDelete:
		return nil, a.delete(parentCtx, data)
	case signupTokenMethodConsume:
		return a.consume(parentCtx, data)
	case signupTokenMethodRelease:
		return nil, a.release(parentCtx, data)
	case signupTokenMethodReplace:
		return nil, a.replace(parentCtx, data)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

// Alarm implements actor.ActorAlarm: it purges expired tokens and reschedules the alarm.
func (a *signupTokenActor) Alarm(parentCtx context.Context, name string, data actor.Envelope) error {
	if name != cleanupAlarmName {
		return common.ErrUnsupportedActorMethod{Method: name}
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}
	if state == nil {
		return nil
	}

	removed := state.removeExpired(time.Now())
	if removed > 0 {
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		err = a.client.SetState(ctx, state, nil)
		if err != nil {
			return fmt.Errorf("error saving actor state: %w", err)
		}
		a.log.InfoContext(parentCtx, "Purged expired signup tokens", slog.Int("count", removed))
	}

	return a.scheduleCleanup(parentCtx, state)
}

func (a *signupTokenActor) create(parentCtx context.Context, data actor.Envelope) (storedSignupToken, error) {
	if data == nil {
		return storedSignupToken{}, fmt.Errorf("request body is empty for method '%s'", signupTokenMethodCreate)
	}

	var token storedSignupToken
	err := data.Decode(&token)
	if err != nil {
		return storedSignupToken{}, fmt.Errorf("request body is not valid for method '%s': %w", signupTokenMethodCreate, err)
	}

	state, err := a.mustGetState(parentCtx)
	if err != nil {
		return storedSignupToken{}, err
	}

	state.Tokens[token.Token] = token

	err = a.saveState(parentCtx, state)
	if err != nil {
		return storedSignupToken{}, err
	}

	err = a.scheduleCleanup(parentCtx, state)
	if err != nil {
		return storedSignupToken{}, err
	}

	return token, nil
}

func (a *signupTokenActor) delete(parentCtx context.Context, data actor.Envelope) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", signupTokenMethodDelete)
	}

	var req signupTokenDeleteRequest
	err := data.Decode(&req)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", signupTokenMethodDelete, err)
	}

	state, err := a.mustGetState(parentCtx)
	if err != nil {
		return err
	}

	// Tokens are keyed by their value, so find the one matching the given ID
	var deleted bool
	for k, t := range state.Tokens {
		if t.ID == req.ID {
			delete(state.Tokens, k)
			deleted = true
			break
		}
	}
	if !deleted {
		return nil
	}

	err = a.saveState(parentCtx, state)
	if err != nil {
		return err
	}

	return a.scheduleCleanup(parentCtx, state)
}

func (a *signupTokenActor) consume(parentCtx context.Context, data actor.Envelope) (signupTokenConsumeResponse, error) {
	if data == nil {
		return signupTokenConsumeResponse{}, fmt.Errorf("request body is empty for method '%s'", signupTokenMethodConsume)
	}

	var req signupTokenConsumeRequest
	err := data.Decode(&req)
	if err != nil {
		return signupTokenConsumeResponse{}, fmt.Errorf("request body is not valid for method '%s': %w", signupTokenMethodConsume, err)
	}

	state, err := a.mustGetState(parentCtx)
	if err != nil {
		return signupTokenConsumeResponse{}, err
	}

	token, ok := state.Tokens[req.Token]
	if !ok {
		return signupTokenConsumeResponse{Status: signupTokenConsumeNotFound}, nil
	}
	if token.isExpired(time.Now()) {
		return signupTokenConsumeResponse{Status: signupTokenConsumeExpired}, nil
	}
	if token.isUsageLimitReached() {
		return signupTokenConsumeResponse{Status: signupTokenConsumeLimitReached}, nil
	}

	// Atomically consume one use of the token
	token.UsageCount++
	state.Tokens[req.Token] = token

	err = a.saveState(parentCtx, state)
	if err != nil {
		return signupTokenConsumeResponse{}, err
	}

	return signupTokenConsumeResponse{
		Status:       signupTokenConsumeOK,
		UserGroupIDs: token.UserGroupIDs,
	}, nil
}

func (a *signupTokenActor) release(parentCtx context.Context, data actor.Envelope) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", signupTokenMethodRelease)
	}

	var req signupTokenReleaseRequest
	err := data.Decode(&req)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", signupTokenMethodRelease, err)
	}

	state, err := a.mustGetState(parentCtx)
	if err != nil {
		return err
	}

	token, ok := state.Tokens[req.Token]
	if !ok || token.UsageCount <= 0 {
		// The token is gone (for example, expired and purged) or was never consumed: nothing to compensate
		return nil
	}

	token.UsageCount--
	state.Tokens[req.Token] = token

	return a.saveState(parentCtx, state)
}

func (a *signupTokenActor) replace(parentCtx context.Context, data actor.Envelope) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", signupTokenMethodReplace)
	}

	var req signupTokenReplaceRequest
	err := data.Decode(&req)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", signupTokenMethodReplace, err)
	}

	state := &signupTokenActorState{Tokens: make(map[string]storedSignupToken, len(req.Tokens))}
	for _, t := range req.Tokens {
		state.Tokens[t.Token] = t
	}

	err = a.saveState(parentCtx, state)
	if err != nil {
		return err
	}

	return a.scheduleCleanup(parentCtx, state)
}

// scheduleCleanup sets the cleanup alarm to fire when the earliest-expiring token expires, or deletes it when there are no tokens left.
func (a *signupTokenActor) scheduleCleanup(parentCtx context.Context, state *signupTokenActorState) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	earliest, found := state.earliestExpiration()
	if !found {
		// No tokens: remove the alarm (if any)
		err := a.client.DeleteAlarm(ctx, cleanupAlarmName)
		if err != nil && !errors.Is(err, actor.ErrAlarmNotFound) {
			return fmt.Errorf("error deleting cleanup alarm: %w", err)
		}
		return nil
	}

	err := a.client.SetAlarm(ctx, cleanupAlarmName, actor.AlarmProperties{DueTime: earliest})
	if err != nil {
		return fmt.Errorf("error setting cleanup alarm: %w", err)
	}
	return nil
}

// mustGetState retrieves the actor state, initializing an empty one if it doesn't exist yet.
func (a *signupTokenActor) mustGetState(parentCtx context.Context) (*signupTokenActorState, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}
	if state == nil {
		state = &signupTokenActorState{Tokens: map[string]storedSignupToken{}}
	} else if state.Tokens == nil {
		state.Tokens = map[string]storedSignupToken{}
	}
	return state, nil
}

func (a *signupTokenActor) saveState(parentCtx context.Context, state *signupTokenActorState) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err := a.client.SetState(ctx, state, nil)
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}
	return nil
}

// collectTokens returns all tokens in the state as a slice.
func collectTokens(state *signupTokenActorState) []storedSignupToken {
	if state == nil {
		return nil
	}
	tokens := make([]storedSignupToken, 0, len(state.Tokens))
	for _, t := range state.Tokens {
		tokens = append(tokens, t)
	}
	return tokens
}
