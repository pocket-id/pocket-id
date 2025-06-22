package dto

import (
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type SignupTokenCreateDto struct {
	ExpiresAt  time.Time `json:"expiresAt" binding:"required"`
	UsageLimit int       `json:"usageLimit" binding:"required,min=1,max=100"`
}

type SignupTokenDto struct {
	ID         string            `json:"id"`
	Token      string            `json:"token"`
	ExpiresAt  datatype.DateTime `json:"expiresAt"`
	UsageLimit int               `json:"usageLimit"`
	UsageCount int               `json:"usageCount"`
	CreatedAt  datatype.DateTime `json:"createdAt"`
}

type SignupWithTokenDto struct {
	Token    string        `json:"token" binding:"required"`
	UserData UserCreateDto `json:"userData" binding:"required"`
}
