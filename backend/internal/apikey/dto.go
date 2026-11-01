package apikey

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type apiKeyCreateDto struct {
	Name        string              `json:"name" binding:"required,min=3,max=50" unorm:"nfc"`
	Description *string             `json:"description" unorm:"nfc"`
	ExpiresAt   datatype.DateTime   `json:"expiresAt" binding:"required"`
	Scopes      datatype.StringList `json:"scopes" binding:"required,min=1"`
}

type apiKeyUpdateDto struct {
	Name        string              `json:"name" binding:"required,min=3,max=50" unorm:"nfc"`
	Description *string             `json:"description" unorm:"nfc"`
	Scopes      datatype.StringList `json:"scopes" binding:"required,min=1"`
}

type apiKeyRenewDto struct {
	ExpiresAt datatype.DateTime `json:"expiresAt" binding:"required"`
}

type apiKeyDto struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         *string             `json:"description"`
	ExpiresAt           datatype.DateTime   `json:"expiresAt"`
	LastUsedAt          *datatype.DateTime  `json:"lastUsedAt"`
	CreatedAt           datatype.DateTime   `json:"createdAt"`
	ExpirationEmailSent bool                `json:"expirationEmailSent"`
	Scopes              datatype.StringList `json:"scopes"`
}

type apiKeyResponseDto struct {
	ApiKey apiKeyDto `json:"apiKey"`
	Token  string    `json:"token"`
}
