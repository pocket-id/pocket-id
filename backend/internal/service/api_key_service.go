package service

import (
	"errors"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"log"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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

func (s *ApiKeyService) ListApiKeys(userID string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.ApiKey, utils.PaginationResponse, error) {
	query := s.db.Where("user_id = ?", userID).Model(&model.ApiKey{})

	var apiKeys []model.ApiKey
	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &apiKeys)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return apiKeys, pagination, nil
}

func (s *ApiKeyService) CreateApiKey(userID string, input dto.ApiKeyCreateDto) (model.ApiKey, string, error) {
	// Check if expiration is in the future
	if !input.ExpiresAt.ToTime().After(time.Now()) {
		return model.ApiKey{}, "", &common.APIKeyExpirationDateError{}
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return model.ApiKey{}, "", err
	}

	apiKey := model.ApiKey{
		Name:        input.Name,
		Key:         utils.CreateSha256Hash(token), // Hash the token for storage
		Description: &input.Description,
		ExpiresAt:   datatype.DateTime(input.ExpiresAt),
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
			return &common.APIKeyNotFoundError{}
		}
		return err
	}

	return s.db.Delete(&apiKey).Error
}

func (s *ApiKeyService) ValidateApiKey(apiKey string) (model.User, error) {
	if apiKey == "" {
		return model.User{}, &common.NoAPIKeyProvidedError{}
	}

	var key model.ApiKey
	hashedKey := utils.CreateSha256Hash(apiKey)

	if err := s.db.Preload("User").Where("key = ? AND expires_at > ?",
		hashedKey, datatype.DateTime(time.Now())).Preload("User").First(&key).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, &common.InvalidAPIKeyError{}
		}

		return model.User{}, err
	}

	// Update last used time
	now := datatype.DateTime(time.Now())
	key.LastUsedAt = &now
	if err := s.db.Save(&key).Error; err != nil {
		log.Printf("Failed to update last used time: %v", err)
	}

	return key.User, nil
}
