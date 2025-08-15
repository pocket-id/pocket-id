package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AppConfigService struct {
	dbConfig atomic.Pointer[model.AppConfig]
	db       *gorm.DB
}

func NewAppConfigService(ctx context.Context, db *gorm.DB) (*AppConfigService, error) {
	service := &AppConfigService{
		db: db,
	}

	err := service.LoadDbConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app config service: %w", err)
	}

	err = service.initInstanceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize instance ID: %w", err)
	}

	return service, nil
}

// GetDbConfig returns the application configuration.
// Important: Treat the object as read-only: do not modify its properties directly!
func (s *AppConfigService) GetDbConfig() *model.AppConfig {
	v := s.dbConfig.Load()
	if v == nil {
		// This indicates a development-time error
		panic("called GetDbConfig before DbConfig is loaded")
	}

	return v
}

func (s *AppConfigService) getDefaultDbConfig() *model.AppConfig {
	// Values are the default ones
	return &model.AppConfig{
		// General
		AppName:                model.AppConfigVariable{Value: "Pocket ID"},
		SessionDuration:        model.AppConfigVariable{Value: "60"},
		EmailsVerified:         model.AppConfigVariable{Value: "false"},
		DisableAnimations:      model.AppConfigVariable{Value: "false"},
		AllowOwnAccountEdit:    model.AppConfigVariable{Value: "true"},
		AllowUserSignups:       model.AppConfigVariable{Value: "disabled"},
		AccentColor:            model.AppConfigVariable{Value: "default"},
		SelfhostedIconsEnabled: model.AppConfigVariable{Value: "false"},
		// Internal
		BackgroundImageType: model.AppConfigVariable{Value: "jpg"},
		LogoLightImageType:  model.AppConfigVariable{Value: "svg"},
		LogoDarkImageType:   model.AppConfigVariable{Value: "svg"},
		InstanceID:          model.AppConfigVariable{Value: ""},
		// Email
		SmtpHost:                      model.AppConfigVariable{},
		SmtpPort:                      model.AppConfigVariable{},
		SmtpFrom:                      model.AppConfigVariable{},
		SmtpUser:                      model.AppConfigVariable{},
		SmtpPassword:                  model.AppConfigVariable{},
		SmtpTls:                       model.AppConfigVariable{Value: "none"},
		SmtpSkipCertVerify:            model.AppConfigVariable{Value: "false"},
		EmailLoginNotificationEnabled: model.AppConfigVariable{Value: "false"},
		EmailOneTimeAccessAsUnauthenticatedEnabled: model.AppConfigVariable{Value: "false"},
		EmailOneTimeAccessAsAdminEnabled:           model.AppConfigVariable{Value: "false"},
		EmailApiKeyExpirationEnabled:               model.AppConfigVariable{Value: "false"},
		// LDAP
		LdapEnabled:                        model.AppConfigVariable{Value: "false"},
		LdapUrl:                            model.AppConfigVariable{},
		LdapBindDn:                         model.AppConfigVariable{},
		LdapBindPassword:                   model.AppConfigVariable{},
		LdapBase:                           model.AppConfigVariable{},
		LdapUserSearchFilter:               model.AppConfigVariable{Value: "(objectClass=person)"},
		LdapUserGroupSearchFilter:          model.AppConfigVariable{Value: "(objectClass=groupOfNames)"},
		LdapSkipCertVerify:                 model.AppConfigVariable{Value: "false"},
		LdapAttributeUserUniqueIdentifier:  model.AppConfigVariable{},
		LdapAttributeUserUsername:          model.AppConfigVariable{},
		LdapAttributeUserEmail:             model.AppConfigVariable{},
		LdapAttributeUserFirstName:         model.AppConfigVariable{},
		LdapAttributeUserLastName:          model.AppConfigVariable{},
		LdapAttributeUserProfilePicture:    model.AppConfigVariable{},
		LdapAttributeGroupMember:           model.AppConfigVariable{Value: "member"},
		LdapAttributeGroupUniqueIdentifier: model.AppConfigVariable{},
		LdapAttributeGroupName:             model.AppConfigVariable{},
		LdapAttributeAdminGroup:            model.AppConfigVariable{},
		LdapSoftDeleteUsers:                model.AppConfigVariable{Value: "true"},
	}
}

func (s *AppConfigService) updateAppConfigStartTransaction(ctx context.Context) (tx *gorm.DB, err error) {
	// We start a transaction before doing any work, to ensure that we are the only ones updating the data in the database
	// This works across multiple processes too
	tx = s.db.Begin()
	err = tx.Error
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}

	// With SQLite there's nothing else we need to do, because a transaction blocks the entire database
	// However, with Postgres we need to manually lock the table to prevent others from doing the same
	switch s.db.Name() {
	case "postgres":
		// We do not use "NOWAIT" so this blocks until the database is available, or the context is canceled
		// Here we use a context with a 10s timeout in case the database is blocked for longer
		lockCtx, lockCancel := context.WithTimeout(ctx, 10*time.Second)
		defer lockCancel()
		err = tx.
			WithContext(lockCtx).
			Exec("LOCK TABLE app_config_variables IN ACCESS EXCLUSIVE MODE").
			Error
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to acquire lock on app_config_variables table: %w", err)
		}
	default:
		// Nothing to do here
	}

	return tx, nil
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

	// Start the transaction
	tx, err := s.updateAppConfigStartTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		tx.Rollback()
	}()

	// From here onwards, we know we are the only process/goroutine with exclusive access to the config
	// Re-load the config from the database to be sure we have the correct data
	cfg, err := s.loadDbConfigInternal(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to reload config from database: %w", err)
	}

	defaultCfg := s.getDefaultDbConfig()

	// Iterate through all the fields to update
	// We update the in-memory data (in the cfg struct) and collect values to update in the database
	rt := reflect.ValueOf(input).Type()
	rv := reflect.ValueOf(input)
	dbUpdate := make([]model.AppConfigVariable, 0, rt.NumField())
	for i := range rt.NumField() {
		field := rt.Field(i)
		value := rv.FieldByName(field.Name).String()

		// Get the value of the json tag, taking only what's before the comma
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		// Update the in-memory config value
		// If the new value is an empty string, then we set the in-memory value to the default one
		// Skip values that are internal only and can't be updated
		if value == "" {
			// Ignore errors here as we know the key exists
			defaultValue, _, _ := defaultCfg.FieldByKey(key)
			err = cfg.UpdateField(key, defaultValue, true)
		} else {
			err = cfg.UpdateField(key, value, true)
		}

		// If we tried to update an internal field, ignore the error (and do not update in the DB)
		if errors.Is(err, model.AppConfigInternalForbiddenError{}) {
			continue
		} else if err != nil {
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

	// Start the transaction
	tx, err := s.updateAppConfigStartTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() {
		tx.Rollback()
	}()

	// From here onwards, we know we are the only process/goroutine with exclusive access to the config
	// Re-load the config from the database to be sure we have the correct data
	cfg, err := s.loadDbConfigInternal(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to reload config from database: %w", err)
	}

	defaultCfg := s.getDefaultDbConfig()

	// Iterate through all the fields to update
	// We update the in-memory data (in the cfg struct) and collect values to update in the database
	// (Note the += 2, as we are iterating through key-value pairs)
	dbUpdate := make([]model.AppConfigVariable, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]

		// Ensure that the field is valid
		// We do this by grabbing the default value
		var defaultValue string
		defaultValue, isInternal, err := defaultCfg.FieldByKey(key)
		if err != nil {
			return fmt.Errorf("invalid configuration key '%s': %w", key, err)
		}
		if !isInternal && common.EnvConfig.UiConfigDisabled {
			return &common.UiConfigDisabledError{}
		}

		// Update the in-memory config value
		// If the new value is an empty string, then we set the in-memory value to the default one
		// Skip values that are internal only and can't be updated
		if value == "" {
			err = cfg.UpdateField(key, defaultValue, false)
		} else {
			err = cfg.UpdateField(key, value, false)
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

func (s *AppConfigService) UpdateImage(ctx context.Context, uploadedFile *multipart.FileHeader, imageName string, oldImageType string) (err error) {
	fileType := strings.ToLower(utils.GetFileExtension(uploadedFile.Filename))
	mimeType := utils.GetImageMimeType(fileType)
	if mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	// Save the updated image
	imagePath := common.EnvConfig.UploadPath + "/application-images/" + imageName + "." + fileType
	err = utils.SaveFile(uploadedFile, imagePath)
	if err != nil {
		return err
	}

	// Delete the old image if it has a different file type, then update the type in the database
	if fileType != oldImageType {
		oldImagePath := common.EnvConfig.UploadPath + "/application-images/" + imageName + "." + oldImageType
		err = os.Remove(oldImagePath)
		if err != nil {
			return err
		}

		// Update the file type in the database
		err = s.UpdateAppConfigValues(ctx, imageName+"ImageType", fileType)
		if err != nil {
			return err
		}

	}

	return nil
}

// LoadDbConfig loads the configuration values from the database into the DbConfig struct.
func (s *AppConfigService) LoadDbConfig(ctx context.Context) (err error) {
	dest, err := s.loadDbConfigInternal(ctx, s.db)
	if err != nil {
		return err
	}

	s.dbConfig.Store(dest)

	return nil
}

func (s *AppConfigService) loadDbConfigInternal(ctx context.Context, tx *gorm.DB) (*model.AppConfig, error) {
	// If the UI config is disabled, only load from the env
	if common.EnvConfig.UiConfigDisabled {
		dest, err := s.loadDbConfigFromEnv(ctx, tx)
		return dest, err
	}

	// First, start from the default configuration
	dest := s.getDefaultDbConfig()

	// Load all configuration values from the database
	// This loads all values in a single shot
	var loaded []model.AppConfigVariable
	queryCtx, queryCancel := context.WithTimeout(ctx, 10*time.Second)
	defer queryCancel()
	err := tx.
		WithContext(queryCtx).
		Find(&loaded).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from the database: %w", err)
	}

	// Iterate through all values loaded from the database
	for _, v := range loaded {
		// Find the field in the struct whose "key" tag matches, then update that
		err = dest.UpdateField(v.Key, v.Value, false)

		// We ignore the case of fields that don't exist, as there may be leftover data in the database
		if err != nil && !errors.Is(err, model.AppConfigKeyNotFoundError{}) {
			return nil, fmt.Errorf("failed to process config for key '%s': %w", v.Key, err)
		}
	}

	return dest, nil
}

func (s *AppConfigService) loadDbConfigFromEnv(ctx context.Context, tx *gorm.DB) (*model.AppConfig, error) {
	// First, start from the default configuration
	dest := s.getDefaultDbConfig()

	// Iterate through each field
	rt := reflect.ValueOf(dest).Elem().Type()
	rv := reflect.ValueOf(dest).Elem()
	for i := range rt.NumField() {
		field := rt.Field(i)

		// Get the key and internal tag values
		key, attrs, _ := strings.Cut(field.Tag.Get("key"), ",")

		// Internal fields are loaded from the database as they can't be set from the environment
		if attrs == "internal" {
			var value string
			err := tx.WithContext(ctx).
				Model(&model.AppConfigVariable{}).
				Where("key = ?", key).
				Select("value").
				First(&value).Error
			if err == nil {
				rv.Field(i).FieldByName("Value").SetString(value)
			}
			continue
		}

		envVarName := utils.CamelCaseToScreamingSnakeCase(key)

		// Set the value if it's set
		value, ok := os.LookupEnv(envVarName)
		if ok {
			rv.Field(i).FieldByName("Value").SetString(value)
			continue
		}

		// If it's sensitive, we also allow reading from file
		if attrs == "sensitive" {
			fileName := os.Getenv(envVarName + "_FILE")
			if fileName != "" {
				b, err := os.ReadFile(fileName)
				if err != nil {
					return nil, fmt.Errorf("failed to read secret '%s' from file '%s': %w", envVarName, fileName, err)
				}
				rv.Field(i).FieldByName("Value").SetString(string(b))
				continue
			}
		}
	}

	return dest, nil
}

func (s *AppConfigService) initInstanceID(ctx context.Context) error {
	// Check if the instance ID is already set
	instanceID := s.GetDbConfig().InstanceID.Value
	if instanceID != "" {
		return nil
	}

	newInstanceID, err := uuid.GenerateUUID()
	if err != nil {
		return fmt.Errorf("failed to generate new instance ID: %w", err)
	}

	err = s.UpdateAppConfigValues(ctx, "instanceId", newInstanceID)
	if err != nil {
		return fmt.Errorf("failed to update instance ID in the database: %w", err)
	}

	return nil
}
