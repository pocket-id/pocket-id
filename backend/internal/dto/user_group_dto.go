package dto

import (
	"errors"
	"unicode/utf8"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type UserGroupDto struct {
	ID                 string                  `json:"id"`
	FriendlyName       string                  `json:"friendlyName"`
	Name               string                  `json:"name"`
	CustomClaims       []CustomClaimDto        `json:"customClaims"`
	LdapID             *string                 `json:"ldapId"`
	CreatedAt          datatype.DateTime       `json:"createdAt"`
	Users              []UserDto               `json:"users"`
	AllowedOidcClients []OidcClientMetaDataDto `json:"allowedOidcClients"`
}

type UserGroupMinimalDto struct {
	ID           string            `json:"id"`
	FriendlyName string            `json:"friendlyName"`
	Name         string            `json:"name"`
	CustomClaims []CustomClaimDto  `json:"customClaims"`
	UserCount    int64             `json:"userCount"`
	LdapID       *string           `json:"ldapId"`
	CreatedAt    datatype.DateTime `json:"createdAt"`
}

type UserGroupUpdateAllowedOidcClientsDto struct {
	OidcClientIDs []string `json:"oidcClientIds" required:"true"`
}

type UserGroupCreateDto struct {
	FriendlyName string `json:"friendlyName" required:"true" minLength:"2" maxLength:"50" unorm:"nfc"`
	Name         string `json:"name" required:"true" minLength:"2" maxLength:"255" unorm:"nfc"`
	LdapID       string `json:"-"`
}

func (g UserGroupCreateDto) Validate() error {
	friendlyNameLength := utf8.RuneCountInString(g.FriendlyName)
	if friendlyNameLength < 2 || friendlyNameLength > 50 {
		return errors.New("friendly name is invalid")
	}
	nameLength := utf8.RuneCountInString(g.Name)
	if nameLength < 2 || nameLength > 255 {
		return errors.New("name is invalid")
	}
	return nil
}

type UserGroupUpdateUsersDto struct {
	UserIDs []string `json:"userIds" required:"true"`
}
