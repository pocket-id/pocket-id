package dto

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type OidcClientMetaDataDto struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	HasLogo                  bool    `json:"hasLogo"`
	HasDarkLogo              bool    `json:"hasDarkLogo"`
	LaunchURL                *string `json:"launchURL"`
	RequiresReauthentication bool    `json:"requiresReauthentication"`
	ClientType               string  `json:"clientType"`
}

type OidcClientDto struct {
	OidcClientMetaDataDto
	CallbackURLs                        []string                 `json:"callbackURLs"`
	LogoutCallbackURLs                  []string                 `json:"logoutCallbackURLs"`
	IsPublic                            bool                     `json:"isPublic"`
	PkceEnabled                         bool                     `json:"pkceEnabled"`
	RequiresPushedAuthorizationRequests bool                     `json:"requiresPushedAuthorizationRequests"`
	SkipConsent                         bool                     `json:"skipConsent"`
	Credentials                         OidcClientCredentialsDto `json:"credentials"`
	IsGroupRestricted                   bool                     `json:"isGroupRestricted"`
	PkceSupported                       bool                     `json:"pkceSupported,omitempty"`
	AccessTokenDurationMinutes          int64                    `json:"accessTokenDurationMinutes"`
	RefreshTokenDurationMinutes         int64                    `json:"refreshTokenDurationMinutes"`
}

type OidcClientWithAllowedUserGroupsDto struct {
	OidcClientDto
	AllowedUserGroups []UserGroupMinimalDto `json:"allowedUserGroups"`
}

type OidcClientWithAllowedGroupsCountDto struct {
	OidcClientDto
	AllowedUserGroupsCount int64 `json:"allowedUserGroupsCount"`
}

type OidcClientUpdateDto struct {
	Name                                string                   `json:"name" binding:"required,max=50" unorm:"nfc"`
	Description                         string                   `json:"description" binding:"omitempty,max=150" unorm:"nfc"`
	CallbackURLs                        []string                 `json:"callbackURLs" binding:"omitempty,dive,callback_url_pattern"`
	LogoutCallbackURLs                  []string                 `json:"logoutCallbackURLs" binding:"omitempty,dive,callback_url_pattern"`
	IsPublic                            bool                     `json:"isPublic"`
	PkceEnabled                         bool                     `json:"pkceEnabled"`
	RequiresReauthentication            bool                     `json:"requiresReauthentication"`
	RequiresPushedAuthorizationRequests bool                     `json:"requiresPushedAuthorizationRequests"`
	SkipConsent                         bool                     `json:"skipConsent"`
	Credentials                         OidcClientCredentialsDto `json:"credentials"`
	LaunchURL                           *string                  `json:"launchURL" binding:"omitempty,url"`
	HasLogo                             bool                     `json:"hasLogo"`
	HasDarkLogo                         bool                     `json:"hasDarkLogo"`
	LogoURL                             *string                  `json:"logoUrl"`
	DarkLogoURL                         *string                  `json:"darkLogoUrl"`
	IsGroupRestricted                   bool                     `json:"isGroupRestricted"`
	AccessTokenDurationMinutes          int64                    `json:"accessTokenDurationMinutes" binding:"omitempty,token_duration"`
	RefreshTokenDurationMinutes         int64                    `json:"refreshTokenDurationMinutes" binding:"omitempty,token_duration"`
}

type OidcClientCreateDto struct {
	OidcClientUpdateDto
	ID string `json:"id" binding:"omitempty,client_id,min=2,max=128"`
}

// OidcClientSecretDto describes a client secret without disclosing its value, which is only ever returned right after the secret is created
type OidcClientSecretDto struct {
	ID string `json:"id"`
	// Prefix holds the first few characters of the secret in clear text, and is empty for secrets migrated from the single-secret column
	Prefix    string             `json:"prefix"`
	CreatedAt datatype.DateTime  `json:"createdAt"`
	ExpiresAt *datatype.DateTime `json:"expiresAt"`
	IsActive  bool               `json:"isActive"`
}

// OidcClientSecretCreateDto is the request body for creating a new client secret
type OidcClientSecretCreateDto struct {
	// Secret allows callers to supply their own value instead of having Pocket ID generate one
	Secret string `json:"secret" binding:"omitempty,min=16,printascii"`
	// ExpiresAt makes the secret unusable after the given time (if nil, secrets don't expire)
	ExpiresAt *datatype.DateTime `json:"expiresAt"`
}

// OidcClientSecretCreatedDto is returned when a secret is created, and is the only response that contains the secret's value
type OidcClientSecretCreatedDto struct {
	OidcClientSecretDto
	Secret string `json:"secret"`
}

type OidcClientCredentialsDto struct {
	FederatedIdentities []OidcClientFederatedIdentityDto `json:"federatedIdentities,omitempty"`
	// Secrets is read-only: secrets are managed through the dedicated client secret endpoints and any value sent by a client is ignored
	// It is always serialized, as an empty list when the client has no secrets, so clients don't need to handle a missing value
	Secrets []OidcClientSecretDto `json:"secrets"`
}

type OidcClientFederatedIdentityDto struct {
	Issuer           string `json:"issuer"`
	Subject          string `json:"subject,omitempty"`
	Audience         string `json:"audience,omitempty"`
	JWKS             string `json:"jwks,omitempty"`
	ReplayProtection bool   `json:"replayProtection"`
}

type OidcUpdateAllowedUserGroupsDto struct {
	UserGroupIDs []string `json:"userGroupIds" binding:"required"`
}

type OidcLogoutDto struct {
	IdTokenHint           string `form:"id_token_hint"`
	ClientId              string `form:"client_id"`
	PostLogoutRedirectUri string `form:"post_logout_redirect_uri"`
	State                 string `form:"state"`
}

type OidcDeviceAuthorizationResponseDto struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type ScopeInfoDto struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type DeviceCodeInfoDto struct {
	Scope                    []string              `json:"scope"`
	ScopeInfo                []ScopeInfoDto        `json:"scopeInfo"`
	AuthorizationRequired    bool                  `json:"authorizationRequired"`
	ReauthenticationRequired bool                  `json:"reauthenticationRequired"`
	Client                   OidcClientMetaDataDto `json:"client"`
}

type AuthorizedOidcClientDto struct {
	Scope      string                `json:"scope"`
	Client     OidcClientMetaDataDto `json:"client"`
	LastUsedAt datatype.DateTime     `json:"lastUsedAt"`
}

type OidcClientPreviewDto struct {
	IdToken     map[string]any `json:"idToken"`
	AccessToken map[string]any `json:"accessToken"`
	UserInfo    map[string]any `json:"userInfo"`
}

type AccessibleOidcClientDto struct {
	OidcClientMetaDataDto
	LastUsedAt *datatype.DateTime `json:"lastUsedAt"`
}
