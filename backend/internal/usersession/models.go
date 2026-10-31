package usersession

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// UserSession is one signed-in session, keyed on the jti of the access token it was minted with
type UserSession struct {
	model.Base
	ExpiresAt datatype.DateTime

	UserID string
}
