package model

type UserGroup struct {
	Base
	FriendlyName string `sortable:"true"`
	Name         string `sortable:"true"`
	LdapID       *string
	Users        []User `gorm:"many2many:user_groups_users;"`
	CustomClaims []CustomClaim
}
