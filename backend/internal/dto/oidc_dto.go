package dto

type OidcClientMetaDataDto struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HasLogo bool   `json:"hasLogo"`
}

type OidcClientDto struct {
	OidcClientMetaDataDto
	CallbackURLs       []string `json:"callbackURLs"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
}

type OidcClientWithAllowedUserGroupsDto struct {
	OidcClientDto
	AllowedUserGroups []UserGroupDtoWithUserCount `json:"allowedUserGroups"`
}

type OidcClientCreateDto struct {
	Name               string   `json:"name" binding:"required,max=50"`
	CallbackURLs       []string `json:"callbackURLs" binding:"required"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
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
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
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
