package dto

import "github.com/pocket-id/pocket-id/backend/internal/utils"

type OneTimeAccessTokenCreateDto struct {
	UserID string             `json:"userId"`
	TTL    utils.JSONDuration `json:"ttl" binding:"ttl"`
}

type OneTimeAccessEmailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" binding:"required,email" unorm:"nfc"`
	RedirectPath string `json:"redirectPath"`
}

type OneTimeAccessEmailAsAdminDto struct {
	TTL utils.JSONDuration `json:"ttl" binding:"ttl"`
}
