package model

type UserGroup struct {
	Base
	FriendlyName       string `sortable:"true"`
	Name               string `sortable:"true"`
	LdapID             *string
	Users              []User `gorm:"many2many:user_groups_users;"`
	CustomClaims       []CustomClaim
	AllowedOidcClients []OidcClient `gorm:"many2many:oidc_clients_allowed_user_groups;"`
}
