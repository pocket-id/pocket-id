package webauthn

import (
	"github.com/go-webauthn/webauthn/protocol"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type WebauthnCredentialDto struct {
	ID              string                            `json:"id"`
	Name            string                            `json:"name"`
	CredentialID    string                            `json:"credentialID"`
	AttestationType string                            `json:"attestationType"`
	Transport       []protocol.AuthenticatorTransport `json:"transport"`

	BackupEligible bool `json:"backupEligible"`
	BackupState    bool `json:"backupState"`

	AAGUID              string `json:"aaguid"`
	HasIcon             bool   `json:"hasIcon"`
	AttestationVerified bool   `json:"attestationVerified"`
	// VerifiedAuthenticatorName is empty unless the attestation was verified and the AAGUID is known
	VerifiedAuthenticatorName string `json:"verifiedAuthenticatorName,omitempty"`

	CreatedAt datatype.DateTime `json:"createdAt"`
}

type webauthnCredentialUpdateDto struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
}
