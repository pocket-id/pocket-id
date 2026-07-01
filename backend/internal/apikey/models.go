package apikey

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// ApiKey is a personal access token a user can use to authenticate against the API
type ApiKey struct {
	model.Base

	Name                string `sortable:"true"`
	Key                 string
	Description         *string
	ExpiresAt           datatype.DateTime  `sortable:"true"`
	LastUsedAt          *datatype.DateTime `sortable:"true"`
	ExpirationEmailSent bool

	UserID string
	User   model.User
}
