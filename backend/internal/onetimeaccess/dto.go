package onetimeaccess

import "github.com/pocket-id/pocket-id/backend/internal/utils"

type tokenCreateDto struct {
	TTL utils.JSONDuration `json:"ttl" binding:"ttl"`
}

type emailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" binding:"required,email" unorm:"nfc"`
	RedirectPath string `json:"redirectPath"`
}

type emailAsAdminDto struct {
	TTL utils.JSONDuration `json:"ttl" binding:"ttl"`
}
