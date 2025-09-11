package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
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
	HasLogo                  bool `gorm:"-"`
	IsPublic                 bool
	PkceEnabled              bool
	RequiresReauthentication bool
	Credentials              OidcClientCredentials
	LaunchURL                *string

	AllowedUserGroups         []UserGroup `gorm:"many2many:oidc_clients_allowed_user_groups;"`
	CreatedByID               *string
	CreatedBy                 *User
	UserAuthorizedOidcClients []UserAuthorizedOidcClient `gorm:"foreignKey:ClientID;references:ID"`
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

func (c *OidcClient) AfterFind(_ *gorm.DB) (err error) {
	// Compute HasLogo field
	c.HasLogo = c.ImageType != nil && *c.ImageType != ""
	return nil
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
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, occ)
	case string:
		return json.Unmarshal([]byte(v), occ)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

func (occ OidcClientCredentials) Value() (driver.Value, error) {
	return json.Marshal(occ)
}

type UrlList []string //nolint:recvcheck

func (cu *UrlList) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, cu)
	case string:
		return json.Unmarshal([]byte(v), cu)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
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
