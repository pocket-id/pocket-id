package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
)

type UserAuthorizedOidcClient struct {
	Scope  string
	UserID string `gorm:"primary_key;"`
	User   User

	ClientID string `gorm:"primary_key;"`
	Client   OidcClient
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

	Name               string `sortable:"true"`
	Secret             string
	CallbackURLs       UrlList
	LogoutCallbackURLs UrlList
	ImageType          *string
	HasLogo            bool `gorm:"-"`
	IsPublic           bool
	PkceEnabled        bool

	AllowedUserGroups []UserGroup `gorm:"many2many:oidc_clients_allowed_user_groups;"`
	CreatedByID       string
	CreatedBy         User
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

func (c *OidcClient) AfterFind(_ *gorm.DB) (err error) {
	// Compute HasLogo field
	c.HasLogo = c.ImageType != nil && *c.ImageType != ""
	return nil
}

type UrlList []string //nolint:recvcheck

func (cu *UrlList) Scan(value interface{}) error {
	if v, ok := value.([]byte); ok {
		return json.Unmarshal(v, cu)
	} else {
		return errors.New("type assertion to []byte failed")
	}
}

func (cu UrlList) Value() (driver.Value, error) {
	return json.Marshal(cu)
}
