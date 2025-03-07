package model

import (
	"time"
)

type ApiKey struct {
	Base

	Name        string
	Key         string
	Description string
	Enabled     bool
	ExpiresAt   time.Time
	LastUsedAt  *time.Time
	UserID      string
	User        User `gorm:"foreignKey:UserID"`
}
