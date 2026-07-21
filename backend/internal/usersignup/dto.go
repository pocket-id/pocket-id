package usersignup

import (
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type signUpDto struct {
	Username  string  `json:"username" binding:"required,username,min=1,max=50" unorm:"nfc"`
	Email     *string `json:"email" binding:"omitempty,email" unorm:"nfc"`
	FirstName string  `json:"firstName" binding:"max=50" unorm:"nfc"`
	LastName  string  `json:"lastName" binding:"max=50" unorm:"nfc"`
	Token     string  `json:"token"`
}

type signupTokenCreateDto struct {
	TTL          utils.JSONDuration `json:"ttl" binding:"required,ttl"`
	UsageLimit   int                `json:"usageLimit" binding:"required,min=1,max=100"`
	UserGroupIDs []string           `json:"userGroupIds"`
	EmailDomain  *string            `json:"emailDomain"`
}

type signupTokenDto struct {
	ID          string                    `json:"id"`
	Token       string                    `json:"token"`
	ExpiresAt   datatype.DateTime         `json:"expiresAt"`
	UsageLimit  int                       `json:"usageLimit"`
	UsageCount  int                       `json:"usageCount"`
	EmailDomain *string                   `json:"emailDomain" binding:"omitempty,email_domain"`
	UserGroups  []dto.UserGroupMinimalDto `json:"userGroups"`
	CreatedAt   datatype.DateTime         `json:"createdAt"`
}

// signupTokenInfoDto exposes the limited, publicly readable metadata of a signup token
type signupTokenInfoDto struct {
	EmailDomain *string `json:"emailDomain"`
}
