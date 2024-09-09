package service

import (
	"fmt"
	"github.com/stonith404/pocket-id/backend/internal/common"
	"github.com/stonith404/pocket-id/backend/internal/dto"
	"github.com/stonith404/pocket-id/backend/internal/model"
	"github.com/stonith404/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"log"
	"mime/multipart"
	"os"
	"reflect"
)

type AppConfigService struct {
	DbConfig *model.AppConfig
	db       *gorm.DB
}

func NewAppConfigService(db *gorm.DB) *AppConfigService {
	service := &AppConfigService{
		DbConfig: &defaultDbConfig,
		db:       db,
	}
	if err := service.InitDbConfig(); err != nil {
		log.Fatalf("Failed to initialize app config service: %v", err)
	}
	return service
}

var defaultDbConfig = model.AppConfig{
	AppName: model.AppConfigVariable{
		Key:      "appName",
		Type:     "string",
		IsPublic: true,
		Value:    "Pocket ID",
	},
	SessionDuration: model.AppConfigVariable{
		Key:   "sessionDuration",
		Type:  "number",
		Value: "60",
	},
	BackgroundImageType: model.AppConfigVariable{
		Key:        "backgroundImageType",
		Type:       "string",
		IsInternal: true,
		Value:      "jpg",
	},
	LogoImageType: model.AppConfigVariable{
		Key:        "logoImageType",
		Type:       "string",
		IsInternal: true,
		Value:      "svg",
	},
	EmailEnabled: model.AppConfigVariable{
		Key:   "emailEnabled",
		Type:  "bool",
		Value: "false",
	},
	SmtpHost: model.AppConfigVariable{
		Key:  "smtpHost",
		Type: "string",
	},
	SmtpPort: model.AppConfigVariable{
		Key:  "smtpPort",
		Type: "number",
	},
	SmtpFrom: model.AppConfigVariable{
		Key:  "smtpFrom",
		Type: "string",
	},
	SmtpUser: model.AppConfigVariable{
		Key:  "smtpUser",
		Type: "string",
	},
	SmtpPassword: model.AppConfigVariable{
		Key:  "smtpPassword",
		Type: "string",
	},
}

func (s *AppConfigService) UpdateAppConfig(input dto.AppConfigUpdateDto) ([]model.AppConfigVariable, error) {
	var savedConfigVariables []model.AppConfigVariable

	tx := s.db.Begin()
	rt := reflect.ValueOf(input).Type()
	rv := reflect.ValueOf(input)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		key := field.Tag.Get("json")
		value := rv.FieldByName(field.Name).String()

		var appConfigVariable model.AppConfigVariable
		if err := tx.First(&appConfigVariable, "key = ? AND is_internal = false", key).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		appConfigVariable.Value = value
		if err := tx.Save(&appConfigVariable).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		savedConfigVariables = append(savedConfigVariables, appConfigVariable)
	}

	tx.Commit()

	if err := s.loadDbConfigFromDb(); err != nil {
		return nil, err
	}

	return savedConfigVariables, nil
}

func (s *AppConfigService) UpdateImageType(imageName string, fileType string) error {
	key := fmt.Sprintf("%sImageType", imageName)
	err := s.db.Model(&model.AppConfigVariable{}).Where("key = ?", key).Update("value", fileType).Error
	if err != nil {
		return err
	}

	return s.loadDbConfigFromDb()
}

func (s *AppConfigService) ListAppConfig(showAll bool) ([]model.AppConfigVariable, error) {
	var configuration []model.AppConfigVariable
	var err error

	if showAll {
		err = s.db.Find(&configuration).Error
	} else {
		err = s.db.Find(&configuration, "is_public = true").Error
	}

	if err != nil {
		return nil, err
	}

	return configuration, nil
}

func (s *AppConfigService) UpdateImage(uploadedFile *multipart.FileHeader, imageName string, oldImageType string) error {
	fileType := utils.GetFileExtension(uploadedFile.Filename)
	mimeType := utils.GetImageMimeType(fileType)
	if mimeType == "" {
		return common.ErrFileTypeNotSupported
	}

	// Delete the old image if it has a different file type
	if fileType != oldImageType {
		oldImagePath := fmt.Sprintf("%s/application-images/%s.%s", common.EnvConfig.UploadPath, imageName, oldImageType)
		if err := os.Remove(oldImagePath); err != nil {
			return err
		}
	}

	imagePath := fmt.Sprintf("%s/application-images/%s.%s", common.EnvConfig.UploadPath, imageName, fileType)
	if err := utils.SaveFile(uploadedFile, imagePath); err != nil {
		return err
	}

	// Update the file type in the database
	if err := s.UpdateImageType(imageName, fileType); err != nil {
		return err
	}

	return nil
}

// InitDbConfig creates the default configuration values in the database if they do not exist,
// updates existing configurations if they differ from the default, and deletes any configurations
// that are not in the default configuration.
func (s *AppConfigService) InitDbConfig() error {
	// Reflect to get the underlying value of DbConfig and its default configuration
	defaultConfigReflectValue := reflect.ValueOf(defaultDbConfig)
	defaultKeys := make(map[string]struct{})

	// Iterate over the fields of DbConfig
	for i := 0; i < defaultConfigReflectValue.NumField(); i++ {
		defaultConfigVar := defaultConfigReflectValue.Field(i).Interface().(model.AppConfigVariable)

		defaultKeys[defaultConfigVar.Key] = struct{}{}

		var storedConfigVar model.AppConfigVariable
		if err := s.db.First(&storedConfigVar, "key = ?", defaultConfigVar.Key).Error; err != nil {
			// If the configuration does not exist, create it
			if err := s.db.Create(&defaultConfigVar).Error; err != nil {
				return err
			}
			continue
		}

		// Update existing configuration if it differs from the default
		if storedConfigVar.Type != defaultConfigVar.Type || storedConfigVar.IsPublic != defaultConfigVar.IsPublic || storedConfigVar.IsInternal != defaultConfigVar.IsInternal {
			storedConfigVar.Type = defaultConfigVar.Type
			storedConfigVar.IsPublic = defaultConfigVar.IsPublic
			storedConfigVar.IsInternal = defaultConfigVar.IsInternal
			if err := s.db.Save(&storedConfigVar).Error; err != nil {
				return err
			}
		}
	}

	// Delete any configurations not in the default keys
	var allConfigVars []model.AppConfigVariable
	if err := s.db.Find(&allConfigVars).Error; err != nil {
		return err
	}

	for _, config := range allConfigVars {
		if _, exists := defaultKeys[config.Key]; !exists {
			if err := s.db.Delete(&config).Error; err != nil {
				return err
			}
		}
	}
	return s.loadDbConfigFromDb()
}

func (s *AppConfigService) loadDbConfigFromDb() error {
	dbConfigReflectValue := reflect.ValueOf(s.DbConfig).Elem()

	for i := 0; i < dbConfigReflectValue.NumField(); i++ {
		dbConfigField := dbConfigReflectValue.Field(i)
		currentConfigVar := dbConfigField.Interface().(model.AppConfigVariable)
		var storedConfigVar model.AppConfigVariable
		if err := s.db.First(&storedConfigVar, "key = ?", currentConfigVar.Key).Error; err != nil {
			return err
		}

		dbConfigField.Set(reflect.ValueOf(storedConfigVar))
	}

	return nil
}
