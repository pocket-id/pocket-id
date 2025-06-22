package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type SignupToken struct {
	ID         string            `gorm:"primaryKey" json:"id"`
	CreatedAt  datatype.DateTime `gorm:"not null" json:"createdAt"`
	Token      string            `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt  datatype.DateTime `gorm:"not null" json:"expiresAt"`
	UsageLimit int               `gorm:"not null;default:1" json:"usageLimit"`
	UsageCount int               `gorm:"not null;default:0" json:"usageCount"`
}

func (st *SignupToken) BeforeCreate(tx *gorm.DB) error {
	if st.ID == "" {
		st.ID = uuid.New().String()
	}
	st.CreatedAt = datatype.DateTime(time.Now())
	return nil
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
