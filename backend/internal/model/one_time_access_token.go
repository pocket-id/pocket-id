package model

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type OneTimeAccessToken struct {
	Base
	Token       string
	DeviceToken *string
	ExpiresAt   datatype.DateTime

	UserID string
	User   User
}
