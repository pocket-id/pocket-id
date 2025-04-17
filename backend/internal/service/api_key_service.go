package service

import (
	"context"
	"errors"
	"log"
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApiKeyService struct {
	db           *gorm.DB
	emailService *EmailService
}

func NewApiKeyService(db *gorm.DB, emailService *EmailService) *ApiKeyService {
	return &ApiKeyService{db: db, emailService: emailService}
}

func (s *ApiKeyService) ListApiKeys(ctx context.Context, userID string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.ApiKey, utils.PaginationResponse, error) {
	query := s.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Model(&model.ApiKey{})

	var apiKeys []model.ApiKey
	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &apiKeys)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return apiKeys, pagination, nil
}

func (s *ApiKeyService) CreateApiKey(ctx context.Context, userID string, input dto.ApiKeyCreateDto) (model.ApiKey, string, error) {
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

	err = s.db.
		WithContext(ctx).
		Create(&apiKey).
		Error
	if err != nil {
		return model.ApiKey{}, "", err
	}

	// Return the raw token only once - it cannot be retrieved later
	return apiKey, token, nil
}

func (s *ApiKeyService) RevokeApiKey(ctx context.Context, userID, apiKeyID string) error {
	var apiKey model.ApiKey
	err := s.db.
		WithContext(ctx).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		Delete(&apiKey).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &common.APIKeyNotFoundError{}
		}
		return err
	}

	return nil
}

func (s *ApiKeyService) ValidateApiKey(ctx context.Context, apiKey string) (model.User, error) {
	if apiKey == "" {
		return model.User{}, &common.NoAPIKeyProvidedError{}
	}

	now := time.Now()
	hashedKey := utils.CreateSha256Hash(apiKey)

	var key model.ApiKey
	err := s.db.
		WithContext(ctx).
		Model(&model.ApiKey{}).
		Clauses(clause.Returning{}).
		Where("key = ? AND expires_at > ?", hashedKey, datatype.DateTime(now)).
		Updates(&model.ApiKey{
			LastUsedAt: utils.Ptr(datatype.DateTime(now)),
		}).
		Preload("User").
		First(&key).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, &common.InvalidAPIKeyError{}
		}

		return model.User{}, err
	}

	return key.User, nil
}

func (s *ApiKeyService) ListExpiringApiKeys(ctx context.Context, daysAhead int) ([]model.ApiKey, error) {
	var keys []model.ApiKey
	now := time.Now()
	cutoff := now.AddDate(0, 0, daysAhead)

	// Convert to Unix timestamps
	nowUnix := now.Unix()
	cutoffUnix := cutoff.Unix()

	log.Printf("Looking for keys expiring between %v (%d) and %v (%d)",
		now, nowUnix, cutoff, cutoffUnix)

	// Query using Unix timestamps
	err := s.db.
		WithContext(ctx).
		Preload("User").
		Where("expires_at > ? AND expires_at <= ?", nowUnix, cutoffUnix).
		Find(&keys).
		Error

	// Debug found keys
	for _, key := range keys {
		log.Printf("Found expiring key: %s (expires: %v / %d)",
			key.Name, key.ExpiresAt.ToTime(), key.ExpiresAt)
	}

	return keys, err
}

func (s *ApiKeyService) SendApiKeyExpiringSoonEmail(ctx context.Context, apiKey model.ApiKey) error {
	user := apiKey.User
	if user.ID == "" {
		if err := s.db.WithContext(ctx).First(&user, "id = ?", apiKey.UserID).Error; err != nil {
			return err
		}
	}
	return SendEmail(ctx, s.emailService, email.Address{
		Name:  user.FullName(),
		Email: user.Email,
	}, ApiKeyExpiringSoonTemplate, &ApiKeyExpiringSoonTemplateData{
		ApiKeyName: apiKey.Name,
		ExpiresAt:  apiKey.ExpiresAt.ToTime(),
	})
}
