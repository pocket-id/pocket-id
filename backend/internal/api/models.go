package api

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
)

// BuiltInAPIID is the ID of the API that represents Pocket ID itself, which Bootstrap writes on every startup and admins can only grant clients access to
const BuiltInAPIID = "00000000-0000-0000-0000-000000000000"

type API struct {
	model.Base

	Name             string `sortable:"case-insensitive"`
	Audience         string `sortable:"case-insensitive"`
	UpdatedAt        *datatype.DateTime
	AllowCIMDClients bool `gorm:"column:allow_cimd_clients"`

	Permissions []Permission `gorm:"foreignKey:APIID;references:ID;constraint:OnDelete:CASCADE"`
}

type Permission struct {
	model.Base

	APIID                 string `gorm:"column:api_id"`
	Key                   string `sortable:"true"`
	Name                  string
	Description           *string
	AllowedForCIMDClients bool `gorm:"column:allowed_for_cimd_clients"`
}

func (Permission) TableName() string { return "api_permissions" }

type OidcClientAllowedAPI struct {
	OidcClientID string
	APIID        string `gorm:"column:api_id"`
	SubjectType  oidc.SubjectType
}

func (OidcClientAllowedAPI) TableName() string {
	return "oidc_clients_allowed_apis"
}

type OidcClientAllowedAPIPermission struct {
	OidcClientID    string
	APIPermissionID string
	SubjectType     oidc.SubjectType
}

func (OidcClientAllowedAPIPermission) TableName() string {
	return "oidc_clients_allowed_api_permissions"
}
