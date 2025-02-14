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
}

type OidcClientWithAllowedUserGroupsDto struct {
	PublicOidcClientDto
	CallbackURLs       []string                    `json:"callbackURLs"`
	LogoutCallbackURLs []string                    `json:"logoutCallbackURLs"`
	IsPublic           bool                        `json:"isPublic"`
	PkceEnabled        bool                        `json:"pkceEnabled"`
	AllowedUserGroups  []UserGroupDtoWithUserCount `json:"allowedUserGroups"`
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
	Code         string `form:"code" binding:"required"`
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
