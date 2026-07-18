package appconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AppConfigService struct {
	actSvc    *actor.Service
	envConfig *AppConfigModel
}

func NewService(ctx context.Context, actors *local.Host, db *gorm.DB) (service *AppConfigService, err error) {
	service = &AppConfigService{}

	// If the UI config is disabled, we do not need to init the config actor
	if common.EnvConfig.UiConfigDisabled {
		service.envConfig, err = service.loadDbConfigFromEnv()
		if err != nil {
			return nil, fmt.Errorf("error loading app config from the env: %w", err)
		}

		return service, nil
	}

	// Note: we need to assign to the "err" variable in this method (for tracing), do not inline this into the "if"
	ctx, span := tracing.Start(ctx, "pocketid.appconfig.init")
	defer tracing.End(span, err)

	// Load the legacy config if any, which we need to send to the actor as bootstrap data
	legacyCfg, err := LoadLegacyConfig(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("error loading legacy config: %w", err)
	}

	// Register the AppConfig actor
	// This is a singleton actor and it's bootstrapped with the legacy config if present
	bootstrapData := &appConfigActorBootstrap{
		LegacyConfig: legacyCfg,
	}
	err = actors.RegisterSingletonActor(
		AppConfigActorType, NewAppConfigActor,
		local.WithBootstrapData(bootstrapData),
		local.WithIdleTimeout(-1), // Disable idle timeout for this actor
	)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", AppConfigActorType, err)
	}

	service.actSvc = actors.Service()

	return service, nil
}

// GetConfig returns the application configuration
// Important: Treat the object as read-only: do not modify its properties directly!
func (s *AppConfigService) GetConfig(parentCtx context.Context) (*AppConfigModel, error) {
	// If the UI config is disabled, only load from the env
	if common.EnvConfig.UiConfigDisabled {
		return s.envConfig, nil
	}

	// Retrieve the config from the actor
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	res, err := s.actSvc.Peek(ctx, AppConfigActorType, actor.SingletonActorID, "get", nil)
	if err != nil {
		return nil, fmt.Errorf("error retrieving config from actor: %w", err)
	}
	if res == nil {
		return nil, errors.New("config actor response was empty")
	}

	var cfg AppConfigModel
	err = res.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config actor response: %w", err)
	}

	return &cfg, nil
}

// UpdateAppConfig replaces the entire application configuration with the values from the input DTO.
func (s *AppConfigService) UpdateAppConfig(ctx context.Context, input dto.AppConfigUpdateDto) ([]AppConfigVariable, error) {
	// If the UI config is disabled, we cannot continue
	if common.EnvConfig.UiConfigDisabled {
		return nil, &common.UiConfigDisabledError{}
	}

	// Replace the entire config by invoking the actor
	cfg, err := s.invokeConfigActor(ctx, "replace", input)
	if err != nil {
		return nil, err
	}

	// Return the updated config
	return cfg.ToAppConfigVariableSlice(true, false), nil
}

// UpdateAppConfigValues updates the provided application configuration values.
// Keys correspond to the "json" tags on the config model.
// An empty string value resets the property to its default value.
func (s *AppConfigService) UpdateAppConfigValues(ctx context.Context, keysAndValues ...string) error {
	// Count of keysAndValues must be even
	if len(keysAndValues)%2 != 0 {
		return errors.New("invalid number of arguments received")
	}

	// If the UI config is disabled, we cannot continue
	if common.EnvConfig.UiConfigDisabled {
		return &common.UiConfigDisabledError{}
	}

	// Collect the key-value pairs into a map for the actor
	// (Note the += 2, as we are iterating through key-value pairs)
	values := make(map[string]string, len(keysAndValues)/2)
	for i := 1; i < len(keysAndValues); i += 2 {
		values[keysAndValues[i-1]] = keysAndValues[i]
	}

	// Update the config by invoking the actor
	_, err := s.invokeConfigActor(ctx, "update", values)
	return err
}

// ListAppConfig returns the application configuration as a slice of key/value pairs.
// If showAll is false, only properties marked as public are included.
func (s *AppConfigService) ListAppConfig(ctx context.Context, showAll bool) ([]AppConfigVariable, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	return cfg.ToAppConfigVariableSlice(showAll, true), nil
}

// invokeConfigActor invokes a method on the AppConfig actor and decodes the returned state.
func (s *AppConfigService) invokeConfigActor(parentCtx context.Context, method string, data any) (*AppConfigModel, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	res, err := s.actSvc.Invoke(ctx, AppConfigActorType, actor.SingletonActorID, method, data)
	if err != nil {
		return nil, fmt.Errorf("error invoking config actor method '%s': %w", method, err)
	}
	if res == nil {
		return nil, errors.New("config actor response was empty")
	}

	var cfg AppConfigModel
	err = res.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config actor response: %w", err)
	}

	return &cfg, nil
}

func (s *AppConfigService) loadDbConfigFromEnv() (*AppConfigModel, error) {
	// First, start from the default configuration
	dest := getDefaultConfig()

	// Iterate through each field
	rt := reflect.ValueOf(dest).Elem().Type()
	rv := reflect.ValueOf(dest).Elem()
	for i := range rt.NumField() {
		field := rt.Field(i)

		// Derive the environment variable name from the configuration's JSON key
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		envVarName := utils.CamelCaseToScreamingSnakeCase(key)

		// Set the value if it's set
		value, ok := os.LookupEnv(envVarName)
		if ok {
			rv.Field(i).SetString(value)
			continue
		}

		// If it's sensitive, we also allow reading from file
		if field.Tag.Get("sensitive") == "true" {
			fileName := os.Getenv(envVarName + "_FILE")
			if fileName != "" {
				// #nosec G703 - Value is provided by admin
				b, err := os.ReadFile(fileName)
				if err != nil {
					return nil, fmt.Errorf("failed to read secret '%s' from file '%s': %w", envVarName, fileName, err)
				}

				rv.Field(i).SetString(string(b))
				continue
			}
		}
	}

	return dest, nil
}
