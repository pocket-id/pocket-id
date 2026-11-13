package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/italypaleale/francis/actor"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AppConfigService struct {
	actSvc    *actor.Service
	envConfig *AppConfigModel
	// Properties set in the environment, keyed by their JSON name, which take precedence over the ones saved in the UI
	envOverrides map[string]string
}

func NewService(ctx context.Context, actors francishost.Host, db *gorm.DB) (service *AppConfigService, err error) {
	service = &AppConfigService{}

	// Load and validate the properties set in the environment
	service.envOverrides, err = loadEnvOverrides()
	if err != nil {
		return nil, fmt.Errorf("error loading app config from the env: %w", err)
	}
	envConfig, err := newEnvConfig(service.envOverrides)
	if err != nil {
		return nil, fmt.Errorf("error loading app config from the env: %w", err)
	}

	// If the UI config is disabled, the environment holds the whole config and we do not need to init the config actor
	if common.EnvConfig.UiConfigDisabled {
		service.envConfig = envConfig
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
		AppConfigActorType, newAppConfigActorFactory(db),
		francishost.WithBootstrapData(bootstrapData),
		francishost.WithIdleTimeout(-1), // Disable idle timeout for this actor
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

	cfg, err := s.getSavedConfig(parentCtx)
	if err != nil {
		return nil, err
	}

	// Properties set in the environment take precedence over the ones saved in the UI
	err = cfg.Update(s.envOverrides)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// getSavedConfig returns the configuration saved in the UI, without the properties set in the environment
func (s *AppConfigService) getSavedConfig(parentCtx context.Context) (*AppConfigModel, error) {
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

// GetCIMDURLAllowlist returns the configured CIMD metadata-document URL
// allowlist. Returns an empty slice if unset or malformed (which denies all).
func (s *AppConfigService) GetCIMDURLAllowlist() []string {
	cfg, err := s.GetConfig(context.Background())
	if err != nil {
		return nil
	}
	raw := string(cfg.CIMDURLAllowlist)
	if raw == "" {
		return nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err != nil {
		return nil
	}
	return patterns
}

// UpdateAppConfig applies an update to the saved configuration and returns the configuration in effect
// Only the properties whose value changes are sent to the actor, which applies them atomically, so concurrent updates of different properties never undo each other
func (s *AppConfigService) UpdateAppConfig(ctx context.Context, update dto.Update[dto.AppConfigUpdateDto]) ([]AppConfigVariable, error) {
	// If the UI config is disabled, we cannot continue
	if common.EnvConfig.UiConfigDisabled {
		return nil, apperror.UIConfigDisabled()
	}

	// Compute the update from the saved configuration, which holds the secrets but not the values set in the environment
	saved, err := s.getSavedConfig(ctx)
	if err != nil {
		return nil, err
	}
	input, err := update(saved)
	if err != nil {
		return nil, err
	}

	// Validate the CIMD URL allowlist patterns, if provided
	if input.CIMDURLAllowlist != "" {
		var patterns []string
		if err := json.Unmarshal([]byte(input.CIMDURLAllowlist), &patterns); err != nil {
			return nil, apperror.InvalidCIMDURLPattern(input.CIMDURLAllowlist)
		}
		for _, p := range patterns {
			if err := utils.ValidateCallbackURLPattern(p); err != nil {
				return nil, apperror.InvalidCIMDURLPattern(p)
			}
		}
	}

	// Collect the properties that change, leaving out the ones set in the environment and secrets sent back redacted, so neither ends up in the database
	changes := make(map[string]string)
	inputValue := reflect.ValueOf(input)
	savedValue := reflect.ValueOf(saved).Elem()
	for i := range inputValue.NumField() {
		field := inputValue.Type().Field(i)
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		value := inputValue.Field(i).String()
		modelField, _ := savedValue.Type().FieldByName(field.Name)
		_, overridden := s.envOverrides[key]
		redacted := value == redactedValue && modelField.Tag.Get("sensitive") == "true"
		if overridden || redacted || value == savedValue.FieldByName(field.Name).String() {
			continue
		}
		changes[key] = value
	}

	// Apply the changes by invoking the actor
	cfg, err := s.invokeConfigActor(ctx, "update", changes)
	if err != nil {
		return nil, err
	}

	// Return the updated config as it is in effect
	err = cfg.Update(s.envOverrides)
	if err != nil {
		return nil, err
	}
	return cfg.ToAppConfigVariableSlice(true, s.envOverrides), nil
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
		return apperror.UIConfigDisabled()
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

	return cfg.ToAppConfigVariableSlice(showAll, s.envOverrides), nil
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

// loadEnvOverrides returns the app configuration properties set in the environment, keyed by their JSON name
// An empty variable counts as unset, like an empty value saved in the UI resets the property to its default
func loadEnvOverrides() (map[string]string, error) {
	overrides := map[string]string{}

	// Iterate through each field
	rt := reflect.TypeFor[AppConfigModel]()
	for i := range rt.NumField() {
		field := rt.Field(i)
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		envVarName := field.Tag.Get("env")
		if envVarName == "" {
			return nil, fmt.Errorf("app configuration field %s is missing its environment variable name", field.Name)
		}

		// Sensitive values can also be loaded from a file, using the variable with the "_FILE" suffix (files have precedence over the env var)
		if field.Tag.Get("sensitive") == "true" {
			value, ok, err := common.LoadStringEnvVarFromFile(envVarName)
			if err != nil {
				return nil, err
			}
			if ok && value != "" {
				overrides[key] = value
				continue
			}
		}

		if value := os.Getenv(envVarName); value != "" {
			overrides[key] = value
		}
	}

	// TODO: Remove in next major version (v3)
	// Preserve the old environment setting unless an explicit notification mode replaces it
	if _, ok := overrides["emailLoginNotificationMode"]; !ok {
		if enabled := os.Getenv("EMAIL_LOGIN_NOTIFICATION_ENABLED"); enabled != "" {
			if enabled != "true" && enabled != "false" {
				return nil, errors.New("EMAIL_LOGIN_NOTIFICATION_ENABLED must be true or false")
			}
			overrides["emailLoginNotificationMode"] = string(legacyLoginNotificationMode(AppConfigValue(enabled)))
		}
	}

	// TODO: Remove in next major version (v3)
	// Preserve the old environment setting unless an explicit email verification mode replaces it
	if _, ok := overrides["emailVerification"]; !ok {
		if enabled := os.Getenv("EMAIL_VERIFICATION_ENABLED"); enabled != "" {
			if enabled != "true" && enabled != "false" {
				return nil, errors.New("EMAIL_VERIFICATION_ENABLED must be true or false")
			}
			overrides["emailVerification"] = string(legacyEmailVerification(AppConfigValue(enabled)))
		}
	}

	return overrides, nil
}

// newEnvConfig returns the default configuration with the environment overrides applied, after validating them
func newEnvConfig(overrides map[string]string) (*AppConfigModel, error) {
	config := getDefaultConfig()
	err := config.Update(overrides)
	if err != nil {
		return nil, err
	}

	// Validate the resolved configuration before exposing values to the rest of the application
	err = validateEnvConfig(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// validateEnvConfig applies the HTTP configuration rules and reports failures using environment variable names
func validateEnvConfig(config *AppConfigModel) error {
	// Map the resolved model to the canonical update DTO so both configuration paths share validation rules
	var input dto.AppConfigUpdateDto
	err := dto.MapStruct(config, &input)
	if err != nil {
		return fmt.Errorf("failed to prepare environment app configuration for validation: %w", err)
	}

	// Collect every invalid environment variable
	err = input.Validate()
	if err != nil {
		validationErrors, ok := errors.AsType[validator.ValidationErrors](err)
		if !ok {
			return fmt.Errorf("failed to validate environment app configuration: %w", err)
		}

		failures := make([]error, 0, len(validationErrors))
		for _, validationError := range validationErrors {
			// With the UI config enabled, the other properties of a cross-property rule may be saved in the UI, so saving the config checks it instead
			if !common.EnvConfig.UiConfigDisabled && validationError.Tag() == "email_verification" {
				continue
			}

			envName, ok := appConfigEnvName(validationError.Field())
			if !ok {
				return fmt.Errorf("failed to find the environment variable for app configuration field %s", validationError.Field())
			}
			_, message := dto.ValidationErrorDetails(validationError)
			failures = append(failures, fmt.Errorf("%s %s", envName, message))
		}
		if len(failures) == 0 {
			return nil
		}

		return fmt.Errorf("invalid environment app configuration: %w", errors.Join(failures...))
	}

	return nil
}
