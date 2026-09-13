package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

// Valid user field names that can be used as mapping sources
var validUserFieldSources = map[string]bool{
	string(model.UserFieldID):            true,
	string(model.UserFieldEmail):         true,
	string(model.UserFieldEmailVerified): true,
	string(model.UserFieldFirstName):     true,
	string(model.UserFieldLastName):      true,
	string(model.UserFieldFullName):      true,
	string(model.UserFieldDisplayName):   true,
	string(model.UserFieldUsername):      true,
	string(model.UserFieldLocale):        true,
	string(model.UserFieldPicture):       true,
	string(model.UserFieldGroups):        true,
}

// Reserved claims that cannot be remapped
var reservedClaimsForMapping = map[string]bool{
	"iss":       true,
	"aud":       true,
	"exp":       true,
	"iat":       true,
	"auth_time": true,
	"nonce":     true,
	"acr":       true,
	"amr":       true,
	"azp":       true,
	"nbf":       true,
	"jti":       true,
}

type OidcClaimMappingPolicyService struct {
	db *gorm.DB
}

func NewOidcClaimMappingPolicyService(db *gorm.DB) *OidcClaimMappingPolicyService {
	return &OidcClaimMappingPolicyService{
		db: db,
	}
}

// ListClaimMappingPolicy returns a paginated list of claim mapping policys with their metadata
func (s *OidcClaimMappingPolicyService) ListClaimMappingPolicy(ctx context.Context, name string, listRequestOptions utils.ListRequestOptions) ([]model.OidcClaimMappingPolicy, utils.PaginationResponse, error) {

	var claimMappingPolicys []model.OidcClaimMappingPolicy
	query := s.db.WithContext(ctx).Model(&model.OidcClaimMappingPolicy{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &claimMappingPolicys)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return claimMappingPolicys, response, err
}

func (s *OidcClaimMappingPolicyService) GetClaimMappingPolicy(ctx context.Context, listID string) (model.OidcClaimMappingPolicy, error) {
	var claimMappingPolicy model.OidcClaimMappingPolicy
	err := s.db.WithContext(ctx).First(&claimMappingPolicy, "id = ?", listID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.OidcClaimMappingPolicy{}, apperror.NotFound("Claim mapping policy")
	}
	if err != nil {
		return model.OidcClaimMappingPolicy{}, err
	}

	return claimMappingPolicy, nil
}

// ListClientsByClaimMappingPolicy returns a paginated list of the OIDC clients assigned to a claim
// mapping policy, optionally filtered by client name
func (s *OidcClaimMappingPolicyService) ListClientsByClaimMappingPolicy(ctx context.Context, policyID string, name string, listRequestOptions utils.ListRequestOptions) ([]model.OidcClient, utils.PaginationResponse, error) {
	// Resolve the policy first, so an unknown ID reports a 404 instead of an empty page
	if _, err := s.GetClaimMappingPolicy(ctx, policyID); err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	var clients []model.OidcClient
	query := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("claim_mapping_policy_id = ?", policyID)

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return clients, response, nil
}

// ListAssignableClients returns a paginated list of the OIDC clients that are not on this claim
// mapping policy yet, so they can be offered for assignment
func (s *OidcClaimMappingPolicyService) ListAssignableClients(ctx context.Context, policyID string, name string, listRequestOptions utils.ListRequestOptions) ([]model.OidcClient, utils.PaginationResponse, error) {
	if _, err := s.GetClaimMappingPolicy(ctx, policyID); err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	var clients []model.OidcClient
	query := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("claim_mapping_policy_id IS NULL OR claim_mapping_policy_id <> ?", policyID)

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return clients, response, nil
}

// AssignClientToClaimMappingPolicy points a client at the policy. A client has a single policy, so
// this replaces whichever one it was on before.
func (s *OidcClaimMappingPolicyService) AssignClientToClaimMappingPolicy(ctx context.Context, policyID string, clientID string) error {
	if _, err := s.GetClaimMappingPolicy(ctx, policyID); err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ?", clientID).
		Update("claim_mapping_policy_id", policyID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("OIDC client")
	}

	return nil
}

// RemoveClientFromClaimMappingPolicy detaches a client from the policy, which makes it fall back to
// the default one. The client must currently be on this policy, so a stale request cannot detach a
// client that has since been moved elsewhere.
func (s *OidcClaimMappingPolicyService) RemoveClientFromClaimMappingPolicy(ctx context.Context, policyID string, clientID string) error {
	if _, err := s.GetClaimMappingPolicy(ctx, policyID); err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ? AND claim_mapping_policy_id = ?", clientID, policyID).
		Update("claim_mapping_policy_id", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("OIDC client")
	}

	return nil
}

func (s *OidcClaimMappingPolicyService) GetDefaultClaimMappingPolicy(ctx context.Context) (*model.OidcClaimMappingPolicy, error) {
	var claimMappingPolicy model.OidcClaimMappingPolicy
	err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&claimMappingPolicy).Error
	if err != nil {
		return nil, err
	}
	return &claimMappingPolicy, nil
}

func (s *OidcClaimMappingPolicyService) CreateClaimMappingPolicy(ctx context.Context, dto dto.OidcClaimMappingPolicyDto) (*dto.OidcClaimMappingPolicyDto, error) {
	// Validate claim mappings
	if err := validateClaimMappings(dto.ClaimMappings); err != nil {
		return &dto, err
	}
	// Create the claim mapping policy
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	claimMappingPolicy := model.OidcClaimMappingPolicy{}
	// dto -> model
	updateClaimMappingPolicyFromDto(&claimMappingPolicy, dto)

	if err := tx.Create(&claimMappingPolicy).Error; err != nil {
		tx.Rollback()
		return &dto, err
	}

	if claimMappingPolicy.IsDefault {
		// Unset default for other claim mapping policys
		if err := tx.Model(&model.OidcClaimMappingPolicy{}).Where("id != ?", claimMappingPolicy.ID).Update("is_default", false).Error; err != nil {
			tx.Rollback()
			return &dto, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return &dto, err
	}

	dto.ID = claimMappingPolicy.ID
	return &dto, nil
}

func (s *OidcClaimMappingPolicyService) UpdateClaimMappingPolicy(ctx context.Context, listID string, dto dto.OidcClaimMappingPolicyDto) (*dto.OidcClaimMappingPolicyDto, error) {
	// Validate claim mappings
	if err := validateClaimMappings(dto.ClaimMappings); err != nil {
		return &dto, err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var claimMappingPolicy model.OidcClaimMappingPolicy
	if err := tx.First(&claimMappingPolicy, "id = ?", listID).Error; err != nil {
		tx.Rollback()
		return &dto, err
	}
	// dto -> model
	updateClaimMappingPolicyFromDto(&claimMappingPolicy, dto)

	if err := tx.Save(&claimMappingPolicy).Error; err != nil {
		tx.Rollback()
		return &dto, err
	}

	if claimMappingPolicy.IsDefault {
		// Unset default for other claim mapping policys
		if err := tx.Model(&model.OidcClaimMappingPolicy{}).Where("id != ?", claimMappingPolicy.ID).Update("is_default", false).Error; err != nil {
			tx.Rollback()
			return &dto, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return &dto, err
	}

	dto.ID = claimMappingPolicy.ID
	return &dto, nil
}

func updateClaimMappingPolicyFromDto(claimMappingPolicy *model.OidcClaimMappingPolicy, dto dto.OidcClaimMappingPolicyDto) {
	claimMappingPolicy.Name = dto.Name
	claimMappingPolicy.IsDefault = dto.IsDefault
	claimMappingPolicy.ClaimMappings = make([]model.OidcClaimMapping, len(dto.ClaimMappings))
	for i, mapping := range dto.ClaimMappings {
		claimMappingPolicy.ClaimMappings[i] = model.OidcClaimMapping{
			ClaimName:   mapping.ClaimName,
			SourceType:  model.OidcClaimMappingSourceType(mapping.SourceType),
			SourceValue: mapping.SourceValue,
			Scope:       mapping.Scope,
			AccessToken: mapping.AccessToken,
			IDToken:     mapping.IDToken,
			UserInfo:    mapping.UserInfo,
		}
	}
}

func (s *OidcClaimMappingPolicyService) DeleteClaimMappingPolicy(ctx context.Context, policyID string) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var claimMappingPolicy model.OidcClaimMappingPolicy
	if err := tx.First(&claimMappingPolicy, "id = ?", policyID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if claimMappingPolicy.IsDefault {
		tx.Rollback()
		return fmt.Errorf("cannot delete the default claim mapping policy")
	}

	// Clear the reference on any client that was using this policy, so they fall back to the default one
	if err := tx.Model(&model.OidcClient{}).Where("claim_mapping_policy_id = ?", policyID).Update("claim_mapping_policy_id", nil).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&claimMappingPolicy).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func validateClaimMappings(mappings []dto.OidcClaimMappingDto) error {
	seenClaims := make(map[string]bool)

	for _, mapping := range mappings {

		if len(mapping.ClaimName) == 0 || len(mapping.SourceType) == 0 || len(mapping.SourceValue) == 0 {
			return fmt.Errorf("claim name, source type and source value are required")
		}
		// Check for duplicates
		if seenClaims[mapping.ClaimName] {
			return fmt.Errorf("duplicate claim mapping for '%s'", mapping.ClaimName)
		}
		seenClaims[mapping.ClaimName] = true

		// Check if claim is reserved
		if reservedClaimsForMapping[mapping.ClaimName] {
			return fmt.Errorf("cannot remap reserved claim '%s'", mapping.ClaimName)
		}

		// Validate source based on type
		switch mapping.SourceType {
		case "user_field":
			if !validUserFieldSources[mapping.SourceValue] {
				return fmt.Errorf("invalid user field '%s' for mapping", mapping.SourceValue)
			}
		case "custom_claim":
			// Custom claim key validation
			if len(mapping.SourceValue) == 0 || len(mapping.SourceValue) > 255 {
				return fmt.Errorf("invalid custom claim key length")
			}
		case "static":
			// Static values are always valid (string or JSON)
		default:
			return fmt.Errorf("invalid source type '%s'", mapping.SourceType)
		}
	}

	if !seenClaims["sub"] {
		return fmt.Errorf("claim mapping must include 'sub' claim")
	}
	return nil
}
