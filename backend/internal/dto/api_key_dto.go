package dto

import (
	"golang.org/x/text/unicode/norm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type ApiKeyCreateDto struct {
	Name        string            `json:"name" binding:"required,min=3,max=50"`
	Description string            `json:"description"`
	ExpiresAt   datatype.DateTime `json:"expiresAt" binding:"required"`
}

func (a *ApiKeyCreateDto) Normalize() {
	a.Name = norm.NFC.String(a.Name)
	a.Description = norm.NFC.String(a.Description)
}

type ApiKeyDto struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Description         string             `json:"description"`
	ExpiresAt           datatype.DateTime  `json:"expiresAt"`
	LastUsedAt          *datatype.DateTime `json:"lastUsedAt"`
	CreatedAt           datatype.DateTime  `json:"createdAt"`
	ExpirationEmailSent bool               `json:"expirationEmailSent"`
}

type ApiKeyResponseDto struct {
	ApiKey ApiKeyDto `json:"apiKey"`
	Token  string    `json:"token"`
}
