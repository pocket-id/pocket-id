package dto

type OidcClaimMappingPolicyMetadataDto struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

type OidcClaimMappingPolicyDto struct {
	OidcClaimMappingPolicyMetadataDto
	ClaimMappings []OidcClaimMappingDto `json:"claimMappings"`
}

type OidcClaimMappingDto struct {
	ClaimName   string   `json:"claimName" binding:"required,max=255"`
	SourceType  string   `json:"sourceType" binding:"required,oneof=user_field custom_claim static"`
	SourceValue string   `json:"sourceValue" binding:"required,max=255"`
	Scope       []string `json:"scope" binding:"omitempty,dive,oneof=openid profile email address phone offline_access groups"`
	AccessToken bool     `json:"accessToken"`
	IDToken     bool     `json:"idToken"`
	UserInfo    bool     `json:"userInfo"`
}
