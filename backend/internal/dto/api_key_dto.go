package dto

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type ApiKeyCreateDto struct {
	Name        string            `json:"name" binding:"required,min=3,max=50"`
	Description string            `json:"description"`
	ExpiresAt   datatype.DateTime `json:"expiresAt" binding:"required"`
}

type ApiKeyDto struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	ExpiresAt   datatype.DateTime  `json:"expiresAt"`
	LastUsedAt  *datatype.DateTime `json:"lastUsedAt"`
	CreatedAt   datatype.DateTime  `json:"createdAt"`
}

type ApiKeyResponseDto struct {
	ApiKey ApiKeyDto `json:"apiKey"`
	Token  string    `json:"token"`
}
