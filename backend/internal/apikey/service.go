package apikey

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		return nil, utils.PaginationResponse{}, err
	}

	return apiKeys, pagination, nil
}

func (s *Service) CreateApiKey(ctx context.Context, userID string, input apiKeyCreateDto) (ApiKey, string, error) {
	// Check if expiration is in the future
	if !input.ExpiresAt.ToTime().After(time.Now()) {
		return ApiKey{}, "", &common.APIKeyExpirationDateError{}
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return ApiKey{}, "", err
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
			return ApiKey{}, "", &common.AlreadyInUseError{Property: "API key name"}
		}
		return ApiKey{}, "", err
	}

	// Return the raw token only once - it cannot be retrieved later
	return apiKey, token, nil
}

func (s *Service) RenewApiKey(ctx context.Context, userID, apiKeyID string, expiration time.Time) (ApiKey, string, error) {
	// Check if expiration is in the future
	if !expiration.After(time.Now()) {
		return ApiKey{}, "", &common.APIKeyExpirationDateError{}
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

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApiKey{}, "", &common.APIKeyNotFoundError{}
		}
		return ApiKey{}, "", err
	}

	// Only allow renewal if the key has already expired
	if apiKey.ExpiresAt.ToTime().After(time.Now()) {
		return ApiKey{}, "", &common.APIKeyNotExpiredError{}
	}

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return ApiKey{}, "", err
	}

	apiKey.Key = utils.CreateSha256Hash(token)
	apiKey.ExpiresAt = datatype.DateTime(expiration)

	err = tx.WithContext(ctx).Save(&apiKey).Error
	if err != nil {
		return ApiKey{}, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return ApiKey{}, "", err
	}

	return apiKey, token, nil
}

func (s *Service) RevokeApiKey(ctx context.Context, userID, apiKeyID string) error {
	var apiKey ApiKey
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

func (s *Service) ValidateApiKey(ctx context.Context, apiKey string) (model.User, error) {
	if apiKey == "" {
		return model.User{}, &common.NoAPIKeyProvidedError{}
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
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, &common.InvalidAPIKeyError{}
		}

		return model.User{}, err
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

	return keys, err
}

// MarkExpirationEmailSent records that the expiration notification email was sent for the given API key
func (s *Service) MarkExpirationEmailSent(ctx context.Context, apiKeyID string) error {
	return s.db.WithContext(ctx).
		Model(&ApiKey{}).
		Where("id = ?", apiKeyID).
		Update("expiration_email_sent", true).
		Error
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
		return model.User{}, err
	}

	usernameSuffix, err := utils.GenerateRandomAlphanumericString(6)
	if err != nil {
		return model.User{}, err
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

	return user, err
}

func (s *Service) deleteStaticApiKeyUser(ctx context.Context) error {
	return s.db.
		WithContext(ctx).
		Delete(&model.User{}, "id = ?", common.StaticApiKeyUserID).
		Error
}
