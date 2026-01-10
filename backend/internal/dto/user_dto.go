package dto

import (
	"errors"

	"github.com/gin-gonic/gin/binding"
)

type UserDto struct {
	ID            string                `json:"id"`
	Username      string                `json:"username"`
	Email         *string               `json:"email"`
	EmailVerified bool                  `json:"emailVerified"`
	FirstName     string                `json:"firstName"`
	LastName      *string               `json:"lastName"`
	DisplayName   string                `json:"displayName"`
	IsAdmin       bool                  `json:"isAdmin"`
	Locale        *string               `json:"locale"`
	CustomClaims  []CustomClaimDto      `json:"customClaims"`
	UserGroups    []UserGroupMinimalDto `json:"userGroups"`
	LdapID        *string               `json:"ldapId"`
	Disabled      bool                  `json:"disabled"`
}

type UserCreateDto struct {
	Username      string   `json:"username" binding:"required,username,min=2,max=50" unorm:"nfc"`
	Email         *string  `json:"email" binding:"omitempty,email" unorm:"nfc"`
	EmailVerified bool     `json:"emailVerified"`
	FirstName     string   `json:"firstName" binding:"required,min=1,max=50" unorm:"nfc"`
	LastName      string   `json:"lastName" binding:"max=50" unorm:"nfc"`
	DisplayName   string   `json:"displayName" binding:"required,min=1,max=100" unorm:"nfc"`
	IsAdmin       bool     `json:"isAdmin"`
	Locale        *string  `json:"locale"`
	Disabled      bool     `json:"disabled"`
	UserGroupIds  []string `json:"userGroupIds"`
	LdapID        string   `json:"-"`
}

func (u UserCreateDto) Validate() error {
	e, ok := binding.Validator.Engine().(interface {
		Struct(s any) error
	})
	if !ok {
		return errors.New("validator does not implement the expected interface")
	}

	return e.Struct(u)
}

type EmailVerificationDto struct {
	Token string `json:"token" binding:"required"`
}

type UserUpdateUserGroupDto struct {
	UserGroupIds []string `json:"userGroupIds" binding:"required"`
}
