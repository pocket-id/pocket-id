package model

import (
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type SignupToken struct {
	Base

	Token      string            `json:"token"`
	ExpiresAt  datatype.DateTime `json:"expiresAt" sortable:"true"`
	UsageLimit int               `json:"usageLimit" sortable:"true"`
	UsageCount int               `json:"usageCount" sortable:"true"`
	UserGroups []UserGroup       `gorm:"many2many:signup_tokens_user_groups;"`
}

func (st *SignupToken) IsExpired() bool {
	return time.Time(st.ExpiresAt).Before(time.Now())
}

func (st *SignupToken) IsUsageLimitReached() bool {
	return st.UsageCount >= st.UsageLimit
}

func (st *SignupToken) IsValid() bool {
	return !st.IsExpired() && !st.IsUsageLimitReached()
}
