package onetimeaccess

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type tokenCreateDto struct {
	TTL utils.JSONDuration `json:"ttl" required:"false"`
}

type emailAsUnauthenticatedUserDto struct {
	Email        string `json:"email" required:"true" format:"email" unorm:"nfc"`
	RedirectPath string `json:"redirectPath" required:"false"`
}

type emailAsAdminDto struct {
	TTL utils.JSONDuration `json:"ttl" required:"false"`
}

func (d *tokenCreateDto) Resolve(huma.Context) []error {
	return resolveTTL(d.TTL)
}

func (d *emailAsAdminDto) Resolve(huma.Context) []error {
	return resolveTTL(d.TTL)
}

func resolveTTL(ttl utils.JSONDuration) []error {
	if dto.ValidateTTL(ttl) {
		return nil
	}
	return []error{&huma.ErrorDetail{Location: "body.ttl", Message: "TTL must be greater than one second and no more than 31 days"}}
}
