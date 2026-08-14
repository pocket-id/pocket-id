package model

import datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"

const RuntimeCredentialAlgorithmEd25519 = "Ed25519"

// RuntimeCredential is the FCA02 public authenticator record paired with short-lived proof state that never stores private keys
type RuntimeCredential struct {
	Base

	Name       string
	Algorithm  string
	PublicKey  []byte
	LastUsedAt *datatype.DateTime
	ExpiresAt  *datatype.DateTime
	RevokedAt  *datatype.DateTime

	UserID string
	User   User
}

type RuntimeCredentialChallenge struct {
	Base

	Operation           string
	Challenge           []byte
	ExpiresAt           datatype.DateTime
	RuntimeCredentialID *string
	CredentialName      *string
	Algorithm           *string
	PublicKey           []byte

	UserID string
	User   User
}
