package apikey

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Service holds the business logic for managing user API keys
type Service struct {
	db           *gorm.DB
	staticApiKey string
	auditLog     AuditLogger
}

func newService(db *gorm.DB, staticApiKey string, auditLog AuditLogger) *Service {
	return &Service{db: db, staticApiKey: staticApiKey, auditLog: auditLog}
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

// CreateApiKey creates a key for the user that holds only the requested scopes, each of which must be in grantable
func (s *Service) CreateApiKey(ctx context.Context, userID string, grantable authz.ScopeSet, input apiKeyCreateDto, origin auditlogs.Origin) (ApiKey, string, error) {
	// Check if expiration is in the future
	if !input.ExpiresAt.ToTime().After(time.Now()) {
		return ApiKey{}, "", apperror.InvalidAPIKeyExpiration()
	}

	scopes, err := grantedScopes(input.Scopes, grantable)
	if err != nil {
		return ApiKey{}, "", err
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
		Scopes:      scopes,
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
	s.auditLog.Record(ctx, auditlogs.EventAPIKeyCreated, origin, userID, auditlogs.Data{"apiKeyName": apiKey.Name}, nil)

	// Return the raw token only once - it cannot be retrieved later
	return apiKey, token, nil
}

// UpdateApiKey changes a key's name, description and scopes without changing its token
func (s *Service) UpdateApiKey(ctx context.Context, userID, apiKeyID string, grantable authz.ScopeSet, update dto.Update[apiKeyUpdateDto]) (ApiKey, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the key and compute the update from it, so a PATCH merges onto the state it saves over
	var apiKey ApiKey
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		First(&apiKey).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ApiKey{}, apperror.APIKeyNotFound()
	} else if err != nil {
		return ApiKey{}, fmt.Errorf("error loading API key: %w", err)
	}
	input, err := update(apiKey)
	if err != nil {
		return ApiKey{}, err
	}
	scopes, err := grantedScopes(input.Scopes, grantable)
	if err != nil {
		return ApiKey{}, err
	}

	apiKey.Name = input.Name
	apiKey.Description = input.Description
	apiKey.Scopes = scopes

	err = tx.WithContext(ctx).Save(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ApiKey{}, apperror.AlreadyInUse("API key name")
		}
		return ApiKey{}, fmt.Errorf("error saving API key: %w", err)
	}

	return apiKey, tx.Commit().Error
}

// grantedScopes returns the requested scopes sorted and without duplicates, or an error if any of them is not in grantable
func grantedScopes(requested []string, grantable authz.ScopeSet) (datatype.StringList, error) {
	for _, scope := range requested {
		if !grantable.Has(authz.Scope(scope)) {
			return nil, apperror.InvalidAPIKeyScope(scope)
		}
	}

	scopes := slices.Clone(requested)
	slices.Sort(scopes)
	return slices.Compact(scopes), nil
}

// RenewApiKey replaces a key's token
// If the key has not expired yet, its old token keeps working for gracePeriod, capped at the key's previous expiration
// The token replaced by an earlier rotation stops working right away
func (s *Service) RenewApiKey(ctx context.Context, userID, apiKeyID string, expiration time.Time, gracePeriod time.Duration, origin auditlogs.Origin) (ApiKey, string, error) {
	now := time.Now()

	// Check if expiration is in the future
	if !expiration.After(now) {
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

	// Generate a secure random API key
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error generating API key token: %w", err)
	}

	// Keep the old token working during the grace period so its integrations can switch over without downtime
	apiKey.PreviousKey = nil
	apiKey.PreviousKeyExpiresAt = nil
	previousExpiresAt := now.Add(gracePeriod)
	if previousExpiresAt.After(apiKey.ExpiresAt.ToTime()) {
		previousExpiresAt = apiKey.ExpiresAt.ToTime()
	}
	if previousExpiresAt.After(now) {
		apiKey.PreviousKey = new(apiKey.Key)
		apiKey.PreviousKeyExpiresAt = new(datatype.DateTime(previousExpiresAt))
	}

	apiKey.Key = utils.CreateSha256Hash(token)
	apiKey.ExpiresAt = datatype.DateTime(expiration)
	// The renewed key expires again, so its owner should be notified again
	apiKey.ExpirationEmailSent = false

	err = tx.WithContext(ctx).Save(&apiKey).Error
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error saving API key: %w", err)
	}
	s.auditLog.Record(ctx, auditlogs.EventAPIKeyRenewed, origin, userID, auditlogs.Data{"apiKeyName": apiKey.Name}, tx)

	err = tx.Commit().Error
	if err != nil {
		return ApiKey{}, "", fmt.Errorf("error committing transaction: %w", err)
	}

	return apiKey, token, nil
}

func (s *Service) RevokeApiKey(ctx context.Context, userID, apiKeyID string, origin auditlogs.Origin) error {
	var apiKey ApiKey
	result := s.db.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		Delete(&apiKey)
	if result.Error != nil {
		return fmt.Errorf("error deleting API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.APIKeyNotFound()
	}
	s.auditLog.Record(ctx, auditlogs.EventAPIKeyRevoked, origin, userID, auditlogs.Data{"apiKeyName": apiKey.Name}, nil)

	return nil
}

// ValidateApiKey resolves a raw API key to the stored key with its owner preloaded
// The static API key belongs to no user, so it resolves to a key without a UserID, which no stored key can have
func (s *Service) ValidateApiKey(ctx context.Context, apiKey string) (ApiKey, error) {
	if apiKey == "" {
		return ApiKey{}, apperror.NoAPIKeyProvided()
	}

	if s.staticApiKey != "" && subtle.ConstantTimeCompare([]byte(apiKey), []byte(s.staticApiKey)) == 1 {
		return ApiKey{}, nil
	}

	now := time.Now()
	hashedKey := utils.CreateSha256Hash(apiKey)

	var key ApiKey
	err := s.db.
		WithContext(ctx).
		Model(&ApiKey{}).
		Clauses(clause.Returning{}).
		Where("(key = ? AND expires_at > ?) OR (previous_key = ? AND previous_key_expires_at > ?)", hashedKey, datatype.DateTime(now), hashedKey, datatype.DateTime(now)).
		Updates(&ApiKey{
			LastUsedAt: new(datatype.DateTime(now)),
		}).
		Preload("User").
		First(&key).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ApiKey{}, apperror.InvalidAPIKey()
	} else if err != nil {
		return ApiKey{}, fmt.Errorf("error loading API key: %w", err)
	}

	return key, nil
}

// AllScopes returns every scope an API key can hold
func AllScopes() datatype.StringList {
	scopes := make(datatype.StringList, 0)
	for scope := range authz.UserScopes(true, authz.KindAPIKey) {
		scopes = append(scopes, string(scope))
	}
	return scopes
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
