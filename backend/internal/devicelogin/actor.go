package devicelogin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	requestActorType          = "device-login-request"
	requestActorMethodCreate  = "create"
	requestActorMethodInspect = "inspect"
	requestActorMethodPoll    = "poll"
	requestActorMethodDecide  = "decide"
	requestActorMethodConsume = "consume"
)

type requestActorResultCode string

const (
	requestActorResultNone      requestActorResultCode = ""
	requestActorResultCollision requestActorResultCode = "collision"
	requestActorResultInvalid   requestActorResultCode = "invalid"
	requestActorResultDenied    requestActorResultCode = "denied"
)

type requestActorState struct {
	Code            string
	DeviceTokenHash string
	Status          RequestStatus
	ExpiresAt       time.Time
	IPAddress       string
	UserAgent       string
	UserID          string
}

type requestActorResult struct {
	Code      requestActorResultCode
	Status    RequestStatus
	UserCode  string
	IPAddress string
	UserAgent string
	ExpiresAt time.Time
	UserID    string
}

type requestActorCreateInput struct {
	Code            string
	DeviceTokenHash string
	IPAddress       string
	UserAgent       string
}

type requestActorPollInput struct {
	DeviceTokenHash string
}

type requestActorDecisionInput struct {
	Decision string
	UserID   string
}

type requestActorConsumeInput struct {
	DeviceTokenHash string
}

type requestActor struct {
	client actor.Client[requestActorState]
}

func newRequestActor(actorID string, actorService *actor.Service) actor.Actor {
	return &requestActor{
		client: actor.NewActorClient[requestActorState](requestActorType, actorID, actorService),
	}
}

func (a *requestActor) Invoke(ctx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case requestActorMethodCreate:
		var input requestActorCreateInput
		err := decodeActorInput(data, &input)
		if err != nil {
			return nil, err
		}
		return a.create(ctx, input)
	case requestActorMethodDecide:
		var input requestActorDecisionInput
		err := decodeActorInput(data, &input)
		if err != nil {
			return nil, err
		}
		return a.decide(ctx, input)
	case requestActorMethodConsume:
		var input requestActorConsumeInput
		err := decodeActorInput(data, &input)
		if err != nil {
			return nil, err
		}
		return a.consume(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported device login actor method %q", method)
	}
}

func (a *requestActor) Peek(ctx context.Context, method string, data actor.Envelope) (any, error) {
	switch method {
	case requestActorMethodInspect:
		return a.inspect(ctx)
	case requestActorMethodPoll:
		var input requestActorPollInput
		if err := decodeActorInput(data, &input); err != nil {
			return nil, err
		}
		return a.poll(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported device login actor peek method %q", method)
	}
}

func (a *requestActor) create(ctx context.Context, input requestActorCreateInput) (requestActorResult, error) {
	// Reject invalid initialization before touching durable state
	if input.Code == "" || input.DeviceTokenHash == "" {
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}

	// Preserve an existing live request when the short user code collides
	state, err := a.client.GetState(ctx)
	if err != nil {
		return requestActorResult{}, fmt.Errorf("failed to load device login actor state: %w", err)
	}
	if state.Code != "" {
		if state.ExpiresAt.After(time.Now()) {
			return requestActorResult{Code: requestActorResultCollision}, nil
		}
	}

	expiresAt := time.Now().Round(time.Second).Add(RequestDuration)

	// Persist the request until its public expiry
	state = requestActorState{
		Code:            input.Code,
		DeviceTokenHash: input.DeviceTokenHash,
		Status:          RequestStatusPending,
		ExpiresAt:       expiresAt,
		IPAddress:       input.IPAddress,
		UserAgent:       input.UserAgent,
	}
	if err = a.persistState(ctx, state); err != nil {
		return requestActorResult{}, err
	}

	return requestActorResult{Status: state.Status, ExpiresAt: state.ExpiresAt}, nil
}

func (a *requestActor) inspect(ctx context.Context) (requestActorResult, error) {
	// Only pending live requests may reveal requester metadata or be decided
	state, valid, err := a.liveState(ctx)
	if err != nil {
		return requestActorResult{}, err
	}
	if !valid || state.Status != RequestStatusPending {
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}

	return requestActorResult{
		Status:    state.Status,
		UserCode:  state.Code,
		IPAddress: state.IPAddress,
		UserAgent: state.UserAgent,
		ExpiresAt: state.ExpiresAt,
	}, nil
}

func (a *requestActor) poll(ctx context.Context, input requestActorPollInput) (requestActorResult, error) {
	// Validate the request lifetime and the high-entropy device binding on every poll
	state, err := a.loadState(ctx)
	if err != nil {
		return requestActorResult{}, err
	}
	if state.Code == "" || !utils.ConstantTimeStringEqual(state.DeviceTokenHash, input.DeviceTokenHash) {
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}

	result := requestActorResult{Status: state.Status, ExpiresAt: state.ExpiresAt}
	switch state.Status {
	case RequestStatusPending:
		if !state.ExpiresAt.After(time.Now()) {
			return requestActorResult{Code: requestActorResultInvalid}, nil
		}
		return result, nil
	case RequestStatusApproved:
		if state.UserID == "" || !state.ExpiresAt.After(time.Now()) {
			return requestActorResult{Code: requestActorResultInvalid}, nil
		}
		result.UserID = state.UserID
		return result, nil
	case RequestStatusDenied:
		if !state.ExpiresAt.After(time.Now()) {
			return requestActorResult{Code: requestActorResultInvalid}, nil
		}
		result.Code = requestActorResultDenied
		return result, nil
	default:
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}
}

func (a *requestActor) decide(ctx context.Context, input requestActorDecisionInput) (requestActorResult, error) {
	// Serialize every decision against exchange and require the request to still be pending
	state, valid, err := a.liveState(ctx)
	if err != nil {
		return requestActorResult{}, err
	}
	if !valid || state.Status != RequestStatusPending {
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}

	switch input.Decision {
	case "deny":
		state.Status = RequestStatusDenied
	case "approve":
		state.Status = RequestStatusApproved
		state.UserID = input.UserID
	default:
		return requestActorResult{}, fmt.Errorf("unsupported device login decision %q", input.Decision)
	}

	// Preserve the original expiry when committing the terminal decision
	if err = a.persistState(ctx, state); err != nil {
		return requestActorResult{}, err
	}

	return requestActorResult{Status: state.Status, ExpiresAt: state.ExpiresAt}, nil
}

func (a *requestActor) consume(ctx context.Context, input requestActorConsumeInput) (requestActorResult, error) {
	// Serialize consume attempts and validate the device binding inside the actor turn
	state, err := a.loadState(ctx)
	if err != nil {
		return requestActorResult{}, err
	}
	if state.Code == "" || !utils.ConstantTimeStringEqual(state.DeviceTokenHash, input.DeviceTokenHash) || !state.ExpiresAt.After(time.Now()) {
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}
	switch state.Status {
	case RequestStatusPending:
		return requestActorResult{Status: state.Status, ExpiresAt: state.ExpiresAt}, nil
	case RequestStatusDenied:
		return requestActorResult{Code: requestActorResultDenied, Status: state.Status, ExpiresAt: state.ExpiresAt}, nil
	case RequestStatusApproved:
		if state.UserID == "" {
			return requestActorResult{Code: requestActorResultInvalid}, nil
		}
		// Delete the approved request before returning so only one exchange can continue
		if err = a.client.DeleteState(ctx); err != nil {
			return requestActorResult{}, fmt.Errorf("failed to delete consumed device login actor state: %w", err)
		}
		return requestActorResult{
			Status:    state.Status,
			ExpiresAt: state.ExpiresAt,
		}, nil
	default:
		return requestActorResult{Code: requestActorResultInvalid}, nil
	}
}

func (a *requestActor) liveState(ctx context.Context) (requestActorState, bool, error) {
	state, err := a.loadState(ctx)
	if err != nil {
		return requestActorState{}, false, err
	}
	if state.Code == "" {
		return state, false, nil
	}
	return state, state.ExpiresAt.After(time.Now()), nil
}

func (a *requestActor) loadState(ctx context.Context) (requestActorState, error) {
	state, err := a.client.GetState(ctx)
	if err != nil {
		return requestActorState{}, fmt.Errorf("failed to load device login actor state: %w", err)
	}
	return state, nil
}

func (a *requestActor) persistState(ctx context.Context, state requestActorState) error {
	ttl := time.Until(state.ExpiresAt)
	if ttl <= 0 {
		return errors.New("cannot persist expired device login actor state")
	}

	err := a.client.SetState(ctx, state, &actor.SetStateOpts{
		TTL: ttl,
	})
	if err != nil {
		return fmt.Errorf("failed to persist device login actor state: %w", err)
	}
	return nil
}

func decodeActorInput(data actor.Envelope, target any) error {
	if data == nil {
		return errors.New("device login actor input is missing")
	}

	err := data.Decode(target)
	if err != nil {
		return fmt.Errorf("failed to decode device login actor input: %w", err)
	}
	return nil
}
