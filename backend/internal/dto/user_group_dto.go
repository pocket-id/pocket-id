package dto

import (
	"errors"

	"github.com/gin-gonic/gin/binding"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type UserGroupDto struct {
	ID             string            `json:"id"`
	FriendlyName   string            `json:"friendlyName"`
	Name           string            `json:"name"`
	CustomClaims   []CustomClaimDto  `json:"customClaims"`
	LdapAttributes []CustomClaimDto  `json:"ldapAttributes"`
	LdapID         *string           `json:"ldapId"`
	CreatedAt      datatype.DateTime `json:"createdAt"`
}

type UserGroupDtoWithUsers struct {
	ID             string            `json:"id"`
	FriendlyName   string            `json:"friendlyName"`
	Name           string            `json:"name"`
	CustomClaims   []CustomClaimDto  `json:"customClaims"`
	LdapAttributes []CustomClaimDto  `json:"ldapAttributes"`
	Users          []UserDto         `json:"users"`
	LdapID         *string           `json:"ldapId"`
	CreatedAt      datatype.DateTime `json:"createdAt"`
}

type UserGroupDtoWithUserCount struct {
	ID             string            `json:"id"`
	FriendlyName   string            `json:"friendlyName"`
	Name           string            `json:"name"`
	CustomClaims   []CustomClaimDto  `json:"customClaims"`
	LdapAttributes []CustomClaimDto  `json:"ldapAttributes"`
	UserCount      int64             `json:"userCount"`
	LdapID         *string           `json:"ldapId"`
	CreatedAt      datatype.DateTime `json:"createdAt"`
}

type UserGroupCreateDto struct {
	FriendlyName string `json:"friendlyName" binding:"required,min=2,max=50" unorm:"nfc"`
	Name         string `json:"name" binding:"required,min=2,max=255" unorm:"nfc"`
	LdapID       string `json:"-"`
}

func (g UserGroupCreateDto) Validate() error {
	e, ok := binding.Validator.Engine().(interface {
		Struct(s any) error
	})
	if !ok {
		return errors.New("validator does not implement the expected interface")
	}

	return e.Struct(g)
}

type UserGroupUpdateUsersDto struct {
	UserIDs []string `json:"userIds" binding:"required"`
}
