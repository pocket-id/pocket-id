package claimmappingpolicy

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type OidcClaimMappingPolicyMetadataDto struct {
	ID        string            `json:"id"`
	Name      string            `json:"name" binding:"required,max=255"`
	IsDefault bool              `json:"isDefault"`
	CreatedAt datatype.DateTime `json:"createdAt"`
	// ClaimCount and ClientCount are only set in responses and ignored in requests
	ClaimCount int `json:"claimCount"`
	// ClientCount includes every client without a policy of its own when the policy is the default
	ClientCount int64 `json:"clientCount"`
}

type OidcClaimMappingPolicyDto struct {
	OidcClaimMappingPolicyMetadataDto

	ClaimMappings []OidcClaimMappingDto `json:"claimMappings" binding:"dive"`
}

type OidcClaimMappingDto struct {
	ClaimName   string   `json:"claimName" binding:"required,max=255"`
	SourceType  string   `json:"sourceType" binding:"required,oneof=user_field custom_claim static"`
	SourceValue string   `json:"sourceValue" binding:"required,max=255"`
	Scope       []string `json:"scope" binding:"min=1,dive,required,max=128"`
	AccessToken bool     `json:"accessToken"`
	IDToken     bool     `json:"idToken"`
	UserInfo    bool     `json:"userInfo"`
}

type CustomScopeDto struct {
	ID          string            `json:"id"`
	CreatedAt   datatype.DateTime `json:"createdAt"`
	Key         string            `json:"key" binding:"required,max=128"`
	Name        string            `json:"name" binding:"required,max=255"`
	Description string            `json:"description" binding:"required,max=1000"`
	// Policies is only set in responses and ignored in requests
	Policies []string `json:"policies"`
}

func customScopeToDto(scope CustomScope) CustomScopeDto {
	return CustomScopeDto{
		ID:          scope.ID,
		CreatedAt:   scope.CreatedAt,
		Key:         scope.Key,
		Name:        scope.Name,
		Description: scope.Description,
		Policies:    scope.Policies,
	}
}
