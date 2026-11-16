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
	Email        string             `json:"email" binding:"omitempty,email" unorm:"nfc"`
}

type signupTokenDto struct {
	ID         string                    `json:"id"`
	Token      string                    `json:"token,omitempty"`
	Prefix     string                    `json:"prefix"`
	ExpiresAt  datatype.DateTime         `json:"expiresAt"`
	UsageLimit int                       `json:"usageLimit"`
	UsageCount int                       `json:"usageCount"`
	UserGroups []dto.UserGroupMinimalDto `json:"userGroups"`
	SentTo     string                    `json:"sentTo,omitempty"`
	CreatedAt  datatype.DateTime         `json:"createdAt"`
}
