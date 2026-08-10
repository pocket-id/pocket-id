package scimsync

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type ServiceProvider struct {
	model.Base

	Endpoint     string `sortable:"true"`
	Token        datatype.EncryptedString
	LastSyncedAt *datatype.DateTime `sortable:"true"`

	OidcClientID string
	OidcClient   model.OidcClient `gorm:"foreignKey:OidcClientID;references:ID;"`
}

func (ServiceProvider) TableName() string {
	return "scim_service_providers"
}
