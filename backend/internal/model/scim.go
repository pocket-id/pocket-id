package model

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type ScimServiceProvider struct {
	Base

	Endpoint     string `sortable:"true"`
	Token        datatype.EncryptedString
	LastSyncedAt *datatype.DateTime `sortable:"true"`

	OidcClientID string
	OidcClient   OidcClient `gorm:"foreignKey:OidcClientID;references:ID;"`
}
