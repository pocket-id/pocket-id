package onetimeaccess

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// One-time access tokens are stored entirely in the actor state store.
// Each token is its own actor, whose actor ID is the token value itself.
// The state is persisted with a TTL equal to the token's lifetime, so it's purged automatically when the token expires (there's no separate cleanup job).

// TokenActorType is the actor type for the one-time access token actor
const TokenActorType = "OneTimeAccessToken"

// Methods exposed by the one-time access token actor
// Because we cannot invoke an actor while a DB transaction is open (that would deadlock on SQLite), consuming a token is done by invoking the actor first (which atomically validates and deletes the token), and only afterwards performing the remaining work.
// On failure, the caller compensates by restoring the token via the "restore" method as best-effort.
const (
	// TokenMethodRestore stores a token's state, and is also how a consumed token is put back
	TokenMethodRestore = "restore"

	tokenMethodConsume = "consume"
)

// tokenConsumeStatus is the outcome of a "consume" invocation.
type tokenConsumeStatus string

const (
	// tokenConsumeOK indicates the token was valid and has been consumed
	tokenConsumeOK tokenConsumeStatus = "ok"
	// tokenConsumeNotFound indicates the token doesn't exist (or has expired)
	tokenConsumeNotFound tokenConsumeStatus = "not_found"
	// tokenConsumeDeviceMismatch indicates the provided device token doesn't match
	tokenConsumeDeviceMismatch tokenConsumeStatus = "device_mismatch"
)

// TokenState is the persisted state of a one-time access token actor.
// The token value itself is the actor's ID, so it isn't repeated here.
type TokenState struct {
	UserID      string
	DeviceToken *string
	ExpiresAt   time.Time
}

// tokenConsumeRequest is the payload for the "consume" method
type tokenConsumeRequest struct {
	DeviceToken string
}

// tokenConsumeResponse is the response of the "consume" method
type tokenConsumeResponse struct {
	Status tokenConsumeStatus
	// State is included only when Status is "ok", so the caller can restore it if a later step fails
	State TokenState
}

// tokenActor is the actor that manages a single one-time access token
type tokenActor struct {
	log    *slog.Logger
	client actor.Client[TokenState]
}

// NewTokenActor allocates a new one-time access token actor
// It satisfies actor.Factory
func NewTokenActor(actorID string, service *actor.Service) actor.Actor {
	return &tokenActor{
		log: slog.With(
			slog.String("scope", "actor"),
			slog.String("actorType", TokenActorType),
		),
		client: actor.NewActorClient[TokenState](TokenActorType, actorID, service),
	}
}

// Invoke implements actor.ActorInvoke
func (a *tokenActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case tokenMethodConsume:
		return a.consume(parentCtx, data)
	case TokenMethodRestore:
		return nil, a.restore(parentCtx, data)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

// consume atomically validates the token and, if valid, deletes it.
func (a *tokenActor) consume(parentCtx context.Context, data actor.Envelope) (tokenConsumeResponse, error) {
	var req tokenConsumeRequest
	if data != nil {
		err := data.Decode(&req)
		if err != nil {
			return tokenConsumeResponse{}, fmt.Errorf("request body is not valid for method '%s': %w", tokenMethodConsume, err)
		}
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return tokenConsumeResponse{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	// An empty UserID means there's no state: the token doesn't exist (or its state already expired and was purged)
	if state.UserID == "" || state.ExpiresAt.Before(time.Now()) {
		return tokenConsumeResponse{
			Status: tokenConsumeNotFound,
		}, nil
	}

	// If the token requires a device token, it must match
	// A mismatch leaves the token untouched, mirroring the pre-actor behavior
	if state.DeviceToken != nil && req.DeviceToken != *state.DeviceToken {
		return tokenConsumeResponse{
			Status: tokenConsumeDeviceMismatch,
		}, nil
	}

	// The token is valid: delete the state (one-time use)
	ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err = a.client.DeleteState(ctx)
	if err != nil {
		return tokenConsumeResponse{}, fmt.Errorf("error deleting actor state: %w", err)
	}

	return tokenConsumeResponse{
		Status: tokenConsumeOK,
		State:  state,
	}, nil
}

// restore re-creates the token state, used to compensate when a step after consuming the token fails.
func (a *tokenActor) restore(parentCtx context.Context, data actor.Envelope) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", TokenMethodRestore)
	}

	var state TokenState
	err := data.Decode(&state)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", TokenMethodRestore, err)
	}

	// If the token has meanwhile expired, there's nothing to restore
	ttl := time.Until(state.ExpiresAt)
	if ttl <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err = a.client.SetState(ctx, state, &actor.SetStateOpts{
		TTL: ttl,
	})
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}

	return nil
}
