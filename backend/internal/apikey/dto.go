package apikey

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type apiKeyCreateDto struct {
	Name        string            `json:"name" required:"true" minLength:"3" maxLength:"50" unorm:"nfc"`
	Description *string           `json:"description" required:"false" unorm:"nfc"`
	ExpiresAt   datatype.DateTime `json:"expiresAt" required:"true"`
}

type apiKeyRenewDto struct {
	ExpiresAt datatype.DateTime `json:"expiresAt" required:"true"`
}

type apiKeyDto struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Description         *string            `json:"description"`
	ExpiresAt           datatype.DateTime  `json:"expiresAt"`
	LastUsedAt          *datatype.DateTime `json:"lastUsedAt"`
	CreatedAt           datatype.DateTime  `json:"createdAt"`
	ExpirationEmailSent bool               `json:"expirationEmailSent"`
}

type apiKeyResponseDto struct {
	ApiKey apiKeyDto `json:"apiKey"`
	Token  string    `json:"token"`
}
