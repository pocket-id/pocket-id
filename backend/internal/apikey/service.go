package apikey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Service holds the business logic for managing user API keys
type Service struct {
	db           *gorm.DB
	staticApiKey string
}

func newService(ctx context.Context, db *gorm.DB, staticApiKey string) (*Service, error) {
	s := &Service{
		db:           db,
		staticApiKey: staticApiKey,
	}

	if staticApiKey == "" {
		err := s.deleteStaticApiKeyUser(ctx)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Service) ListApiKeys(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]ApiKey, utils.PaginationResponse, error) {
	query := s.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Model(&ApiKey{})

	var apiKeys []ApiKey
	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &apiKeys)
	if err != nil {
		return nil, utils.PaginationResponse{}, fmt.Errorf("error listing API keys: %w", err)
	}

	return apiKeys, pagination, nil
}

func (s *Service) CreateApiKey(ctx context.Context, userID string, input apiKeyCreateDto) (ApiKey, string, error) {
	// Check if expiration is in the future
	if !input.ExpiresAt.ToTime().After(time.Now()) {
		return ApiKey{}, "", apperror.InvalidAPIKeyExpiration()
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error generating API key token: %w", err)
	}

	apiKey := ApiKey{
		Name:        input.Name,
		Key:         utils.CreateSha256Hash(token), // Hash the token for storage
		Description: input.Description,
		ExpiresAt:   input.ExpiresAt,
		UserID:      userID,
	}

	err = s.db.
		WithContext(ctx).
		Create(&apiKey).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ApiKey{}, "", apperror.AlreadyInUse("API key name")
		}
		return ApiKey{}, "", fmt.Errorf("error creating API key: %w", err)
	}

	// Return the raw token only once - it cannot be retrieved later
	return apiKey, token, nil
}

func (s *Service) RenewApiKey(ctx context.Context, userID, apiKeyID string, expiration time.Time) (ApiKey, string, error) {
	// Check if expiration is in the future
	if !expiration.After(time.Now()) {
		return ApiKey{}, "", apperror.InvalidAPIKeyExpiration()
	}

	tx := s.db.Begin()
	defer tx.Rollback()

	var apiKey ApiKey
	err := tx.
		WithContext(ctx).
		Model(&ApiKey{}).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		First(&apiKey).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ApiKey{}, "", apperror.APIKeyNotFound()
	} else if err != nil {
		return ApiKey{}, "", fmt.Errorf("error loading API key: %w", err)
	}

	// Only allow renewal if the key has already expired
	if apiKey.ExpiresAt.ToTime().After(time.Now()) {
		return ApiKey{}, "", apperror.APIKeyNotExpired()
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error generating API key token: %w", err)
	}

	apiKey.Key = utils.CreateSha256Hash(token)
	apiKey.ExpiresAt = datatype.DateTime(expiration)

	err = tx.WithContext(ctx).Save(&apiKey).Error
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error saving API key: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error committing transaction: %w", err)
	}

	return apiKey, token, nil
}

func (s *Service) RevokeApiKey(ctx context.Context, userID, apiKeyID string) error {
	var apiKey ApiKey
	result := s.db.
		WithContext(ctx).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		Delete(&apiKey)
	if result.Error != nil {
		return fmt.Errorf("error deleting API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.APIKeyNotFound()
	}

	return nil
}

func (s *Service) ValidateApiKey(ctx context.Context, apiKey string) (model.User, error) {
	if apiKey == "" {
		return model.User{}, apperror.NoAPIKeyProvided()
	}

	if s.staticApiKey != "" && apiKey == s.staticApiKey {
		return s.initStaticApiKeyUser(ctx)
	}

	now := time.Now()
	hashedKey := utils.CreateSha256Hash(apiKey)

	var key ApiKey
	err := s.db.
		WithContext(ctx).
		Model(&ApiKey{}).
		Clauses(clause.Returning{}).
		Where("key = ? AND expires_at > ?", hashedKey, datatype.DateTime(now)).
		Updates(&ApiKey{
			LastUsedAt: new(datatype.DateTime(now)),
		}).
		Preload("User").
		First(&key).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, apperror.InvalidAPIKey()
	} else if err != nil {
		return model.User{}, fmt.Errorf("error loading API key: %w", err)
	}

	return key.User, nil
}

func (s *Service) ListExpiringApiKeys(ctx context.Context, daysAhead int) ([]ApiKey, error) {
	var keys []ApiKey
	now := time.Now()
	cutoff := now.AddDate(0, 0, daysAhead)

	err := s.db.
		WithContext(ctx).
		Preload("User").
		Where("expires_at > ? AND expires_at <= ? AND expiration_email_sent = ?", datatype.DateTime(now), datatype.DateTime(cutoff), false).
		Find(&keys).
		Error
	if err != nil {
		return nil, fmt.Errorf("error listing API keys: %w", err)
	}

	return keys, nil
}

// MarkExpirationEmailSent records that the expiration notification email was sent for the given API key
func (s *Service) MarkExpirationEmailSent(ctx context.Context, apiKeyID string) error {
	err := s.db.WithContext(ctx).
		Model(&ApiKey{}).
		Where("id = ?", apiKeyID).
		Update("expiration_email_sent", true).
		Error
	if err != nil {
		return fmt.Errorf("error marking API key expiration email sent: %w", err)
	}

	return nil
}

func (s *Service) initStaticApiKeyUser(ctx context.Context) (user model.User, err error) {
	err = s.db.
		WithContext(ctx).
		First(&user, "id = ?", common.StaticApiKeyUserID).
		Error

	if err == nil {
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, fmt.Errorf("error loading static API key user: %w", err)
	}

	usernameSuffix, err := utils.GenerateRandomAlphanumericString(6)
	if err != nil {
		return model.User{}, fmt.Errorf("error generating static API key username suffix: %w", err)
	}

	user = model.User{
		Base: model.Base{
			ID: common.StaticApiKeyUserID,
		},
		FirstName:   "Static API User",
		Username:    "static-api-user-" + usernameSuffix,
		DisplayName: "Static API User",
		IsAdmin:     true,
	}

	err = s.db.
		WithContext(ctx).
		Create(&user).
		Error
	if err != nil {
		return model.User{}, fmt.Errorf("error creating static API key user: %w", err)
	}

	return user, nil
}

func (s *Service) deleteStaticApiKeyUser(ctx context.Context) error {
	err := s.db.
		WithContext(ctx).
		Delete(&model.User{}, "id = ?", common.StaticApiKeyUserID).
		Error
	if err != nil {
		return fmt.Errorf("error deleting static API key user: %w", err)
	}

	return nil
}
