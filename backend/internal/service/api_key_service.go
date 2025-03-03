package service

import (
	"errors"
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
	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return model.ApiKey{}, "", err
	}

	apiKey := model.ApiKey{
		Name:        input.Name,
		Key:         utils.HashApiKey(token), // Hash the token for storage
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
	var key model.ApiKey
	hashedKey := utils.HashApiKey(apiKey)

	if err := s.db.Where("key = ? AND enabled = ? AND expires_at > ?",
		hashedKey, true, time.Now()).Preload("User").First(&key).Error; err != nil {
		return model.User{}, errors.New("invalid API key")
	}

	// Update last used time
	now := time.Now()
	key.LastUsedAt = &now
	s.db.Save(&key)

	return key.User, nil
}
