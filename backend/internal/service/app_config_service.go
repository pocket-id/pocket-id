package service

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"reflect"
	"strings"
	"sync/atomic"

	"gorm.io/gorm"

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

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var err error

	rt := reflect.ValueOf(input).Type()
	rv := reflect.ValueOf(input)

	savedConfigVariables := make([]model.AppConfigVariable, 0, rt.NumField())
	for i := range rt.NumField() {
		field := rt.Field(i)
		key := field.Tag.Get("json")
		value := rv.FieldByName(field.Name).String()

		// If the emailEnabled is set to false, disable the emailOneTimeAccessEnabled
		if key == s.DbConfig.EmailOneTimeAccessEnabled.Key {
			if rv.FieldByName("EmailEnabled").String() == "false" {
				value = "false"
			}
		}

		var appConfigVariable model.AppConfigVariable
		err = tx.
			WithContext(ctx).
			First(&appConfigVariable, "key = ? AND is_internal = false", key).
			Error
		if err != nil {
			return nil, err
		}

		appConfigVariable.Value = value
		err = tx.
			WithContext(ctx).
			Save(&appConfigVariable).
			Error
		if err != nil {
			return nil, err
		}

		savedConfigVariables = append(savedConfigVariables, appConfigVariable)
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	err = s.LoadDbConfig(ctx)
	if err != nil {
		return nil, err
	}

	return savedConfigVariables, nil
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
	// Get the config object
	cfg := s.GetDbConfig()

	// Use reflection to iterate through all fields
	cfgValue := reflect.ValueOf(cfg).Elem()
	cfgType := cfgValue.Type()

	res := make([]model.AppConfigVariable, cfgType.NumField())

	for i := range cfgType.NumField() {
		field := cfgType.Field(i)

		key, attrs, _ := strings.Cut(field.Tag.Get("key"), ",")
		if key == "" {
			continue
		}

		// If we're only showing public variables and this is not public, skip it
		if !showAll && attrs != "public" {
			continue
		}

		fieldValue := cfgValue.Field(i)

		res[i] = model.AppConfigVariable{
			Key:   key,
			Value: fieldValue.FieldByName("Value").String(),
		}
	}

	return res
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
	// First, start from the default configuration
	dest := s.getDefaultDbConfig()
	destValue := reflect.ValueOf(dest).Elem()
	destType := destValue.Type()

	// Load all configuration values from the database
	// This loads all values in a single shot
	loaded := []model.AppConfigVariable{}
	err := s.db.Find(&loaded).Error
	if err != nil {
		return fmt.Errorf("failed to load configuration from the database: %w", err)
	}

	// Iterate through all values loaded from the database
	for _, v := range loaded {
		// If the value is empty, it means we are using the default value
		if v.Value == "" {
			continue
		}

		// Find the field in the struct whose "key" tag matches, then update that
		for i := range destType.NumField() {
			// Grab only the first part of the key, if there's a comma with additional properties
			tagValue, _, _ := strings.Cut(destType.Field(i).Tag.Get("key"), ",")
			if tagValue != v.Key {
				continue
			}

			valueField := destValue.Field(i).FieldByName("Value")
			if !valueField.CanSet() {
				return fmt.Errorf("field Value in AppConfigVariable is not settable")
			}

			// Update the value
			valueField.SetString(v.Value)
		}
	}

	// Update the value in the object
	s.dbConfig.Store(dest)

	return nil
}

func (s *AppConfigService) getConfigVariableFromEnvironmentVariable(key, fallbackValue string) string {
	environmentVariableName := utils.CamelCaseToScreamingSnakeCase(key)

	if value, exists := os.LookupEnv(environmentVariableName); exists {
		return value
	}

	return fallbackValue
}
