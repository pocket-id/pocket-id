package dto

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type SignupTokenCreateDto struct {
	TTL          utils.JSONDuration `json:"ttl" binding:"required,ttl"`
	UsageLimit   int                `json:"usageLimit" binding:"required,min=1,max=100"`
	UserGroupIDs []string           `json:"userGroupIds"`
}

type SignupTokenDto struct {
	ID         string                `json:"id"`
	Token      string                `json:"token"`
	ExpiresAt  datatype.DateTime     `json:"expiresAt"`
	UsageLimit int                   `json:"usageLimit"`
	UsageCount int                   `json:"usageCount"`
	UserGroups []UserGroupMinimalDto `json:"userGroups"`
	CreatedAt  datatype.DateTime     `json:"createdAt"`
}
