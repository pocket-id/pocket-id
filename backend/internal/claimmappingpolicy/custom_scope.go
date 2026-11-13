package claimmappingpolicy

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/pocket-id/fosite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// builtInScopes are the scopes of Pocket ID's identity layer, which a claim mapping can be released with and a custom scope can't be named after
var builtInScopes = []string{"openid", "profile", "email", "groups", "offline_access"}

// validateMappingScopes rejects mappings released with a scope that is neither built in nor an existing custom scope
func validateMappingScopes(tx *gorm.DB, mappings []OidcClaimMappingDto) error {
	var customKeys []string
	for _, mapping := range mappings {
		for _, scope := range mapping.Scope {
			if !slices.Contains(builtInScopes, scope) {
				customKeys = append(customKeys, scope)
			}
		}
	}
	if len(customKeys) == 0 {
		return nil
	}

	var existing []string
	err := tx.Model(&CustomScope{}).Where("key IN ?", customKeys).Pluck("key", &existing).Error
	if err != nil {
		return err
	}
	for _, key := range customKeys {
		if !slices.Contains(existing, key) {
			return apperror.ValidationMessage("unknown scope '" + key + "' in claim mapping")
		}
	}
	return nil
}

func (s *Service) ListCustomScopes(ctx context.Context, search string, listRequestOptions utils.ListRequestOptions) ([]CustomScope, utils.PaginationResponse, error) {
	var scopes []CustomScope
	query := s.db.WithContext(ctx).Model(&CustomScope{})

	if search != "" {
		pattern := utils.LikeContainsPattern(search)
		query = query.Where(`key LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\'`, pattern, pattern)
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &scopes)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	usage, err := policiesByScope(s.db.WithContext(ctx))
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}
	for i := range scopes {
		scopes[i].Policies = usage[scopes[i].Key]
	}
	return scopes, response, nil
}

func (s *Service) getCustomScope(ctx context.Context, id string) (CustomScope, error) {
	var scope CustomScope
	err := s.db.WithContext(ctx).First(&scope, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CustomScope{}, apperror.NotFound("Custom scope")
	}
	return scope, err
}

func (s *Service) CreateCustomScope(ctx context.Context, input CustomScopeDto) (CustomScope, error) {
	db := s.db.WithContext(ctx)
	err := validateCustomScopeKey(db, input.Key)
	if err != nil {
		return CustomScope{}, err
	}

	scope := CustomScope{Key: input.Key, Name: input.Name, Description: input.Description}
	err = db.Create(&scope).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return CustomScope{}, apperror.AlreadyInUse("key")
	}
	return scope, err
}

// UpdateCustomScope changes the scope's name and description
// The key can't change, because claim mappings and the apps that request the scope refer to it
func (s *Service) UpdateCustomScope(ctx context.Context, id string, update dto.Update[CustomScopeDto]) (CustomScope, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Lock the scope and compute the update from it, so a PATCH merges onto the state it saves over
	var scope CustomScope
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&scope, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CustomScope{}, apperror.NotFound("Custom scope")
	} else if err != nil {
		return CustomScope{}, err
	}
	input, err := update(scope)
	if err != nil {
		return CustomScope{}, err
	}
	if input.Key != scope.Key {
		return CustomScope{}, apperror.InvalidField("key", "immutable", "can't be changed")
	}

	scope.Name = input.Name
	scope.Description = input.Description
	err = tx.WithContext(ctx).Save(&scope).Error
	if err != nil {
		return CustomScope{}, err
	}
	return scope, tx.Commit().Error
}

// validateCustomScopeKey rejects keys that would be ambiguous as a scope
func validateCustomScopeKey(tx *gorm.DB, key string) error {
	if !fosite.IsValidScopeToken(key) {
		return apperror.InvalidField("key", "invalid_format", "contains characters that are not valid in an OAuth scope")
	}
	if slices.Contains(builtInScopes, strings.ToLower(key)) || authz.Scope(key).Known() {
		return apperror.InvalidField("key", "reserved", "is reserved by Pocket ID")
	}

	// An API permission with the same key would make a request for that scope mean two different things
	var permissionCount int64
	err := tx.Table("api_permissions").Where("key = ?", key).Count(&permissionCount).Error
	if err != nil {
		return err
	}
	if permissionCount > 0 {
		return apperror.InvalidField("key", "reserved", "is already used by an API permission")
	}

	return nil
}

// DeleteCustomScope deletes the scope unless a claim mapping is still released with it, since that mapping would then never release its claim
func (s *Service) DeleteCustomScope(ctx context.Context, id string) error {
	scope, err := s.getCustomScope(ctx, id)
	if err != nil {
		return err
	}

	usage, err := policiesByScope(s.db.WithContext(ctx))
	if err != nil {
		return err
	}
	if policies := usage[scope.Key]; len(policies) > 0 {
		return apperror.ValidationMessage("the scope is still used by the claim mapping policy '" + policies[0] + "'")
	}

	return s.db.WithContext(ctx).Delete(&CustomScope{}, "id = ?", scope.ID).Error
}

// policiesByScope returns the names of the policies that release at least one claim with each scope
func policiesByScope(db *gorm.DB) (map[string][]string, error) {
	var policies []OidcClaimMappingPolicy
	err := db.Select("name", "claim_mappings").Order("name").Find(&policies).Error
	if err != nil {
		return nil, err
	}

	usage := make(map[string][]string)
	for _, policy := range policies {
		seen := make(map[string]bool)
		for _, mapping := range policy.ClaimMappings {
			for _, scope := range mapping.Scope {
				if !seen[scope] {
					seen[scope] = true
					usage[scope] = append(usage[scope], policy.Name)
				}
			}
		}
	}
	return usage, nil
}

// ClientPolicy returns the claim mapping policy assigned to the client, or the default policy when the client has none or doesn't exist
// Both candidates are fetched in a single query, and the assigned policy is ordered first so it wins over the default
func ClientPolicy(ctx context.Context, db *gorm.DB, clientID string) (*OidcClaimMappingPolicy, error) {
	var policy OidcClaimMappingPolicy
	err := db.WithContext(ctx).
		Where("oidc_claim_mapping_policies.id = (SELECT claim_mapping_policy_id FROM oidc_clients WHERE oidc_clients.id = ?)", clientID).
		Or("oidc_claim_mapping_policies.is_default = ?", true).
		Order("oidc_claim_mapping_policies.is_default ASC").
		First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// ClientCustomScopeKeys returns the keys of the custom scopes the client may request, which are those its policy releases claims with
// A metadata document client never gets one, because anyone can register it and it falls back to the default policy
func ClientCustomScopeKeys(ctx context.Context, db *gorm.DB, client model.OidcClient) ([]string, error) {
	if client.IsMetadataDocument() {
		return nil, nil
	}

	// Only a database that Bootstrap hasn't run on yet has no default policy
	policy, err := ClientPolicy(ctx, db, client.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var candidates []string
	for _, mapping := range policy.ClaimMappings {
		for _, scope := range mapping.Scope {
			if !slices.Contains(builtInScopes, scope) {
				candidates = append(candidates, scope)
			}
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// A scope deleted while a policy save was still validating it can linger in the policy, so only existing scopes count
	var keys []string
	err = db.WithContext(ctx).Model(&CustomScope{}).Where("key IN ?", candidates).Order("key").Pluck("key", &keys).Error
	return keys, err
}

// CustomScopesByKey returns the custom scopes with the given keys, or every custom scope when keys is nil
func CustomScopesByKey(ctx context.Context, db *gorm.DB, keys []string) ([]CustomScope, error) {
	query := db.WithContext(ctx).Order("key")
	if keys != nil {
		query = query.Where("key IN ?", keys)
	}

	var scopes []CustomScope
	err := query.Find(&scopes).Error
	return scopes, err
}
