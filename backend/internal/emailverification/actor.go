package emailverification

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// ActorType is the actor type for email verification state
const ActorType = "EmailVerification"

const (
	methodIssue   = "issue"
	methodConsume = "consume"
	methodDiscard = "discard"
	methodRestore = "restore"
)

type consumeStatus string

const (
	consumeOK       consumeStatus = "ok"
	consumeNotFound consumeStatus = "not_found"
)

// State is the persisted verification state for one user
type State struct {
	TokenHash string
	Email     string
	ExpiresAt time.Time
}

type tokenRequest struct {
	TokenHash string
}

type consumeResponse struct {
	Status consumeStatus
	State  State
}

type emailVerificationActor struct {
	client actor.Client[State]
}

// NewActor allocates the email verification actor for a user
func NewActor(actorID string, service *actor.Service) actor.Actor {
	return &emailVerificationActor{
		client: actor.NewActorClient[State](ActorType, actorID, service),
	}
}

// Invoke implements actor.ActorInvoke
func (a *emailVerificationActor) Invoke(ctx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case methodIssue:
		return nil, a.issue(ctx, data)
	case methodConsume:
		return a.consume(ctx, data)
	case methodDiscard:
		return nil, a.discard(ctx, data)
	case methodRestore:
		return nil, a.restore(ctx, data)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

func (a *emailVerificationActor) issue(ctx context.Context, data actor.Envelope) error {
	state, err := decodeState(data, methodIssue)
	if err != nil {
		return err
	}

	return a.setState(ctx, state)
}

func (a *emailVerificationActor) consume(ctx context.Context, data actor.Envelope) (consumeResponse, error) {
	request, err := decodeTokenRequest(data, methodConsume)
	if err != nil {
		return consumeResponse{}, err
	}

	state, err := a.client.GetState(ctx)
	if err != nil {
		return consumeResponse{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	// Compare if the hash matches
	if state.TokenHash == "" || state.ExpiresAt.Before(time.Now()) ||
		subtle.ConstantTimeCompare([]byte(state.TokenHash), []byte(request.TokenHash)) != 1 {
		return consumeResponse{Status: consumeNotFound}, nil
	}

	err = a.client.DeleteState(ctx)
	if err != nil {
		return consumeResponse{}, fmt.Errorf("error deleting actor state: %w", err)
	}

	return consumeResponse{
		Status: consumeOK,
		State:  state,
	}, nil
}

func (a *emailVerificationActor) discard(ctx context.Context, data actor.Envelope) error {
	request, err := decodeTokenRequest(data, methodDiscard)
	if err != nil {
		return err
	}

	state, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}

	// Only discard if the token hash matches, to avoid discarding a newer token that may have been issued after the one being discarded
	if state.TokenHash == "" || subtle.ConstantTimeCompare([]byte(state.TokenHash), []byte(request.TokenHash)) != 1 {
		return nil
	}

	err = a.client.DeleteState(ctx)
	if err != nil && !errors.Is(err, actor.ErrStateNotFound) {
		return fmt.Errorf("error deleting actor state: %w", err)
	}

	return nil
}

func (a *emailVerificationActor) restore(ctx context.Context, data actor.Envelope) error {
	state, err := decodeState(data, methodRestore)
	if err != nil {
		return err
	}

	current, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}

	// Preserve a newer verification request that may have been issued after consumption
	if current.TokenHash != "" {
		return nil
	}

	return a.setState(ctx, state)
}

func (a *emailVerificationActor) setState(ctx context.Context, state State) error {
	ttl := time.Until(state.ExpiresAt)
	if ttl <= 0 {
		return nil
	}

	err := a.client.SetState(ctx, state, &actor.SetStateOpts{TTL: ttl})
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}

	return nil
}

func decodeState(data actor.Envelope, method string) (State, error) {
	if data == nil {
		return State{}, fmt.Errorf("request body is empty for method '%s'", method)
	}

	var state State
	err := data.Decode(&state)
	if err != nil {
		return State{}, fmt.Errorf("request body is not valid for method '%s': %w", method, err)
	}

	return state, nil
}

func decodeTokenRequest(data actor.Envelope, method string) (tokenRequest, error) {
	if data == nil {
		return tokenRequest{}, fmt.Errorf("request body is empty for method '%s'", method)
	}

	var request tokenRequest
	err := data.Decode(&request)
	if err != nil {
		return tokenRequest{}, fmt.Errorf("request body is not valid for method '%s': %w", method, err)
	}

	return request, nil
}
