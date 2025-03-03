package model

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type ApiKey struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Key         string `gorm:"not null;uniqueIndex"`
	Description string
	Enabled     bool `gorm:"not null;default:true"`
	ExpiresAt   time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
	UserID      string
	User        User `gorm:"foreignKey:UserID"`
}

func (m *ApiKey) BeforeCreate(tx *gorm.DB) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	m.ID = id.String()
	return nil
}
