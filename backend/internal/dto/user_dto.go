package dto

import "time"

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
}

type UserCreateDto struct {
	Username  string  `json:"username" binding:"required,username,min=2,max=50"`
	Email     string  `json:"email" binding:"required,email"`
	FirstName string  `json:"firstName" binding:"required,min=1,max=50"`
	LastName  string  `json:"lastName" binding:"required,min=1,max=50"`
	IsAdmin   bool    `json:"isAdmin"`
	Locale    *string `json:"locale"`
	LdapID    string  `json:"-"`
}

type OneTimeAccessTokenCreateDto struct {
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt" binding:"required"`
}

type OneTimeAccessEmailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" binding:"required,email"`
	RedirectPath string `json:"redirectPath"`
}

type OneTimeAccessEmailAsAdminDto struct {
	ExpiresAt time.Time `json:"expiresAt" binding:"required"`
}

type UserUpdateUserGroupDto struct {
	UserGroupIds []string `json:"userGroupIds" binding:"required"`
}
