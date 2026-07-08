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
	Description                         string					 `json:"description" binding:"omitempty,max=255" unorm:"nfc"`
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
}

type OidcClientCreateDto struct {
	OidcClientUpdateDto
	ID string `json:"id" binding:"omitempty,client_id,min=2,max=128"`
}

type OidcClientCredentialsDto struct {
	FederatedIdentities []OidcClientFederatedIdentityDto `json:"federatedIdentities,omitempty"`
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
