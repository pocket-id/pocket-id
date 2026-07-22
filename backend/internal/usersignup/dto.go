package usersignup

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type signUpDto struct {
	Username  string  `json:"username" required:"true" minLength:"1" maxLength:"50" pattern:"^[a-zA-Z0-9]([a-zA-Z0-9_.@-]*[a-zA-Z0-9])?$" patternDescription:"letters, numbers, underscores, dots, hyphens, and @ symbols without leading or trailing special characters" unorm:"nfc"`
	Email     *string `json:"email" required:"false" format:"email" unorm:"nfc"`
	FirstName string  `json:"firstName" required:"false" maxLength:"50" unorm:"nfc"`
	LastName  string  `json:"lastName" required:"false" maxLength:"50" unorm:"nfc"`
	Token     string  `json:"token" required:"false"`
}

func (d *signupTokenCreateDto) Resolve(huma.Context) []error {
	if dto.ValidateTTL(d.TTL) {
		return nil
	}
	return []error{&huma.ErrorDetail{Location: "body.ttl", Message: "TTL must be greater than one second and no more than 31 days"}}
}

type signupTokenCreateDto struct {
	TTL          utils.JSONDuration `json:"ttl" required:"true"`
	UsageLimit   int                `json:"usageLimit" required:"true" minimum:"1" maximum:"100"`
	UserGroupIDs []string           `json:"userGroupIds" required:"false"`
}

type signupTokenDto struct {
	ID         string                    `json:"id"`
	Token      string                    `json:"token"`
	ExpiresAt  datatype.DateTime         `json:"expiresAt"`
	UsageLimit int                       `json:"usageLimit"`
	UsageCount int                       `json:"usageCount"`
	UserGroups []dto.UserGroupMinimalDto `json:"userGroups"`
	CreatedAt  datatype.DateTime         `json:"createdAt"`
}
