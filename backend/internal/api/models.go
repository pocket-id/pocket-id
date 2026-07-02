package api

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type API struct {
	model.Base

	Name      string `sortable:"true"`
	Audience  string `sortable:"true"`
	UpdatedAt *datatype.DateTime

	Permissions []Permission `gorm:"foreignKey:APIID;references:ID;constraint:OnDelete:CASCADE"`
}

type Permission struct {
	model.Base

	APIID       string `gorm:"column:api_id"`
	Key         string `sortable:"true"`
	Name        string
	Description *string
}

func (Permission) TableName() string { return "api_permissions" }

type OidcClientAllowedAPIPermission struct {
	OidcClientID    string `gorm:"column:oidc_client_id;primaryKey"`
	APIPermissionID string `gorm:"column:api_permission_id;primaryKey"`
}

func (OidcClientAllowedAPIPermission) TableName() string {
	return "oidc_clients_allowed_api_permissions"
}
