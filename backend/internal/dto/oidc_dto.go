package dto

type PublicOidcClientDto struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HasLogo bool   `json:"hasLogo"`
}

type OidcClientDto struct {
	PublicOidcClientDto
	CallbackURLs       []string `json:"callbackURLs"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
	DeviceCodeEnabled  bool     `json:"deviceCodeEnabled"`
}

type OidcClientWithAllowedUserGroupsDto struct {
	PublicOidcClientDto
	CallbackURLs       []string                    `json:"callbackURLs"`
	LogoutCallbackURLs []string                    `json:"logoutCallbackURLs"`
	IsPublic           bool                        `json:"isPublic"`
	PkceEnabled        bool                        `json:"pkceEnabled"`
	DeviceCodeEnabled  bool                        `json:"deviceCodeEnabled"`
	AllowedUserGroups  []UserGroupDtoWithUserCount `json:"allowedUserGroups"`
}

type OidcClientCreateDto struct {
	Name               string   `json:"name" binding:"required,max=50"`
	CallbackURLs       []string `json:"callbackURLs" binding:"required"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
	DeviceCodeEnabled  bool     `json:"deviceCodeEnabled"`
}

type AuthorizeOidcClientRequestDto struct {
	ClientID            string `json:"clientID" binding:"required"`
	Scope               string `json:"scope" binding:"required"`
	CallbackURL         string `json:"callbackURL"`
	Nonce               string `json:"nonce"`
	CodeChallenge       string `json:"codeChallenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod"`
}

type AuthorizeOidcClientResponseDto struct {
	Code        string `json:"code"`
	CallbackURL string `json:"callbackURL"`
}

type AuthorizationRequiredDto struct {
	ClientID string `json:"clientID" binding:"required"`
	Scope    string `json:"scope" binding:"required"`
}

type OidcCreateTokensDto struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code"`
	DeviceCode   string `form:"device_code"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
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

type OidcDeviceAuthorizationRequestDto struct {
	ClientID     string `form:"client_id" binding:"required"`
	Scope        string `form:"scope" binding:"required"`
	ClientSecret string `form:"client_secret"`
}

type OidcDeviceAuthorizationResponseDto struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
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
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	Scope      string `json:"scope"`
}
