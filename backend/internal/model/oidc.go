package model

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type UserAuthorizedOidcClient struct {
	Scope      string
	LastUsedAt datatype.DateTime `sortable:"true"`

	UserID string `gorm:"primary_key;"`
	User   User

	ClientID string `gorm:"primary_key;"`
	Client   OidcClient
}

func (c UserAuthorizedOidcClient) Scopes() []string {
	if len(c.Scope) == 0 {
		return []string{}
	}

	return strings.Split(c.Scope, " ")
}

type OidcAuthorizationCode struct {
	Base

	Code                      string
	Scope                     string
	Nonce                     string
	CodeChallenge             *string
	CodeChallengeMethodSha256 *bool
	ExpiresAt                 datatype.DateTime

	UserID string
	User   User

	ClientID string
}

type OidcClient struct {
	Base

	Name                     string `sortable:"true"`
	Secret                   string
	CallbackURLs             UrlList
	LogoutCallbackURLs       UrlList
	ImageType                *string
	DarkImageType            *string
	IsPublic                 bool
	PkceEnabled              bool `filterable:"true"`
	RequiresReauthentication bool `filterable:"true"`
	Credentials              OidcClientCredentials
	LaunchURL                *string

	AllowedUserGroups         []UserGroup `gorm:"many2many:oidc_clients_allowed_user_groups;"`
	CreatedByID               *string
	CreatedBy                 *User
	UserAuthorizedOidcClients []UserAuthorizedOidcClient `gorm:"foreignKey:ClientID;references:ID"`
}

func (c OidcClient) HasLogo() bool {
	return c.ImageType != nil && *c.ImageType != ""
}

func (c OidcClient) HasDarkLogo() bool {
	return c.DarkImageType != nil && *c.DarkImageType != ""
}

type OidcRefreshToken struct {
	Base

	Token     string
	ExpiresAt datatype.DateTime
	Scope     string

	UserID string
	User   User

	ClientID string
	Client   OidcClient
}

func (c OidcRefreshToken) Scopes() []string {
	if len(c.Scope) == 0 {
		return []string{}
	}

	return strings.Split(c.Scope, " ")
}

type OidcClientCredentials struct { //nolint:recvcheck
	FederatedIdentities []OidcClientFederatedIdentity `json:"federatedIdentities,omitempty"`
}

type OidcClientFederatedIdentity struct {
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject,omitempty"`
	Audience string `json:"audience,omitempty"`
	JWKS     string `json:"jwks,omitempty"` // URL of the JWKS
}

func (occ OidcClientCredentials) FederatedIdentityForIssuer(issuer string) (OidcClientFederatedIdentity, bool) {
	if issuer == "" {
		return OidcClientFederatedIdentity{}, false
	}

	for _, fi := range occ.FederatedIdentities {
		if fi.Issuer == issuer {
			return fi, true
		}
	}

	return OidcClientFederatedIdentity{}, false
}

func (occ *OidcClientCredentials) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(occ, value)
}

func (occ OidcClientCredentials) Value() (driver.Value, error) {
	return json.Marshal(occ)
}

type UrlList []string //nolint:recvcheck

func (cu *UrlList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(cu, value)
}

func (cu UrlList) Value() (driver.Value, error) {
	return json.Marshal(cu)
}

type OidcDeviceCode struct {
	Base
	DeviceCode   string
	UserCode     string
	Scope        string
	ExpiresAt    datatype.DateTime
	IsAuthorized bool

	UserID   *string
	User     User
	ClientID string
	Client   OidcClient
}
