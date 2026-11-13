package appconfig

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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
	db     *gorm.DB
}

// appConfigActorBootstrap is the type for the payload of the init method
type appConfigActorBootstrap struct {
	LegacyConfig map[string]string
}

// newAppConfigActorFactory returns the actor.Factory for the AppConfig actor
func newAppConfigActorFactory(db *gorm.DB) actor.Factory {
	return func(actorID string, service *actor.Service) actor.Actor {
		return newAppConfigActor(actorID, service, db)
	}
}

// newAppConfigActor allocates a new AppConfig actor
func newAppConfigActor(actorID string, service *actor.Service, db *gorm.DB) actor.Actor {
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
		db:     db,
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

	// Upgrade existing state with defaults added by newer Pocket ID versions, and encrypt the secrets saved before they were encrypted
	save := true
	if state != nil {
		hasPlaintextSecrets := state.hasPlaintextSecrets()
		state, err = state.decryptSecrets()
		if err != nil {
			return fmt.Errorf("error decrypting actor state: %w", err)
		}
		save = state.applyDefaults() || hasPlaintextSecrets
	}

	// Check if a new state should be initialized from legacy configuration
	if state == nil && data != nil {
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

	// Initialize a new state with defaults when no legacy configuration exists
	if state == nil {
		state = getDefaultConfig()
	}

	// Persist new and upgraded state eagerly so every later activation sees the complete model
	if save {
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		err = a.setState(ctx, state)
		if err != nil {
			return err
		}
	}

	// The legacy config is not needed anymore once the state is saved, and it holds the secrets in plaintext
	// Failing to delete it does not affect the config, so the next activation just tries again
	err = deleteLegacyConfig(parentCtx, a.db)
	if err != nil {
		a.log.WarnContext(parentCtx, "Failed to delete the legacy config", slog.Any("error", err))
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
	state, err := a.getState(ctx)
	if err != nil {
		return nil, err
	}

	// Return the state
	return state, nil
}

func (a *appConfigActor) Invoke(parentCtx context.Context, method string, data actor.Envelope) (any, error) {
	// Check the method first
	switch method {
	case "get", "update":
		// All good
		// Note: we support "get" also via Invoke and not just Peek
	default:
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}

	// Load the actor state
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	state, err := a.getState(ctx)
	if err != nil {
		return nil, err
	}

	switch method {
	case "get":
		// If the method is "get", just return the actor state, we're done
		// This switch case is a no-op

	case "update":
		// Update the config
		// The input data must be a map[string]string
		if data == nil {
			return nil, errors.New("request body is empty for method 'update'")
		}
		payload := map[string]string{}
		err = data.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("request body is not valid for method 'update': %w", err)
		}

		// Update the in-memory data, which is a decrypted copy of the cached object
		err = state.Update(payload)
		if err != nil {
			return nil, fmt.Errorf("request body is not valid for method 'update': %w", err)
		}

		// Save the updated state, which also updates the cached object
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()
		err = a.setState(ctx, state)
		if err != nil {
			return nil, err
		}

		return state, nil
	}

	// Return the state
	return state, nil
}

// getState returns the actor state with its secrets decrypted
func (a *appConfigActor) getState(ctx context.Context) (*AppConfigModel, error) {
	state, err := a.client.GetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving actor state: %w", err)
	}
	if state == nil {
		return nil, nil
	}

	state, err = state.decryptSecrets()
	if err != nil {
		return nil, fmt.Errorf("error decrypting actor state: %w", err)
	}
	return state, nil
}

// setState saves the actor state with its secrets encrypted, so they never end up in the database in plaintext
func (a *appConfigActor) setState(ctx context.Context, state *AppConfigModel) error {
	encrypted, err := state.encryptSecrets()
	if err != nil {
		return fmt.Errorf("error encrypting actor state: %w", err)
	}

	err = a.client.SetState(ctx, encrypted, nil)
	if err != nil {
		return fmt.Errorf("error saving actor state: %w", err)
	}
	return nil
}
