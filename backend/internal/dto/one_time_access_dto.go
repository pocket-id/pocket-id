package dto

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type OneTimeAccessTokenCreateDto struct {
	TTL utils.JSONDuration `json:"ttl" required:"false"`
}

type OneTimeAccessEmailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" required:"true" format:"email" unorm:"nfc"`
	RedirectPath string `json:"redirectPath" required:"false"`
}

type OneTimeAccessEmailAsAdminDto struct {
	TTL utils.JSONDuration `json:"ttl" required:"false"`
}

func (d *OneTimeAccessTokenCreateDto) Resolve(huma.Context) []error {
	return resolveTTL(d.TTL)
}

func (d *OneTimeAccessEmailAsAdminDto) Resolve(huma.Context) []error {
	return resolveTTL(d.TTL)
}

func resolveTTL(ttl utils.JSONDuration) []error {
	if ValidateTTL(ttl) {
		return nil
	}
	return []error{&huma.ErrorDetail{Location: "body.ttl", Message: "TTL must be greater than one second and no more than 31 days"}}
}
