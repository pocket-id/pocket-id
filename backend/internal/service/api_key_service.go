package service

import (
	"errors"
	"log"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type ApiKeyService struct {
	db *gorm.DB
}

func NewApiKeyService(db *gorm.DB) *ApiKeyService {
	return &ApiKeyService{db: db}
}

func (s *ApiKeyService) ListApiKeys(userID string) ([]model.ApiKey, error) {
	var apiKeys []model.ApiKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (s *ApiKeyService) CreateApiKey(userID string, input dto.ApiKeyCreateDto) (model.ApiKey, string, error) {
	// Check if expiration is in the future
	if !input.ExpiresAt.After(time.Now()) {
		return model.ApiKey{}, "", errors.New("expiration time must be in the future")
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return model.ApiKey{}, "", err
	}

	apiKey := model.ApiKey{
		Name:        input.Name,
		Key:         utils.CreateSha256Hash(token), // Hash the token for storage
		Description: input.Description,
		Enabled:     true,
		ExpiresAt:   input.ExpiresAt,
		UserID:      userID,
	}

	if err := s.db.Create(&apiKey).Error; err != nil {
		return model.ApiKey{}, "", err
	}

	// Return the raw token only once - it cannot be retrieved later
	return apiKey, token, nil
}

func (s *ApiKeyService) RevokeApiKey(userID, apiKeyID string) error {
	var apiKey model.ApiKey
	if err := s.db.Where("id = ? AND user_id = ?", apiKeyID, userID).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("API key not found")
		}
		return err
	}

	return s.db.Delete(&apiKey).Error
}

func (s *ApiKeyService) ValidateApiKey(apiKey string) (model.User, error) {
	if apiKey == "" {
		return model.User{}, errors.New("no API key provided")
	}

	var key model.ApiKey
	hashedKey := utils.CreateSha256Hash(apiKey)

	if err := s.db.Preload("User").Where("key = ? AND enabled = ? AND expires_at > ?",
		hashedKey, true, time.Now()).Preload("User").First(&key).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, errors.New("invalid API key")
		}

		log.Printf("Database error when validating API key: %v", err)
		return model.User{}, err
	}

	// Update last used time
	now := time.Now()
	key.LastUsedAt = &now
	if err := s.db.Save(&key).Error; err != nil {
		log.Printf("Failed to update last used time: %v", err)
	}

	return key.User, nil
}
