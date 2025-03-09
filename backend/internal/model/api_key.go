package model

import (
	"time"
)

type ApiKey struct {
	Base

	Name        string `sortable:"true"`
	Key         string
	Description string
	ExpiresAt   time.Time  `sortable:"true"`
	LastUsedAt  *time.Time `sortable:"true"`

	UserID string
	User   User
}
