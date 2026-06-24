package model

import (
	"database/sql/driver"
	"encoding/json"
	"net/url"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type UserAuthorizedOidcClient struct {
	Scope      datatype.StringList
	LastUsedAt datatype.DateTime `sortable:"true"`

	UserID string `gorm:"primary_key;"`
	User   User

	ClientID string `gorm:"primary_key;"`
	Client   OidcClient
}

type OidcClient struct {
	Base

	Name                                string `sortable:"true"`
	Secret                              string
	CallbackURLs                        UrlList
	LogoutCallbackURLs                  UrlList
	ImageType                           *string
	DarkImageType                       *string
	IsPublic                            bool
	PkceEnabled                         bool `sortable:"true" filterable:"true"`
	RequiresReauthentication            bool `sortable:"true" filterable:"true"`
	RequiresPushedAuthorizationRequests bool `sortable:"true" filterable:"true"`
	Credentials                         OidcClientCredentials
	LaunchURL                           *string
	IsGroupRestricted                   bool `sortable:"true" filterable:"true"`

	// IsMetadataDocument marks a client that was synthesized from an OAuth
	// Client ID Metadata Document (draft-ietf-oauth-client-id-metadata-document)
	// rather than registered manually. Its ID is the https URL of the document.
	IsMetadataDocument bool
	// MetadataExpiresAt is the cache expiry of a metadata-document client; nil
	// for normal clients. Once elapsed, the document is refetched.
	MetadataExpiresAt *datatype.DateTime

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

// ClientIDHost returns the host component of a metadata-document client's ID
// (which is a URL), or an empty string for normal clients. It is shown in the
// authorization UI so users can verify the origin of a dynamically-fetched client.
func (c OidcClient) ClientIDHost() string {
	if !c.IsMetadataDocument {
		return ""
	}
	u, err := url.Parse(c.ID)
	if err != nil {
		return ""
	}
	return u.Host
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

type UrlList []string //nolint:recvcheck

func (cu *UrlList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(cu, value)
}

func (cu UrlList) Value() (driver.Value, error) {
	return json.Marshal(cu)
}
