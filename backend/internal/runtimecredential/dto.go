package runtimecredential

import (
	dto "github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type registrationStartDto struct {
	Token     string `json:"token" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=50"`
	Algorithm string `json:"algorithm" binding:"required"`
	PublicKey string `json:"publicKey" binding:"required"`
}

type loginStartDto struct {
	Username     string `json:"username" binding:"required,username,min=1,max=50"`
	CredentialID string `json:"credentialId" binding:"required,uuid"`
}

type reauthenticationStartDto struct {
	CredentialID string `json:"credentialId" binding:"required,uuid"`
}

type proofFinishDto struct {
	SessionID string `json:"sessionId" binding:"required,uuid"`
	Signature string `json:"signature" binding:"required"`
}

type runtimeCredentialUpdateDto struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
}

type challengeDto struct {
	SessionID string            `json:"sessionId"`
	Challenge string            `json:"challenge"`
	ExpiresAt datatype.DateTime `json:"expiresAt"`
}

type RuntimeCredentialDto struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Algorithm  string             `json:"algorithm"`
	CreatedAt  datatype.DateTime  `json:"createdAt"`
	LastUsedAt *datatype.DateTime `json:"lastUsedAt"`
	ExpiresAt  *datatype.DateTime `json:"expiresAt"`
	RevokedAt  *datatype.DateTime `json:"revokedAt"`
}

type registrationFinishDto struct {
	User       dto.UserDto          `json:"user"`
	Credential RuntimeCredentialDto `json:"credential"`
}
