package dto

import (
	"errors"
	"net/mail"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
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
	Username      string   `json:"username" required:"true" minLength:"1" maxLength:"50" pattern:"^[a-zA-Z0-9]([a-zA-Z0-9_.@-]*[a-zA-Z0-9])?$" patternDescription:"letters, numbers, underscores, dots, hyphens, and @ symbols without leading or trailing special characters" unorm:"nfc"`
	Email         *string  `json:"email" required:"false" format:"email" unorm:"nfc"`
	EmailVerified bool     `json:"emailVerified" required:"false"`
	FirstName     string   `json:"firstName" required:"false" maxLength:"50" unorm:"nfc"`
	LastName      string   `json:"lastName" required:"false" maxLength:"50" unorm:"nfc"`
	DisplayName   string   `json:"displayName" required:"false" maxLength:"100" unorm:"nfc"`
	IsAdmin       bool     `json:"isAdmin" required:"false"`
	Locale        *string  `json:"locale" required:"false"`
	Disabled      bool     `json:"disabled" required:"false"`
	UserGroupIds  []string `json:"userGroupIds" required:"false"`
	LdapID        string   `json:"-"`
}

func (u UserCreateDto) Resolve(huma.Context) []error {
	if u.Email == nil {
		return nil
	}
	address, err := mail.ParseAddress(*u.Email)
	if err != nil || address.Address != *u.Email {
		return []error{&huma.ErrorDetail{Location: "body.email", Message: "Field validation for 'Email' failed on the 'email' tag"}}
	}
	return nil
}

//nolint:staticcheck // LDAP callers and their tests rely on the existing capitalized validation text
func (u UserCreateDto) Validate() error {
	if u.Username == "" {
		return errors.New("Field validation for 'Username' failed on the 'required' tag")
	}
	if !ValidateUsername(u.Username) {
		return errors.New("Field validation for 'Username' failed on the 'username' tag")
	}
	if utf8.RuneCountInString(u.Username) > 50 {
		return errors.New("Field validation for 'Username' failed on the 'max' tag")
	}
	if utf8.RuneCountInString(u.FirstName) > 50 {
		return errors.New("Field validation for 'FirstName' failed on the 'max' tag")
	}
	if utf8.RuneCountInString(u.LastName) > 50 {
		return errors.New("Field validation for 'LastName' failed on the 'max' tag")
	}
	if utf8.RuneCountInString(u.DisplayName) > 100 {
		return errors.New("Field validation for 'DisplayName' failed on the 'max' tag")
	}
	return nil
}

type EmailVerificationDto struct {
	Token string `json:"token" required:"true"`
}

type UserUpdateUserGroupDto struct {
	UserGroupIds []string `json:"userGroupIds" required:"true"`
}
