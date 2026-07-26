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

// Signup tokens are stored entirely in the actor state store.
// Each token is its own actor, whose actor ID is the token value itself.
// The state is persisted with a TTL equal to the token's lifetime, so it's purged automatically when the token expires (there's no separate cleanup job and no expiration alarm).
// Listing tokens uses ListStates, which only returns states that haven't expired yet.

// SignupTokenActorType is the actor type for the signup token actor
const SignupTokenActorType = "SignupToken"

// Methods exposed by the signup token actor
// Because we cannot invoke an actor while a DB transaction is open (that would deadlock on SQLite), consuming a token is done by invoking the actor first (which atomically validates it and increments its usage count), and only afterwards performing the remaining work.
// On failure, the caller compensates by releasing the token via the "release" method as best-effort.
const (
	// SignupTokenMethodCreate stores a new signup token, replacing any existing state
	SignupTokenMethodCreate = "create"
	// SignupTokenMethodDelete removes a signup token
	SignupTokenMethodDelete = "delete"

	signupTokenMethodMigrate = "migrate"
	signupTokenMethodConsume = "consume"
	signupTokenMethodRelease = "release"
)

// signupTokenConsumeStatus is the outcome of a "consume" invocation.
type signupTokenConsumeStatus string

const (
	// signupTokenConsumeOK indicates the token was valid and one use has been consumed
	signupTokenConsumeOK signupTokenConsumeStatus = "ok"
	// signupTokenConsumeNotFound indicates the token doesn't exist (or has expired)
	signupTokenConsumeNotFound signupTokenConsumeStatus = "not_found"
	// signupTokenConsumeLimitReached indicates the token has no uses left
	signupTokenConsumeLimitReached signupTokenConsumeStatus = "limit_reached"
)

// SignupTokenState is the persisted state of a signup token actor.
// The token value itself is the actor's ID, so it isn't repeated here.
type SignupTokenState struct {
	ID           string
	UsageLimit   int
	UsageCount   int
	UserGroupIDs []string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// signupTokenConsumeResponse is the response of the "consume" method
type signupTokenConsumeResponse struct {
	Status signupTokenConsumeStatus
	// UserGroupIDs is set only when Status is "ok", and contains the groups the new user should join
	UserGroupIDs []string
}

// signupTokenActor is the actor that manages a single signup token
type signupTokenActor struct {
	log    *slog.Logger
	client actor.Client[SignupTokenState]
}

// NewSignupTokenActor allocates a new signup token actor
// It satisfies actor.Factory
func NewSignupTokenActor(actorID string, service *actor.Service) actor.Actor {
	return &signupTokenActor{
		log: slog.With(
			slog.String("scope", "actor"),
			slog.String("actorType", SignupTokenActorType),
		),
		client: actor.NewActorClient[SignupTokenState](SignupTokenActorType, actorID, service),
	}
}

// Invoke implements actor.ActorInvoke
func (a *signupTokenActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case SignupTokenMethodCreate:
		return nil, a.create(parentCtx, data, false)
	case signupTokenMethodMigrate:
		return nil, a.create(parentCtx, data, true)
	case signupTokenMethodConsume:
		return a.consume(parentCtx)
	case signupTokenMethodRelease:
		return nil, a.release(parentCtx)
	case SignupTokenMethodDelete:
		return nil, a.delete(parentCtx)
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}
}

// create stores the token's state.
// When onlyIfMissing is true the write is skipped if the actor already has state: this is used by the one-time migration of the pre-actor tokens, so a token that has already been migrated is never reset.
func (a *signupTokenActor) create(parentCtx context.Context, data actor.Envelope, onlyIfMissing bool) error {
	if data == nil {
		return fmt.Errorf("request body is empty for method '%s'", SignupTokenMethodCreate)
	}

	var state SignupTokenState
	err := data.Decode(&state)
	if err != nil {
		return fmt.Errorf("request body is not valid for method '%s': %w", SignupTokenMethodCreate, err)
	}

	if onlyIfMissing {
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		current, err := a.client.GetState(ctx)
		if err != nil {
			return fmt.Errorf("error retrieving actor state: %w", err)
		}

		// An empty ID means there's no state yet
		if current.ID != "" {
			return nil
		}
	}

	return a.setState(parentCtx, state)
}

// consume atomically validates the token and, if it's still usable, records one more use.
func (a *signupTokenActor) consume(parentCtx context.Context) (signupTokenConsumeResponse, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return signupTokenConsumeResponse{}, fmt.Errorf("error retrieving actor state: %w", err)
	}

	// An empty ID means there's no state: the token doesn't exist (or its state already expired and was purged)
	if state.ID == "" || state.ExpiresAt.Before(time.Now()) {
		return signupTokenConsumeResponse{
			Status: signupTokenConsumeNotFound,
		}, nil
	}

	if state.UsageCount >= state.UsageLimit {
		return signupTokenConsumeResponse{
			Status: signupTokenConsumeLimitReached,
		}, nil
	}

	// Consume one use of the token
	state.UsageCount++
	err = a.setState(parentCtx, state)
	if err != nil {
		return signupTokenConsumeResponse{}, err
	}

	return signupTokenConsumeResponse{
		Status:       signupTokenConsumeOK,
		UserGroupIDs: state.UserGroupIDs,
	}, nil
}

// release reverts the usage count increment performed while consuming the token, to compensate when the signup could not be completed.
func (a *signupTokenActor) release(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}

	// The token is gone (for example, it expired and was purged) or was never consumed: nothing to compensate
	if state.ID == "" || state.UsageCount <= 0 {
		return nil
	}

	state.UsageCount--
	return a.setState(parentCtx, state)
}

// delete removes the token.
func (a *signupTokenActor) delete(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err := a.client.DeleteState(ctx)
	if err != nil && !errors.Is(err, actor.ErrStateNotFound) {
		// Deleting a token that doesn't exist (for example, one that expired in the meanwhile) already reaches the desired end state
		return fmt.Errorf("error deleting actor state: %w", err)
	}

	return nil
}

// setState saves the state with a TTL matching the token's remaining lifetime, so it's purged automatically once the token expires.
// Saving is skipped if the token has already expired, since there would be nothing left to store.
func (a *signupTokenActor) setState(parentCtx context.Context, state SignupTokenState) error {
	ttl := time.Until(state.ExpiresAt)
	if ttl <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err := a.client.SetState(ctx, state, &actor.SetStateOpts{
		TTL: ttl,
	})
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}

	return nil
}
