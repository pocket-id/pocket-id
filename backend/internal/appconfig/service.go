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
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
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

// DELETE
func (s *AppConfigService) GetDbConfig() *model.AppConfig {
	return nil
}

func (s *AppConfigService) updateAppConfigUpdateDatabase(ctx context.Context, tx *gorm.DB, dbUpdate *[]model.AppConfigVariable) error {
	err := tx.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			// Perform an "upsert" if the key already exists, replacing the value
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		Create(&dbUpdate).
		Error
	if err != nil {
		return fmt.Errorf("failed to update config in database: %w", err)
	}

	return nil
}

func (s *AppConfigService) UpdateAppConfig(ctx context.Context, input dto.AppConfigUpdateDto) ([]model.AppConfigVariable, error) {
	if common.EnvConfig.UiConfigDisabled {
		return nil, &common.UiConfigDisabledError{}
	}

	// From here onwards, we know we are the only process/goroutine with exclusive access to the config
	// Re-load the config from the database to be sure we have the correct data
	cfg, err := s.loadDbConfigInternal(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to reload config from database: %w", err)
	}

	defaultCfg := getDefaultConfig()

	// Iterate through all the fields to update
	// We update the in-memory data (in the cfg struct) and collect values to update in the database
	rt := reflect.ValueOf(input).Type()
	rv := reflect.ValueOf(input)
	dbUpdate := make([]model.AppConfigVariable, 0, rt.NumField())
	for field := range rt.Fields() {
		value := rv.FieldByName(field.Name).String()

		// Get the value of the json tag, taking only what's before the comma
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		// Update the in-memory config value
		// If the new value is an empty string, then we set the in-memory value to the default one
		if value == "" {
			// Ignore errors here as we know the key exists
			defaultValue, _ := defaultCfg.FieldByKey(key)
			err = cfg.UpdateField(key, defaultValue)
		} else {
			err = cfg.UpdateField(key, value)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to update in-memory config for key '%s': %w", key, err)
		}

		// We always save "value" which can be an empty string
		dbUpdate = append(dbUpdate, model.AppConfigVariable{
			Key:   key,
			Value: value,
		})
	}

	// Update the values in the database
	err = s.updateAppConfigUpdateDatabase(ctx, tx, &dbUpdate)
	if err != nil {
		return nil, err
	}

	// Commit the changes to the DB, then finally save the updated config in the object
	err = tx.Commit().Error
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.dbConfig.Store(cfg)

	// Return the updated config
	res := cfg.ToAppConfigVariableSlice(true, false)
	return res, nil
}

// UpdateAppConfigValues updates the application configuration values in the database.
func (s *AppConfigService) UpdateAppConfigValues(ctx context.Context, keysAndValues ...string) error {
	// Count of keysAndValues must be even
	if len(keysAndValues)%2 != 0 {
		return errors.New("invalid number of arguments received")
	}

	if common.EnvConfig.UiConfigDisabled {
		return &common.UiConfigDisabledError{}
	}

	// Start the transaction
	tx, err := s.updateAppConfigStartTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// From here onwards, we know we are the only process/goroutine with exclusive access to the config
	// Re-load the config from the database to be sure we have the correct data
	cfg, err := s.loadDbConfigInternal(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to reload config from database: %w", err)
	}

	defaultCfg := getDefaultDbConfig()

	// Iterate through all the fields to update
	// We update the in-memory data (in the cfg struct) and collect values to update in the database
	// (Note the += 2, as we are iterating through key-value pairs)
	dbUpdate := make([]model.AppConfigVariable, 0, len(keysAndValues)/2)
	for i := 1; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i-1]
		value := keysAndValues[i]

		// Ensure that the field is valid
		// We do this by grabbing the default value
		var defaultValue string
		defaultValue, err := defaultCfg.FieldByKey(key)
		if err != nil {
			return fmt.Errorf("invalid configuration key '%s': %w", key, err)
		}

		// Update the in-memory config value
		// If the new value is an empty string, then we set the in-memory value to the default one
		if value == "" {
			err = cfg.UpdateField(key, defaultValue)
		} else {
			err = cfg.UpdateField(key, value)
		}
		if err != nil {
			return fmt.Errorf("failed to update in-memory config for key '%s': %w", key, err)
		}

		// We always save "value" which can be an empty string
		dbUpdate = append(dbUpdate, model.AppConfigVariable{
			Key:   key,
			Value: value,
		})
	}

	// Update the values in the database
	err = s.updateAppConfigUpdateDatabase(ctx, tx, &dbUpdate)
	if err != nil {
		return err
	}

	// Commit the changes to the DB, then finally save the updated config in the object
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.dbConfig.Store(cfg)

	return nil
}

func (s *AppConfigService) ListAppConfig(showAll bool) []model.AppConfigVariable {
	return s.GetDbConfig().ToAppConfigVariableSlice(showAll, true)
}

func (s *AppConfigService) loadDbConfigFromEnv() (*AppConfigModel, error) {
	// First, start from the default configuration
	dest := getDefaultConfig()

	// Iterate through each field
	rt := reflect.ValueOf(dest).Elem().Type()
	rv := reflect.ValueOf(dest).Elem()
	for i := range rt.NumField() {
		field := rt.Field(i)

		// Get the key and internal tag values
		key, attrs, _ := strings.Cut(field.Tag.Get("key"), ",")
		envVarName := utils.CamelCaseToScreamingSnakeCase(key)

		// Set the value if it's set
		value, ok := os.LookupEnv(envVarName)
		if ok {
			rv.Field(i).Set(reflect.ValueOf(value))
			continue
		}

		// If it's sensitive, we also allow reading from file
		if attrs == "sensitive" {
			fileName := os.Getenv(envVarName + "_FILE")
			if fileName != "" {
				// #nosec G703 - Value is provided by admin
				b, err := os.ReadFile(fileName)
				if err != nil {
					return nil, fmt.Errorf("failed to read secret '%s' from file '%s': %w", envVarName, fileName, err)
				}

				rv.Field(i).Set(reflect.ValueOf(string(b)))
				continue
			}
		}
	}

	return dest, nil
}
