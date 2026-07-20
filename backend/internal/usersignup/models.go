package usersignup

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// SignupToken is a single- or limited-use token that grants the ability to self-register.
//
// Signup tokens are managed at runtime by the singleton signup token actor (see signuptoken_actor.go); this struct is used both as the carrier for API responses and to read the pre-actor tokens still stored in the database when migrating them into the actor's state on first startup.
type SignupToken struct {
	model.Base

	Token      string            `json:"token"`
	ExpiresAt  datatype.DateTime `json:"expiresAt" sortable:"true"`
	UsageLimit int               `json:"usageLimit" sortable:"true"`
	UsageCount int               `json:"usageCount" sortable:"true"`
	UserGroups []model.UserGroup `gorm:"many2many:signup_tokens_user_groups;"`
}
