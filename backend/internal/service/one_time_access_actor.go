package service

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

// OneTimeAccessTokenActorType is the actor type for the one-time access token actor
const OneTimeAccessTokenActorType = "OneTimeAccessToken"

// Methods exposed by the one-time access token actor
// Because we cannot invoke an actor while a DB transaction is open (that would deadlock on SQLite), consuming a token is done by invoking the actor first (which atomically validates and deletes the token), and only afterwards performing the remaining work. On failure, the caller compensates by restoring the token via the "restore" method.
const (
	oneTimeAccessTokenMethodConsume = "consume"
	oneTimeAccessTokenMethodRestore = "restore"
)

// oneTimeAccessConsumeStatus is the outcome of a "consume" invocation.
type oneTimeAccessConsumeStatus string

const (
	// oneTimeAccessConsumeOK indicates the token was valid and has been consumed
	oneTimeAccessConsumeOK oneTimeAccessConsumeStatus = "ok"
	// oneTimeAccessConsumeNotFound indicates the token doesn't exist (or has expired)
	oneTimeAccessConsumeNotFound oneTimeAccessConsumeStatus = "not_found"
	// oneTimeAccessConsumeDeviceMismatch indicates the provided device token doesn't match
	oneTimeAccessConsumeDeviceMismatch oneTimeAccessConsumeStatus = "device_mismatch"
)

// oneTimeAccessTokenState is the persisted state of a one-time access token actor
type oneTimeAccessTokenState struct {
	UserID      string    `msgpack:"userId"`
	DeviceToken *string   `msgpack:"deviceToken,omitempty"`
	ExpiresAt   time.Time `msgpack:"expiresAt"`
}

// oneTimeAccessConsumeRequest is the payload for the "consume" method
type oneTimeAccessConsumeRequest struct {
	DeviceToken string `msgpack:"deviceToken"`
}

// oneTimeAccessConsumeResponse is the response of the "consume" method
type oneTimeAccessConsumeResponse struct {
	Status oneTimeAccessConsumeStatus `msgpack:"status"`
	// State is included only when Status is "ok", so the caller can restore it if a later step fails
	State oneTimeAccessTokenState `msgpack:"state"`
}

// oneTimeAccessTokenActor is the actor that manages a single one-time access token
type oneTimeAccessTokenActor struct {
	log    *slog.Logger
	client actor.Client[oneTimeAccessTokenState]
}

// NewOneTimeAccessTokenActor allocates a new one-time access token actor.
// It satisfies actor.Factory.
func NewOneTimeAccessTokenActor(actorID string, service *actor.Service) actor.Actor {
	return &oneTimeAccessTokenActor{
		log: slog.With(
			slog.String("scope", "actor"),
			slog.String("actorType", OneTimeAccessTokenActorType),
		),
		client: actor.NewActorClient[oneTimeAccessTokenState](OneTimeAccessTokenActorType, actorID, service),
	}
}

// Invoke implements actor.ActorInvoke
func (a *oneTimeAccessTokenActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case oneTimeAccessTokenMethodConsume:
		return a.consume(parentCtx, data)
	case oneTimeAccessTokenMethodRestore:
		return nil, a.restore(parentCtx, data)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

// consume atomically validates the token and, if valid, deletes it.
func (a *oneTimeAccessTokenActor) consume(parentCtx context.Context, data actor.Envelope) (oneTimeAccessConsumeResponse, error) {
	var req oneTimeAccessConsumeRequest
	if data != nil {
		err := data.Decode(&req)
		if err != nil {
			return oneTimeAccessConsumeResponse{}, fmt.Errorf("request body is not valid for method '%s': %w", oneTimeAccessTokenMethodConsume, err)
		}
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return oneTimeAccessConsumeResponse{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	// An empty UserID means there's no state: the token doesn't exist (or its state already expired and was purged)
	if state.UserID == "" || state.ExpiresAt.Before(time.Now()) {
		return oneTimeAccessConsumeResponse{Status: oneTimeAccessConsumeNotFound}, nil
	}

	// If the token requires a device token, it must match
	// A mismatch leaves the token untouched, mirroring the pre-actor behavior
	if state.DeviceToken != nil && req.DeviceToken != *state.DeviceToken {
		return oneTimeAccessConsumeResponse{Status: oneTimeAccessConsumeDeviceMismatch}, nil
	}

	// The token is valid: delete the state (one-time use)
	ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err = a.client.DeleteState(ctx)
	if err != nil {
		return oneTimeAccessConsumeResponse{}, fmt.Errorf("error deleting actor state: %w", err)
	}

	return oneTimeAccessConsumeResponse{Status: oneTimeAccessConsumeOK, State: state}, nil
}

// restore re-creates the token state, used to compensate when a step after consuming the token fails.
func (a *oneTimeAccessTokenActor) restore(parentCtx context.Context, data actor.Envelope) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", oneTimeAccessTokenMethodRestore)
	}

	var state oneTimeAccessTokenState
	err := data.Decode(&state)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", oneTimeAccessTokenMethodRestore, err)
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
