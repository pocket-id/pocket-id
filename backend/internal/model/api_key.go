package model

import (
	"github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type ApiKey struct {
	Base

	Name        string `sortable:"true"`
	Key         string
	Description *string
	ExpiresAt   datatype.DateTime  `sortable:"true"`
	LastUsedAt  *datatype.DateTime `sortable:"true"`

	UserID string
	User   User
}
