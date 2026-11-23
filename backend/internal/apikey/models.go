package apikey

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// ApiKey is a personal access token a user can use to authenticate against the API
type ApiKey struct {
	model.Base

	Name                string `sortable:"case-insensitive"`
	Key                 string
	Description         *string
	ExpiresAt           datatype.DateTime  `sortable:"true"`
	LastUsedAt          *datatype.DateTime `sortable:"true"`
	ExpirationEmailSent bool
	Scopes              datatype.StringList

	// PreviousKey is the hash of the token a rotation replaced, which keeps working until PreviousKeyExpiresAt
	PreviousKey          *string
	PreviousKeyExpiresAt *datatype.DateTime

	UserID string
	User   model.User
}
