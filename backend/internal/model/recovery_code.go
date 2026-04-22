package model

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type RecoveryCode struct {
	Base

	CodeHash string
	UsedAt   *datatype.DateTime

	UserID string
	User   User
}
