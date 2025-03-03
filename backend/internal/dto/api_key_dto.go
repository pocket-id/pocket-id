package dto

import "time"

type ApiKeyCreateDto struct {
	Name        string    `json:"name" binding:"required,min=3,max=50"`
	Description string    `json:"description"`
	ExpiresAt   time.Time `json:"expiresAt" binding:"required,gtfield=time.Now"`
}

type ApiKeyDto struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ApiKeyResponseDto struct {
	ApiKey ApiKeyDto `json:"apiKey"`
	Token  string    `json:"token"`
}
