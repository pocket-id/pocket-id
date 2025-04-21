package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

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
	DeviceCodeEnabled  bool

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
	DeviceCode   string            `gorm:"unique;not null"`
	UserCode     string            `gorm:"unique;not null"`
	Scope        string            `gorm:"not null"`
	ExpiresAt    datatype.DateTime `gorm:"not null"`
	Interval     int               `gorm:"not null"`
	LastPollTime *datatype.DateTime
	IsAuthorized bool `gorm:"not null;default:false"`
	UserID       *string
	User         User       `gorm:"foreignKey:UserID"`
	ClientID     string     `gorm:"not null"`
	Client       OidcClient `gorm:"foreignKey:ClientID"`
}
