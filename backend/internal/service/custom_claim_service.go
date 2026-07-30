package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type CustomClaimService struct {
	db *gorm.DB
}

func NewCustomClaimService(db *gorm.DB) *CustomClaimService {
	return &CustomClaimService{db: db}
}

// isReservedClaim checks if a claim key is reserved e.g. email, preferred_username
func isReservedClaim(key string) bool {
	switch key {
	case "given_name",
		"family_name",
		"name",
		"email",
		"email_verified",
		"preferred_username",
		"display_name",
		"groups",
		TokenTypeClaim,
		"sub",
		"iss",
		"aud",
		"exp",
		"iat",
		"auth_time",
		"nonce",
		"acr",
		"amr",
		"azp",
		"nbf",
		"jti":
		return true
	default:
		return false
	}
}

// idType is the type of the id used to identify the user or user group
type idType string

const (
	UserID      idType = "user_id"
	UserGroupID idType = "user_group_id"
)

// UpdateCustomClaimsForUser updates the custom claims for a user
func (s *CustomClaimService) UpdateCustomClaimsForUser(ctx context.Context, userID string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Reject missing owners before replacing claims so invalid IDs cannot appear successful
	if err := ensureCustomClaimOwnerExists(ctx, tx, UserID, userID); err != nil {
		return nil, err
	}

	updatedClaims, err := s.updateCustomClaimsInternal(ctx, UserID, userID, claims, tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

// UpdateCustomClaimsForUserGroup updates the custom claims for a user group
func (s *CustomClaimService) UpdateCustomClaimsForUserGroup(ctx context.Context, userGroupID string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Reject missing owners before replacing claims so invalid IDs cannot appear successful
	if err := ensureCustomClaimOwnerExists(ctx, tx, UserGroupID, userGroupID); err != nil {
		return nil, err
	}

	updatedClaims, err := s.updateCustomClaimsInternal(ctx, UserGroupID, userGroupID, claims, tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

func ensureCustomClaimOwnerExists(ctx context.Context, tx *gorm.DB, ownerType idType, ownerID string) error {
	// Select the owner model and corresponding client-safe not-found error
	var (
		target   any
		notFound error
	)
	switch ownerType {
	case UserID:
		target = &model.User{}
		notFound = apperror.UserNotFound()
	case UserGroupID:
		target = &model.UserGroup{}
		notFound = apperror.NotFound("User group")
	default:
		return fmt.Errorf("unsupported custom claim owner type %q", ownerType)
	}

	// Verify the owner in the active transaction before changing its claims
	err := tx.WithContext(ctx).
		Select("id").
		First(target, "id = ?", ownerID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound
	}

	return err
}

// updateCustomClaimsInternal keeps claim replacements within the caller's transaction
func (s *CustomClaimService) updateCustomClaimsInternal(ctx context.Context, idType idType, value string, claims []dto.CustomClaimCreateDto, tx *gorm.DB) ([]model.CustomClaim, error) {
	// Reject duplicate keys before changing persisted claims
	seenKeys := make(map[string]struct{})
	for _, claim := range claims {
		if _, ok := seenKeys[claim.Key]; ok {
			return nil, apperror.DuplicateClaim(claim.Key)
		}
		seenKeys[claim.Key] = struct{}{}
	}

	var existingClaims []model.CustomClaim
	err := tx.
		WithContext(ctx).
		Where(string(idType), value).
		Find(&existingClaims).
		Error
	if err != nil {
		return nil, err
	}

	// Remove stale claims before applying the requested replacement set
	for _, existingClaim := range existingClaims {
		found := false
		for _, claim := range claims {
			if claim.Key == existingClaim.Key {
				found = true
				break
			}
		}

		if !found {
			err = tx.
				WithContext(ctx).
				Delete(&existingClaim).
				Error
			if err != nil {
				return nil, err
			}
		}
	}

	// Reject reserved keys and persist each requested claim
	for _, claim := range claims {
		if isReservedClaim(claim.Key) {
			return nil, apperror.ReservedClaim(claim.Key)
		}
		customClaim := model.CustomClaim{
			Key:   claim.Key,
			Value: claim.Value,
		}

		switch idType {
		case UserID:
			customClaim.UserID = &value
		case UserGroupID:
			customClaim.UserGroupID = &value
		}

		// Preserve claim identity when updating an existing owner and key pair
		err = tx.
			WithContext(ctx).
			Where(string(idType)+" = ? AND key = ?", value, claim.Key).
			Assign(&customClaim).
			FirstOrCreate(&model.CustomClaim{}).
			Error
		if err != nil {
			return nil, err
		}
	}

	// Return the persisted replacement set to the caller
	var updatedClaims []model.CustomClaim
	err = tx.
		WithContext(ctx).
		Where(string(idType)+" = ?", value).
		Find(&updatedClaims).
		Error
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

func (s *CustomClaimService) GetCustomClaimsForUser(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomClaim, error) {
	var customClaims []model.CustomClaim
	err := tx.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&customClaims).
		Error
	return customClaims, err
}

func (s *CustomClaimService) GetCustomClaimsForUserGroup(ctx context.Context, userGroupID string, tx *gorm.DB) ([]model.CustomClaim, error) {
	var customClaims []model.CustomClaim
	err := tx.
		WithContext(ctx).
		Where("user_group_id = ?", userGroupID).
		Find(&customClaims).
		Error
	return customClaims, err
}

// GetCustomClaimsForUserWithUserGroups returns the custom claims of a user and all user groups the user is a member of,
// prioritizing the user's claims over user group claims with the same key.
func (s *CustomClaimService) GetCustomClaimsForUserWithUserGroups(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomClaim, error) {
	// Get the custom claims of the user
	customClaims, err := s.GetCustomClaimsForUser(ctx, userID, tx)
	if err != nil {
		return nil, err
	}

	// Store user's claims in a map to prioritize and prevent duplicates
	claimsMap := make(map[string]model.CustomClaim)
	for _, claim := range customClaims {
		claimsMap[claim.Key] = claim
	}

	// Get all user groups of the user
	var userGroupsOfUser []model.UserGroup
	err = tx.
		WithContext(ctx).
		Preload("CustomClaims").
		Joins("JOIN user_groups_users ON user_groups_users.user_group_id = user_groups.id").
		Where("user_groups_users.user_id = ?", userID).
		Find(&userGroupsOfUser).Error
	if err != nil {
		return nil, err
	}

	// Add only non-duplicate custom claims from user groups
	for _, userGroup := range userGroupsOfUser {
		for _, groupClaim := range userGroup.CustomClaims {
			// Only add claim if it does not exist in the user's claims
			if _, exists := claimsMap[groupClaim.Key]; !exists {
				claimsMap[groupClaim.Key] = groupClaim
			}
		}
	}

	// Convert the claimsMap back to a slice
	finalClaims := make([]model.CustomClaim, 0, len(claimsMap))
	for _, claim := range claimsMap {
		finalClaims = append(finalClaims, claim)
	}

	return finalClaims, nil
}

// GetSuggestions returns a list of custom claim keys that have been used before
func (s *CustomClaimService) GetSuggestions(ctx context.Context) ([]string, error) {
	var customClaimsKeys []string

	err := s.db.
		WithContext(ctx).
		Model(&model.CustomClaim{}).
		Group("key").
		Order("COUNT(*) DESC").
		Pluck("key", &customClaimsKeys).Error

	return customClaimsKeys, err
}
