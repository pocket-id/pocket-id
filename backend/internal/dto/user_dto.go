package dto

import (
	"time"

	"golang.org/x/text/unicode/norm"
)

type UserDto struct {
	ID           string           `json:"id"`
	Username     string           `json:"username"`
	Email        string           `json:"email" `
	FirstName    string           `json:"firstName"`
	LastName     string           `json:"lastName"`
	IsAdmin      bool             `json:"isAdmin"`
	Locale       *string          `json:"locale"`
	CustomClaims []CustomClaimDto `json:"customClaims"`
	UserGroups   []UserGroupDto   `json:"userGroups"`
	LdapID       *string          `json:"ldapId"`
	Disabled     bool             `json:"disabled"`
}

type UserCreateDto struct {
	Username  string  `json:"username" binding:"required,username,min=2,max=50"`
	Email     string  `json:"email" binding:"required,email"`
	FirstName string  `json:"firstName" binding:"required,min=1,max=50"`
	LastName  string  `json:"lastName" binding:"max=50"`
	IsAdmin   bool    `json:"isAdmin"`
	Locale    *string `json:"locale"`
	Disabled  bool    `json:"disabled"`
	LdapID    string  `json:"-"`
}

func (u *UserCreateDto) Normalize() {
	u.Username = norm.NFC.String(u.Username)
	u.Email = norm.NFC.String(u.Email)
	u.FirstName = norm.NFC.String(u.FirstName)
	u.LastName = norm.NFC.String(u.LastName)
}

type OneTimeAccessTokenCreateDto struct {
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt" binding:"required"`
}

type OneTimeAccessEmailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" binding:"required,email"`
	RedirectPath string `json:"redirectPath"`
}

func (o *OneTimeAccessEmailAsUnauthenticatedUserDto) Normalize() {
	o.Email = norm.NFC.String(o.Email)
}

type OneTimeAccessEmailAsAdminDto struct {
	ExpiresAt time.Time `json:"expiresAt" binding:"required"`
}

type UserUpdateUserGroupDto struct {
	UserGroupIds []string `json:"userGroupIds" binding:"required"`
}

type SignUpDto struct {
	Username  string `json:"username" binding:"required,username,min=2,max=50"`
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"firstName" binding:"required,min=1,max=50"`
	LastName  string `json:"lastName" binding:"max=50"`
	Token     string `json:"token"`
}

func (s *SignUpDto) Normalize() {
	s.Username = norm.NFC.String(s.Username)
	s.Email = norm.NFC.String(s.Email)
	s.FirstName = norm.NFC.String(s.FirstName)
	s.LastName = norm.NFC.String(s.LastName)
}
