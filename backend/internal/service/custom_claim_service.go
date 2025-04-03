package service

import (
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

// Reserved claims
var reservedClaims = map[string]struct{}{
	"given_name":         {},
	"family_name":        {},
	"name":               {},
	"email":              {},
	"preferred_username": {},
	"groups":             {},
	"sub":                {},
	"iss":                {},
	"aud":                {},
	"exp":                {},
	"iat":                {},
	"auth_time":          {},
	"nonce":              {},
	"acr":                {},
	"amr":                {},
	"azp":                {},
	"nbf":                {},
	"jti":                {},
}

type CustomClaimService struct {
	db *gorm.DB
}

func NewCustomClaimService(db *gorm.DB) *CustomClaimService {
	return &CustomClaimService{db: db}
}

// isReservedClaim checks if a claim key is reserved e.g. email, preferred_username
func isReservedClaim(key string) bool {
	_, ok := reservedClaims[key]
	return ok
}

// idType is the type of the id used to identify the user or user group
type idType string

const (
	UserID      idType = "user_id"
	UserGroupID idType = "user_group_id"
)

// UpdateCustomClaimsForUser updates the custom claims for a user
func (s *CustomClaimService) UpdateCustomClaimsForUser(userID string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	return s.updateCustomClaims(UserID, userID, claims)
}

// UpdateCustomClaimsForUserGroup updates the custom claims for a user group
func (s *CustomClaimService) UpdateCustomClaimsForUserGroup(userGroupID string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	return s.updateCustomClaims(UserGroupID, userGroupID, claims)
}

// updateCustomClaims updates the custom claims for a user or user group
func (s *CustomClaimService) updateCustomClaims(idType idType, value string, claims []dto.CustomClaimCreateDto) ([]model.CustomClaim, error) {
	// Check for duplicate keys in the claims slice
	seenKeys := make(map[string]bool)
	for _, claim := range claims {
		if seenKeys[claim.Key] {
			return nil, &common.DuplicateClaimError{Key: claim.Key}
		}
		seenKeys[claim.Key] = true
	}

	var existingClaims []model.CustomClaim
	err := s.db.Where(string(idType), value).Find(&existingClaims).Error
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
			err = s.db.Delete(&existingClaim).Error
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
		err = s.db.Where(string(idType)+" = ? AND key = ?", value, claim.Key).Assign(&customClaim).FirstOrCreate(&model.CustomClaim{}).Error
		if err != nil {
			return nil, err
		}
	}

	// Get the updated claims
	var updatedClaims []model.CustomClaim
	err = s.db.Where(string(idType)+" = ?", value).Find(&updatedClaims).Error
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

func (s *CustomClaimService) GetCustomClaimsForUser(userID string) ([]model.CustomClaim, error) {
	var customClaims []model.CustomClaim
	err := s.db.Where("user_id = ?", userID).Find(&customClaims).Error
	return customClaims, err
}

func (s *CustomClaimService) GetCustomClaimsForUserGroup(userGroupID string) ([]model.CustomClaim, error) {
	var customClaims []model.CustomClaim
	err := s.db.Where("user_group_id = ?", userGroupID).Find(&customClaims).Error
	return customClaims, err
}

// GetCustomClaimsForUserWithUserGroups returns the custom claims of a user and all user groups the user is a member of,
// prioritizing the user's claims over user group claims with the same key.
func (s *CustomClaimService) GetCustomClaimsForUserWithUserGroups(userID string) ([]model.CustomClaim, error) {
	// Get the custom claims of the user
	customClaims, err := s.GetCustomClaimsForUser(userID)
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
	err = s.db.Preload("CustomClaims").
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
func (s *CustomClaimService) GetSuggestions() ([]string, error) {
	var customClaimsKeys []string

	err := s.db.Model(&model.CustomClaim{}).
		Group("key").
		Order("COUNT(*) DESC").
		Pluck("key", &customClaimsKeys).Error

	return customClaimsKeys, err
}
