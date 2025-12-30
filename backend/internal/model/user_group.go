package model

import (
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type UserGroup struct {
	Base
	FriendlyName       string `sortable:"true"`
	Name               string `sortable:"true"`
	LdapID             *string
	UpdatedAt          *datatype.DateTime
	Users              []User `gorm:"many2many:user_groups_users;"`
	CustomClaims       []CustomClaim
	AllowedOidcClients []OidcClient `gorm:"many2many:oidc_clients_allowed_user_groups;"`
}

func (ug UserGroup) LastModified() time.Time {
	if ug.UpdatedAt != nil {
		return ug.UpdatedAt.ToTime()
	}
	return ug.CreatedAt.ToTime()
}
