package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
)

// Base contains common columns for all tables.
type Base struct {
	ID        string            `gorm:"primaryKey;not null"`
	CreatedAt datatype.DateTime `sortable:"true"`
}

func (b *Base) BeforeCreate(_ *gorm.DB) (err error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	b.CreatedAt = datatype.DateTime(time.Now())
	return
}
