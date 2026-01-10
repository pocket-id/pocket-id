package model

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

type EmailVerificationToken struct {
	Base

	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   User
}
