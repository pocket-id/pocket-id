package service

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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
		"preferred_username",
		"groups",
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
	return s.updateCustomClaims(ctx, UserID, userID, claims)
}

// UpdateCustomClaimsForUserGroup updates the custom claims for a user group
func (s *CustomClaimService) UpdateCustomClaimsForUserGroup(ctx context.Context, userGroupID string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	return s.updateCustomClaims(ctx, UserGroupID, userGroupID, claims)
}

// updateCustomClaims updates the custom claims for a user or user group
func (s *CustomClaimService) updateCustomClaims(ctx context.Context, idType idType, value string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	// Check for duplicate keys in the claims slice
	seenKeys := make(map[string]struct{})
	for _, claim := range claims {
		if _, ok := seenKeys[claim.Key]; ok {
			return nil, &common.DuplicateClaimError{Key: claim.Key}
		}
		seenKeys[claim.Key] = struct{}{}
	}

	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	var existingClaims []model.CustomClaim
	err := tx.
		WithContext(ctx).
		Where(string(idType), value).
		Find(&existingClaims).
		Error
	if err != nil {
		return nil, err
	}

	// Delete claims that are not in the new list
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

	// Add or update claims
	for _, claim := range claims {
		if isReservedClaim(claim.Key) {
			return nil, &common.ReservedClaimError{Key: claim.Key}
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

		// Update the claim if it already exists or create a new one
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

	// Get the updated claims
	var updatedClaims []model.CustomClaim
	err = tx.
		WithContext(ctx).
		Where(string(idType)+" = ?", value).
		Find(&updatedClaims).
		Error
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
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
