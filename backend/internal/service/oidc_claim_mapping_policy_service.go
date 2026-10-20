package service

import (
	"context"
	"errors"
	"reflect"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// isValidUserFieldSources checks if a user field name can be used as a mapping source
func isValidUserFieldSources(field string) bool {
	switch model.OidcUserField(field) {
	case model.UserFieldID,
		model.UserFieldEmail,
		model.UserFieldEmailVerified,
		model.UserFieldFirstName,
		model.UserFieldLastName,
		model.UserFieldFullName,
		model.UserFieldDisplayName,
		model.UserFieldUsername,
		model.UserFieldLocale,
		model.UserFieldPicture,
		model.UserFieldGroups:
		return true
	default:
		return false
	}
}

// isReservedClaimsForMapping checks if a claim is reserved and cannot be remapped
func isReservedClaimsForMapping(claim string) bool {
	switch claim {
	case "sub",
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
		query = query.Where(`name LIKE ? ESCAPE '\'`, utils.LikeContainsPattern(name))
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &claimMappingPolicys)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return claimMappingPolicys, response, err
}

func (s *OidcClaimMappingPolicyService) GetClaimMappingPolicy(ctx context.Context, policyID string) (model.OidcClaimMappingPolicy, error) {
	return s.getClaimMappingPolicyInternal(ctx, s.db, policyID, false)
}

// getClaimMappingPolicyInternal loads a policy through the given handle, which lets callers read it inside their own transaction
// With forUpdate the row stays locked until the transaction ends, so concurrent writers on the same policy are serialized
func (s *OidcClaimMappingPolicyService) getClaimMappingPolicyInternal(ctx context.Context, tx *gorm.DB, policyID string, forUpdate bool) (model.OidcClaimMappingPolicy, error) {
	var claimMappingPolicy model.OidcClaimMappingPolicy
	q := tx.WithContext(ctx)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&claimMappingPolicy, "id = ?", policyID).Error
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
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Resolve the policy first, so an unknown ID reports a 404 instead of an empty page
	_, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, false)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	var clients []model.OidcClient
	query := tx.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("claim_mapping_policy_id = ?", policyID)

	if name != "" {
		query = query.Where(`name LIKE ? ESCAPE '\'`, utils.LikeContainsPattern(name))
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return clients, response, nil
}

// ListAssignableClients returns a paginated list of the OIDC clients that are not on this claim
// mapping policy yet, so they can be offered for assignment
func (s *OidcClaimMappingPolicyService) ListAssignableClients(ctx context.Context, policyID string, name string, listRequestOptions utils.ListRequestOptions) ([]model.OidcClient, utils.PaginationResponse, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Resolve the policy first, so an unknown ID reports a 404 instead of listing every client
	_, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, false)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	var clients []model.OidcClient
	query := tx.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("claim_mapping_policy_id IS NULL OR claim_mapping_policy_id <> ?", policyID)

	if name != "" {
		query = query.Where(`name LIKE ? ESCAPE '\'`, utils.LikeContainsPattern(name))
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return clients, response, nil
}

// AssignClientToClaimMappingPolicy points a client at the policy. A client has a single policy, so
// this replaces whichever one it was on before.
func (s *OidcClaimMappingPolicyService) AssignClientToClaimMappingPolicy(ctx context.Context, policyID string, clientID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the policy so a concurrent delete cannot remove it between the check and the assignment
	_, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, true)
	if err != nil {
		return err
	}

	// Point the client at the policy
	result := tx.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ?", clientID).
		Update("claim_mapping_policy_id", policyID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("OIDC client")
	}

	return tx.Commit().Error
}

// RemoveClientFromClaimMappingPolicy detaches a client from the policy, which makes it fall back to
// the default one. The client must currently be on this policy, so a stale request cannot detach a
// client that has since been moved elsewhere.
func (s *OidcClaimMappingPolicyService) RemoveClientFromClaimMappingPolicy(ctx context.Context, policyID string, clientID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the policy so the detach is serialized with assignments and deletion of the same policy
	_, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, true)
	if err != nil {
		return err
	}

	// Detach the client only if it is still on this policy
	result := tx.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ? AND claim_mapping_policy_id = ?", clientID, policyID).
		Update("claim_mapping_policy_id", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("OIDC client")
	}

	return tx.Commit().Error
}

func (s *OidcClaimMappingPolicyService) CreateClaimMappingPolicy(ctx context.Context, dto dto.OidcClaimMappingPolicyDto) (*dto.OidcClaimMappingPolicyDto, error) {
	// Validate claim mappings
	err := validateClaimMappings(dto.ClaimMappings)
	if err != nil {
		return &dto, err
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Create the claim mapping policy
	claimMappingPolicy := model.OidcClaimMappingPolicy{}
	updateClaimMappingPolicyFromDto(&claimMappingPolicy, dto)

	err = tx.WithContext(ctx).Create(&claimMappingPolicy).Error
	if err != nil {
		return &dto, err
	}

	// Unset default for other claim mapping policies, in the same transaction so there is never more than one default
	if claimMappingPolicy.IsDefault {
		err := tx.WithContext(ctx).Model(&model.OidcClaimMappingPolicy{}).Where("id != ?", claimMappingPolicy.ID).Update("is_default", false).Error
		if err != nil {
			return &dto, err
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return &dto, err
	}

	dto.ID = claimMappingPolicy.ID
	return &dto, nil
}

func (s *OidcClaimMappingPolicyService) UpdateClaimMappingPolicy(ctx context.Context, policyID string, dto dto.OidcClaimMappingPolicyDto) (*dto.OidcClaimMappingPolicyDto, error) {
	// Validate claim mappings
	err := validateClaimMappings(dto.ClaimMappings)
	if err != nil {
		return &dto, err
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the policy so concurrent updates and deletes on it are serialized
	claimMappingPolicy, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, true)
	if err != nil {
		return &dto, err
	}

	// Apply the changes
	original := claimMappingPolicy
	updateClaimMappingPolicyFromDto(&claimMappingPolicy, dto)

	// The built-in policy is managed by the migrations, so only its default flag may change
	if original.ID == model.BuiltInClaimMappingPolicyID &&
		(claimMappingPolicy.Name != original.Name || !reflect.DeepEqual(claimMappingPolicy.ClaimMappings, original.ClaimMappings)) {
		return &dto, apperror.BuiltInClaimMappingPolicyUpdate()
	}

	// There must always be a default policy, and the built-in one is the fallback when the current default is unset
	unsetDefault := original.IsDefault && !claimMappingPolicy.IsDefault
	if unsetDefault && original.ID == model.BuiltInClaimMappingPolicyID {
		return &dto, apperror.ValidationMessage("The built-in claim mapping policy can't be unset as default, set another policy as default instead")
	}

	err = tx.WithContext(ctx).Save(&claimMappingPolicy).Error
	if err != nil {
		return &dto, err
	}

	// Unset default for other claim mapping policies, in the same transaction so there is never more than one default
	if claimMappingPolicy.IsDefault {
		err = tx.WithContext(ctx).Model(&model.OidcClaimMappingPolicy{}).Where("id != ?", claimMappingPolicy.ID).Update("is_default", false).Error
		if err != nil {
			return &dto, err
		}
	}

	// Promote the built-in policy when the default one is unset, so clients without a policy keep getting claims
	if unsetDefault {
		err = tx.WithContext(ctx).Model(&model.OidcClaimMappingPolicy{}).Where("id = ?", model.BuiltInClaimMappingPolicyID).Update("is_default", true).Error
		if err != nil {
			return &dto, err
		}
	}

	err = tx.Commit().Error
	if err != nil {
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
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the policy so no client can be assigned to it, and it cannot become the default, while it is being deleted
	claimMappingPolicy, err := s.getClaimMappingPolicyInternal(ctx, tx, policyID, true)
	if err != nil {
		return err
	}

	// The built-in policy is managed by the migrations and must always exist
	if claimMappingPolicy.ID == model.BuiltInClaimMappingPolicyID {
		return apperror.BuiltInClaimMappingPolicyUpdate()
	}

	// Clear the reference on any client that was using this policy, so they fall back to the default one
	err = tx.WithContext(ctx).Model(&model.OidcClient{}).Where("claim_mapping_policy_id = ?", policyID).Update("claim_mapping_policy_id", nil).Error
	if err != nil {
		return err
	}

	err = tx.WithContext(ctx).Delete(&claimMappingPolicy).Error
	if err != nil {
		return err
	}

	// Promote the built-in policy when the default one is deleted, so clients without a policy keep getting claims
	if claimMappingPolicy.IsDefault {
		err = tx.WithContext(ctx).Model(&model.OidcClaimMappingPolicy{}).Where("id = ?", model.BuiltInClaimMappingPolicyID).Update("is_default", true).Error
		if err != nil {
			return err
		}
	}

	return tx.Commit().Error
}

func validateClaimMappings(mappings []dto.OidcClaimMappingDto) error {
	seenClaims := make(map[string]bool)

	for _, mapping := range mappings {
		if len(mapping.ClaimName) == 0 || len(mapping.SourceType) == 0 || len(mapping.SourceValue) == 0 {
			return apperror.ValidationMessage("claim name, source type and source value are required")
		}
		// Check for duplicates
		if seenClaims[mapping.ClaimName] {
			return apperror.ValidationMessage("duplicate claim mapping for '" + mapping.ClaimName + "'")
		}
		seenClaims[mapping.ClaimName] = true

		// Check if claim is reserved
		if isReservedClaimsForMapping(mapping.ClaimName) {
			return apperror.ValidationMessage("cannot remap reserved claim '" + mapping.ClaimName + "'")
		}

		// Validate source based on type
		switch mapping.SourceType {
		case "user_field":
			if !isValidUserFieldSources(mapping.SourceValue) {
				return apperror.ValidationMessage("invalid user field '" + mapping.SourceValue + "' for mapping")
			}
		case "custom_claim":
			// Custom claim key validation
			if len(mapping.SourceValue) == 0 || len(mapping.SourceValue) > 255 {
				return apperror.ValidationMessage("invalid custom claim key length")
			}
		case "static":
			// Static values are always valid (string or JSON)
		default:
			return apperror.ValidationMessage("invalid source type '" + mapping.SourceType + "'")
		}
	}

	return nil
}
