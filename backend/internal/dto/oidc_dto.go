package dto

import (
	"github.com/danielgtaylor/huma/v2"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

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
	Name                                string                   `json:"name" required:"true" maxLength:"50" unorm:"nfc"`
	Description                         string                   `json:"description" required:"false" maxLength:"150" unorm:"nfc"`
	CallbackURLs                        []string                 `json:"callbackURLs" required:"false"`
	LogoutCallbackURLs                  []string                 `json:"logoutCallbackURLs" required:"false"`
	IsPublic                            bool                     `json:"isPublic" required:"false"`
	PkceEnabled                         bool                     `json:"pkceEnabled" required:"false"`
	RequiresReauthentication            bool                     `json:"requiresReauthentication" required:"false"`
	RequiresPushedAuthorizationRequests bool                     `json:"requiresPushedAuthorizationRequests" required:"false"`
	SkipConsent                         bool                     `json:"skipConsent" required:"false"`
	Credentials                         OidcClientCredentialsDto `json:"credentials" required:"false"`
	LaunchURL                           *string                  `json:"launchURL" required:"false" format:"uri"`
	HasLogo                             bool                     `json:"hasLogo" required:"false"`
	HasDarkLogo                         bool                     `json:"hasDarkLogo" required:"false"`
	LogoURL                             *string                  `json:"logoUrl" required:"false"`
	DarkLogoURL                         *string                  `json:"darkLogoUrl" required:"false"`
	IsGroupRestricted                   bool                     `json:"isGroupRestricted" required:"false"`
}

type OidcClientCreateDto struct {
	OidcClientUpdateDto
	ID string `json:"id" required:"false" minLength:"2" maxLength:"128" pattern:"^[a-zA-Z0-9._-]+$" patternDescription:"letters, numbers, dots, underscores, and hyphens"`
}

type OidcClientCredentialsDto struct {
	FederatedIdentities []OidcClientFederatedIdentityDto `json:"federatedIdentities,omitempty"`
}

type OidcClientFederatedIdentityDto struct {
	Issuer           string `json:"issuer" required:"false"`
	Subject          string `json:"subject,omitempty"`
	Audience         string `json:"audience,omitempty"`
	JWKS             string `json:"jwks,omitempty"`
	ReplayProtection bool   `json:"replayProtection" required:"false"`
}

type OidcUpdateAllowedUserGroupsDto struct {
	UserGroupIDs []string `json:"userGroupIds" required:"true"`
}

func (d *OidcClientUpdateDto) Resolve(huma.Context) []error {
	return validateCallbackURLLists(d.CallbackURLs, d.LogoutCallbackURLs)
}

func (d *OidcClientCreateDto) Resolve(huma.Context) []error {
	errs := validateCallbackURLLists(d.CallbackURLs, d.LogoutCallbackURLs)
	if d.ID != "" && !ValidateClientID(d.ID) {
		errs = append(errs, &huma.ErrorDetail{Location: "body.id", Message: "Client ID is invalid"})
	}
	return errs
}

func validateCallbackURLLists(callbackURLs, logoutCallbackURLs []string) []error {
	var errs []error
	for _, callbackURL := range callbackURLs {
		if !ValidateCallbackURLPattern(callbackURL) {
			errs = append(errs, &huma.ErrorDetail{Location: "body.callbackURLs", Message: "Callback URL pattern is invalid"})
		}
	}
	for _, callbackURL := range logoutCallbackURLs {
		if !ValidateCallbackURLPattern(callbackURL) {
			errs = append(errs, &huma.ErrorDetail{Location: "body.logoutCallbackURLs", Message: "Logout callback URL pattern is invalid"})
		}
	}
	return errs
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
