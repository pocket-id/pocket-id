package usersignup

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// SignupToken is a single- or limited-use token that grants the ability to self-register
type SignupToken struct {
	model.Base

	// Token is only set when the token is created, since only its hash is stored
	Token      string            `json:"token"`
	Prefix     string            `json:"prefix"`
	ExpiresAt  datatype.DateTime `json:"expiresAt" sortable:"true"`
	UsageLimit int               `json:"usageLimit" sortable:"true"`
	UsageCount int               `json:"usageCount" sortable:"true"`
	UserGroups []model.UserGroup `json:"userGroups"`
	SentTo     string            `json:"sentTo"`
}
