package model

import (
	"strings"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type User struct {
	Base

	Username                  string  `sortable:"case-insensitive"`
	Email                     *string `sortable:"case-insensitive"`
	EmailVerified             bool    `sortable:"true" filterable:"true"`
	EmailVerificationDeadline *datatype.DateTime
	FirstName                 string `sortable:"case-insensitive"`
	LastName                  string `sortable:"case-insensitive"`
	DisplayName               string `sortable:"case-insensitive"`
	IsAdmin                   bool   `sortable:"true" filterable:"true"`
	Locale                    *string
	LdapID                    *string
	Disabled                  bool `sortable:"true" filterable:"true"`
	UpdatedAt                 *datatype.DateTime

	CustomClaims []CustomClaim
	UserGroups   []UserGroup `gorm:"many2many:user_groups_users;"`
}

func (u User) FullName() string {
	fullname := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if fullname != "" {
		return fullname
	}

	if u.DisplayName != "" {
		return u.DisplayName
	}

	return u.Username
}
