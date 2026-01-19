package dto

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type OidcClientMetaDataDto struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	HasLogo                  bool    `json:"hasLogo"`
	HasDarkLogo              bool    `json:"hasDarkLogo"`
	LaunchURL                *string `json:"launchURL"`
	RequiresReauthentication bool    `json:"requiresReauthentication"`
}

type OidcClientDto struct {
	OidcClientMetaDataDto
	CallbackURLs       []string                 `json:"callbackURLs"`
	LogoutCallbackURLs []string                 `json:"logoutCallbackURLs"`
	IsPublic           bool                     `json:"isPublic"`
	PkceEnabled        bool                     `json:"pkceEnabled"`
	Credentials        OidcClientCredentialsDto `json:"credentials"`
	IsGroupRestricted  bool                     `json:"isGroupRestricted"`
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
	Name                     string                   `json:"name" binding:"required,max=50" unorm:"nfc"`
	CallbackURLs             []string                 `json:"callbackURLs" binding:"omitempty,dive,callback_url"`
	LogoutCallbackURLs       []string                 `json:"logoutCallbackURLs" binding:"omitempty,dive,callback_url"`
	IsPublic                 bool                     `json:"isPublic"`
	PkceEnabled              bool                     `json:"pkceEnabled"`
	RequiresReauthentication bool                     `json:"requiresReauthentication"`
	Credentials              OidcClientCredentialsDto `json:"credentials"`
	LaunchURL                *string                  `json:"launchURL" binding:"omitempty,url"`
	HasLogo                  bool                     `json:"hasLogo"`
	HasDarkLogo              bool                     `json:"hasDarkLogo"`
	LogoURL                  *string                  `json:"logoUrl"`
	DarkLogoURL              *string                  `json:"darkLogoUrl"`
	IsGroupRestricted        bool                     `json:"isGroupRestricted"`
}

type OidcClientCreateDto struct {
	OidcClientUpdateDto
	ID string `json:"id" binding:"omitempty,client_id,min=2,max=128"`
}

type OidcClientCredentialsDto struct {
	FederatedIdentities []OidcClientFederatedIdentityDto `json:"federatedIdentities,omitempty"`
}

type OidcClientFederatedIdentityDto struct {
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject,omitempty"`
	Audience string `json:"audience,omitempty"`
	JWKS     string `json:"jwks,omitempty"`
}

type AuthorizeOidcClientRequestDto struct {
	ClientID              string `json:"clientID" binding:"required"`
	Scope                 string `json:"scope" binding:"required"`
	CallbackURL           string `json:"callbackURL"`
	Nonce                 string `json:"nonce"`
	CodeChallenge         string `json:"codeChallenge"`
	CodeChallengeMethod   string `json:"codeChallengeMethod"`
	ReauthenticationToken string `json:"reauthenticationToken"`
}

type AuthorizeOidcClientResponseDto struct {
	Code        string `json:"code"`
	CallbackURL string `json:"callbackURL"`
	Issuer      string `json:"issuer"`
}

type AuthorizationRequiredDto struct {
	ClientID string `json:"clientID" binding:"required"`
	Scope    string `json:"scope" binding:"required"`
}

type OidcCreateTokensDto struct {
	GrantType           string `form:"grant_type" binding:"required"`
	Code                string `form:"code"`
	DeviceCode          string `form:"device_code"`
	ClientID            string `form:"client_id"`
	ClientSecret        string `form:"client_secret"`
	CodeVerifier        string `form:"code_verifier"`
	RefreshToken        string `form:"refresh_token"`
	ClientAssertion     string `form:"client_assertion"`
	ClientAssertionType string `form:"client_assertion_type"`
	Resource            string `form:"resource"`
}

type OidcIntrospectDto struct {
	Token string `form:"token" binding:"required"`
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

type OidcTokenResponseDto struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	IdToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

type OidcIntrospectionResponseDto struct {
	Active     bool     `json:"active"`
	TokenType  string   `json:"token_type,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	Expiration int64    `json:"exp,omitempty"`
	IssuedAt   int64    `json:"iat,omitempty"`
	NotBefore  int64    `json:"nbf,omitempty"`
	Subject    string   `json:"sub,omitempty"`
	Audience   []string `json:"aud,omitempty"`
	Issuer     string   `json:"iss,omitempty"`
	Identifier string   `json:"jti,omitempty"`
}

type OidcDeviceAuthorizationRequestDto struct {
	ClientID            string `form:"client_id" binding:"required"`
	Scope               string `form:"scope" binding:"required"`
	ClientSecret        string `form:"client_secret"`
	ClientAssertion     string `form:"client_assertion"`
	ClientAssertionType string `form:"client_assertion_type"`
	Nonce               string `form:"nonce"`
}

type OidcDeviceAuthorizationResponseDto struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	QRCodeURI               string `json:"qr_code_uri"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	RequiresAuthorization   bool   `json:"requires_authorization"`
}

type OidcDeviceTokenRequestDto struct {
	GrantType    string `form:"grant_type" binding:"required,eq=urn:ietf:params:oauth:grant-type:device_code"`
	DeviceCode   string `form:"device_code" binding:"required"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
}

type DeviceCodeInfoDto struct {
	Scope                 string                `json:"scope"`
	AuthorizationRequired bool                  `json:"authorizationRequired"`
	Client                OidcClientMetaDataDto `json:"client"`
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
