package appconfig

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

// The AppConfig singleton actor maintains the dynamic configuration for the Pocket ID cluster
// Instances of Pocket ID should bootstrap the AppConfig's actor upon startup to ensure the config is loaded (and migrated if needed)
// After startup, Peek can be used for read-only operations such as retrieving the config or listing it

// AppConfigActorType is the actor type for the AppConfig actor
const AppConfigActorType = "AppConfig"

// appConfigActor is a singleton actor that manages the dynamic app configuration
type appConfigActor struct {
	log    *slog.Logger
	client actor.Client[*AppConfigModel]
}

// appConfigActorBootstrap is the type for the payload of the init method
type appConfigActorBootstrap struct {
	LegacyConfig map[string]string
}

// NewAppConfigActor allocates a new AppConfig actor
// It satisfies actor.Factory
func NewAppConfigActor(actorID string, service *actor.Service) actor.Actor {
	log := slog.
		With(
			slog.String("scope", "actor"),
			slog.String("actorType", AppConfigActorType),
			slog.String("actorID", actorID),
		)

	log.Info("AppConfig actor created")

	return &appConfigActor{
		log:    log,
		client: actor.NewActorClient[*AppConfigModel](AppConfigActorType, actorID, service),
	}
}

// Bootstrap implements actor.ActorBootstrapper for the singleton actor
func (a *appConfigActor) Bootstrap(parentCtx context.Context, data actor.Envelope) error {
	// Load the actor state
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving actor state: %w", err)
	}

	// If we already have a state, nothing else to do
	if state != nil {
		return nil
	}

	// Check if the request data contains legacy config to init from
	if data != nil {
		payload := appConfigActorBootstrap{}
		err = data.Decode(&payload)
		if err != nil {
			return fmt.Errorf("request body is not valid for method 'init': %w", err)
		}

		if len(payload.LegacyConfig) > 0 {
			state, err = fromLegacyConfig(payload.LegacyConfig)
			if err != nil {
				return fmt.Errorf("request body is not valid for method 'init': LegacyConfig property could not be parsed: %w", err)
			}
		}
	}

	// If we still have no state, generate a new default config
	if state == nil {
		state = getDefaultConfig()
	}

	// Save the updated state
	ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err = a.client.SetState(ctx, state, nil)
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}

	return nil
}

func (a *appConfigActor) Peek(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	// Only supported method is "get"
	if method != "get" {
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}

	// Load the actor state
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}

	// Return the state
	return state, nil
}

func (a *appConfigActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	// Check the method first
	switch method {
	case "get", "update", "replace":
		// All good
		// Note: we support "get" also via Invoke and not just Peek
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}

	// Load the actor state
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}

	switch method {
	case "get":
		// If the method is "get", just return the actor state, we're done
		// This switch case is a no-op

	case "replace":
		// Replace the entire config
		// The input data must be a dto.AppConfigUpdateDto
		payload := dto.AppConfigUpdateDto{}
		if data == nil {
			return nil, fmt.Errorf("request body is empty for method 'replace': %w", err)
		}
		err = data.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("request body is not valid for method 'replace': %w", err)
		}

		// Update the in-memory data
		// Work on a clone to avoid touching the cached object in case of errors (we'll update after we've committed the state)
		newState := state.Clone()
		newState.Replace(payload)

		// Save the updated state
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		err = a.client.SetState(ctx, state, nil)

		// Update the cached state too
		*state = *newState

	case "update":
		// Update the config
		// The input data must be a map[string]string
		var payload map[string]string
		if data == nil {
			return nil, fmt.Errorf("request body is empty for method 'update': %w", err)
		}
		err = data.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("request body is not valid for method 'update': %w", err)
		}

		// Update the in-memory data
		// Work on a clone to avoid touching the cached object in case of errors (we'll update after we've committed the state)
		newState := state.Clone()
		newState.Update(payload)

		// Save the updated state
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		err = a.client.SetState(ctx, state, nil)

		// Update the cached state too
		*state = *newState
	}

	// Return the state
	return state, nil
}
