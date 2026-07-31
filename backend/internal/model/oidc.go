package model

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type UserAuthorizedOidcClient struct {
	Scope      datatype.StringList
	LastUsedAt datatype.DateTime `sortable:"true"`

	UserID string `gorm:"primary_key;"`
	User   User

	ClientID string `gorm:"primary_key;"`
	Client   OidcClient
}

// OidcClientType identifies how an OIDC client was registered.
type OidcClientType string

const (
	OidcClientTypeStandard OidcClientType = "standard"
	// OAuth Client ID Metadata Document
	OidcClientTypeCIMD OidcClientType = "cimd"
)

type OidcClient struct {
	Base

	Name                                string `sortable:"true"`
	Description                         string
	Secret                              string
	CallbackURLs                        datatype.StringList
	LogoutCallbackURLs                  datatype.StringList
	ImageType                           *string
	DarkImageType                       *string
	IsPublic                            bool
	PkceEnabled                         bool `sortable:"true" filterable:"true"`
	RequiresReauthentication            bool `sortable:"true" filterable:"true"`
	RequiresPushedAuthorizationRequests bool `sortable:"true" filterable:"true"`
	SkipConsent                         bool `sortable:"true" filterable:"true"`
	Credentials                         OidcClientCredentials
	LaunchURL                           *string
	IsGroupRestricted                   bool           `sortable:"true" filterable:"true"`
	PkceSupported                       bool           `sortable:"true" filterable:"true"`
	ClientType                          OidcClientType `gorm:"default:standard" sortable:"true" filterable:"true"`
	MetadataExpiresAt                   *datatype.DateTime
	MetadataGrantTypes                  datatype.StringList

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

// IsMetadataDocument reports whether the client was synthesized from an OAuth
// Client ID Metadata Document. Its ID is then the https URL of the document.
func (c OidcClient) IsMetadataDocument() bool {
	return c.ClientType == OidcClientTypeCIMD
}

type OidcClientCredentials struct { //nolint:recvcheck
	FederatedIdentities []OidcClientFederatedIdentity `json:"federatedIdentities,omitempty"`
}

type OidcClientFederatedIdentity struct {
	Issuer           string `json:"issuer"`
	Subject          string `json:"subject,omitempty"`
	Audience         string `json:"audience,omitempty"`
	JWKS             string `json:"jwks,omitempty"` // URL of the JWKS
	ReplayProtection bool   `json:"replayProtection,omitempty"`
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
