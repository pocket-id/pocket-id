package model

import (
	"time"
)

type ApiKey struct {
	Base

	Name        string `sortable:"true"`
	Key         string
	Description string     `sortable:"true"`
	Enabled     bool       `sortable:"true"`
	ExpiresAt   time.Time  `sortable:"true"`
	LastUsedAt  *time.Time `sortable:"true"`
	UserID      string
	User        User `gorm:"foreignKey:UserID"`
}
