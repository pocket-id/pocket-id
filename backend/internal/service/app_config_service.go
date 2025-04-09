package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

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

func NewAppConfigService(ctx context.Context, db *gorm.DB) *AppConfigService {
	service := &AppConfigService{
		dbConfig: atomic.Pointer[model.AppConfig]{},
		db:       db,
	}

	err := service.LoadDbConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app config service: %v", err)
	}

	return service
}

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
		AppName:             model.AppConfigVariable{Value: "Pocket ID"},
		SessionDuration:     model.AppConfigVariable{Value: "60"},
		EmailsVerified:      model.AppConfigVariable{Value: "false"},
		AllowOwnAccountEdit: model.AppConfigVariable{Value: "true"},
		// Internal
		BackgroundImageType: model.AppConfigVariable{Value: "jpg"},
		LogoLightImageType:  model.AppConfigVariable{Value: "svg"},
		LogoDarkImageType:   model.AppConfigVariable{Value: "svg"},
		// Email
		SmtpHost:                      model.AppConfigVariable{},
		SmtpPort:                      model.AppConfigVariable{},
		SmtpFrom:                      model.AppConfigVariable{},
		SmtpUser:                      model.AppConfigVariable{},
		SmtpPassword:                  model.AppConfigVariable{},
		SmtpTls:                       model.AppConfigVariable{Value: "none"},
		SmtpSkipCertVerify:            model.AppConfigVariable{Value: "false"},
		EmailLoginNotificationEnabled: model.AppConfigVariable{Value: "false"},
		EmailOneTimeAccessEnabled:     model.AppConfigVariable{Value: "false"},
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
	}
}

func (s *AppConfigService) UpdateAppConfig(ctx context.Context, input dto.AppConfigUpdateDto) ([]model.AppConfigVariable, error) {
	if common.EnvConfig.UiConfigDisabled {
		return nil, &common.UiConfigDisabledError{}
	}

	// If EmailLoginNotificationEnabled is set to false (explicitly), disable the EmailOneTimeAccessEnabled
	if input.EmailLoginNotificationEnabled == "false" {
		input.EmailOneTimeAccessEnabled = "false"
	}

	// We start a transaction before doing any work, to ensure that we are the only ones updating the data in the database
	// This works across multiple processes too
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()
	err := tx.Error
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
			Exec("LOCK TABLE app_config_variable IN ACCESS EXCLUSIVE MODE").
			Error
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock on app_config_variable table: %w", err)
		}
	default:
		// Nothing to do here
	}

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
			defaultValue, _ := defaultCfg.FieldByKey(key)
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
	err = tx.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		Create(&dbUpdate).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to update config in database: %w", err)
	}

	// Commit the changes to the DB, then finally save the updated config in the object
	err = tx.Commit().Error
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.dbConfig.Store(cfg)

	// Return the updated config
	res := cfg.ToAppConfigVariableSlice(true)
	return res, nil
}

func (s *AppConfigService) updateImageType(ctx context.Context, imageName string, fileType string) error {
	key := imageName + "ImageType"
	err := s.db.
		WithContext(ctx).
		Model(&model.AppConfigVariable{}).
		Where("key = ?", key).
		Update("value", fileType).
		Error
	if err != nil {
		return err
	}

	return s.LoadDbConfig(ctx)
}

func (s *AppConfigService) ListAppConfig(showAll bool) []model.AppConfigVariable {
	return s.GetDbConfig().ToAppConfigVariableSlice(showAll)
}

func (s *AppConfigService) UpdateImage(ctx context.Context, uploadedFile *multipart.FileHeader, imageName string, oldImageType string) (err error) {
	fileType := utils.GetFileExtension(uploadedFile.Filename)
	mimeType := utils.GetImageMimeType(fileType)
	if mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	// Delete the old image if it has a different file type
	if fileType != oldImageType {
		oldImagePath := common.EnvConfig.UploadPath + "/application-images/" + imageName + "." + oldImageType
		err = os.Remove(oldImagePath)
		if err != nil {
			return err
		}
	}

	imagePath := common.EnvConfig.UploadPath + "/application-images/" + imageName + "." + fileType
	err = utils.SaveFile(uploadedFile, imagePath)
	if err != nil {
		return err
	}

	// Update the file type in the database
	err = s.updateImageType(ctx, imageName, fileType)
	if err != nil {
		return err
	}

	return nil
}

// LoadDbConfig loads the configuration values from the database into the DbConfig struct.
func (s *AppConfigService) LoadDbConfig(ctx context.Context) error {
	dest, err := s.loadDbConfigInternal(ctx, s.db)
	if err != nil {
		return err
	}

	// Update the value in the object
	s.dbConfig.Store(dest)

	return nil
}

func (s *AppConfigService) loadDbConfigInternal(ctx context.Context, tx *gorm.DB) (*model.AppConfig, error) {
	// First, start from the default configuration
	dest := s.getDefaultDbConfig()

	// Load all configuration values from the database
	// This loads all values in a single shot
	loaded := []model.AppConfigVariable{}
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
		// If the value is empty, it means we are using the default value
		if v.Value == "" {
			continue
		}

		// Find the field in the struct whose "key" tag matches, then update that
		err = dest.UpdateField(v.Key, v.Value, false)

		// We ignore the case of fields that don't exist, as there may be leftover data in the database
		if err != nil && !errors.Is(err, model.AppConfigKeyNotFoundError{}) {
			return nil, fmt.Errorf("failed to process config for key '%s': %w", v.Key, err)
		}
	}

	return dest, nil
}

func (s *AppConfigService) getConfigVariableFromEnvironmentVariable(key, fallbackValue string) string {
	environmentVariableName := utils.CamelCaseToScreamingSnakeCase(key)

	if value, exists := os.LookupEnv(environmentVariableName); exists {
		return value
	}

	return fallbackValue
}
