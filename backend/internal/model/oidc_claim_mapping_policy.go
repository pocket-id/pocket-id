package model

import (
	"database/sql/driver"
	"encoding/json"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// OidcClaimMappingPolicy defines a set of claim mappings that can be applied to a client.
// There's only one default claim mapping set, but additional sets can be created for specific clients if needed.
type OidcClaimMappingPolicy struct {
	Base

	Name      string
	IsDefault bool

	ClaimMappings OidcClaimMappings
}

// OidcClaimMappings used to store the claim mappings as json in the database.
type OidcClaimMappings []OidcClaimMapping //nolint:recvcheck

func (ocm *OidcClaimMappings) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(ocm, value)
}

func (ocm OidcClaimMappings) Value() (driver.Value, error) {
	return json.Marshal(ocm)
}

// OidcClaimMapping defines how a claim is mapped from a source to a target claim name
// It can be used to map claims from user fields, custom claims, or static values to a specific claim name in the token
// The claim can be included in the access token and/or ID token, following the scope of the request.
type OidcClaimMapping struct {
	ClaimName   string                     `json:"claimName"`
	SourceType  OidcClaimMappingSourceType `json:"sourceType"`
	SourceValue string                     `json:"sourceValue"`
	Scope       datatype.StringList        `json:"scope"`
	AccessToken bool                       `json:"accessToken"`
	IDToken     bool                       `json:"idToken"`
	UserInfo    bool                       `json:"userInfo"`
}

type OidcClaimMappingSourceType string

const (
	MappingSourceUserField   OidcClaimMappingSourceType = "user_field"
	MappingSourceCustomClaim OidcClaimMappingSourceType = "custom_claim"
	MappingSourceStatic      OidcClaimMappingSourceType = "static"
)

type OidcUserField string

const (
	UserFieldID            OidcUserField = "id"
	UserFieldEmail         OidcUserField = "email"
	UserFieldEmailVerified OidcUserField = "email_verified"
	UserFieldFirstName     OidcUserField = "first_name"
	UserFieldLastName      OidcUserField = "last_name"
	UserFieldDisplayName   OidcUserField = "display_name"
	UserFieldFullName      OidcUserField = "full_name"
	UserFieldUsername      OidcUserField = "username"
	UserFieldLocale        OidcUserField = "locale"
	UserFieldPicture       OidcUserField = "picture"
	UserFieldGroups        OidcUserField = "groups"
)
