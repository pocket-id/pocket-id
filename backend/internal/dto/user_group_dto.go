package dto

import (
	"golang.org/x/text/unicode/norm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type UserGroupDto struct {
	ID           string            `json:"id"`
	FriendlyName string            `json:"friendlyName"`
	Name         string            `json:"name"`
	CustomClaims []CustomClaimDto  `json:"customClaims"`
	LdapID       *string           `json:"ldapId"`
	CreatedAt    datatype.DateTime `json:"createdAt"`
}

type UserGroupDtoWithUsers struct {
	ID           string            `json:"id"`
	FriendlyName string            `json:"friendlyName"`
	Name         string            `json:"name"`
	CustomClaims []CustomClaimDto  `json:"customClaims"`
	Users        []UserDto         `json:"users"`
	LdapID       *string           `json:"ldapId"`
	CreatedAt    datatype.DateTime `json:"createdAt"`
}

type UserGroupDtoWithUserCount struct {
	ID           string            `json:"id"`
	FriendlyName string            `json:"friendlyName"`
	Name         string            `json:"name"`
	CustomClaims []CustomClaimDto  `json:"customClaims"`
	UserCount    int64             `json:"userCount"`
	LdapID       *string           `json:"ldapId"`
	CreatedAt    datatype.DateTime `json:"createdAt"`
}

type UserGroupCreateDto struct {
	FriendlyName string `json:"friendlyName" binding:"required,min=2,max=50"`
	Name         string `json:"name" binding:"required,min=2,max=255"`
	LdapID       string `json:"-"`
}

func (u *UserGroupCreateDto) Normalize() {
	u.FriendlyName = norm.NFC.String(u.FriendlyName)
	u.Name = norm.NFC.String(u.Name)
}

type UserGroupUpdateUsersDto struct {
	UserIDs []string `json:"userIds" binding:"required"`
}
