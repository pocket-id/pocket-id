package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type WebauthnCredential struct {
	Base

	Name            string
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transport       AuthenticatorTransportList

	BackupEligible bool `json:"backupEligible"`
	BackupState    bool `json:"backupState"`

	AAGUID string `gorm:"column:aaguid;default:00000000-0000-0000-0000-000000000000"`

	// AttestationVerified is set when the authenticator's make and model were proven through the FIDO Metadata Service at registration
	AttestationVerified bool

	UserID string
}

// HasIcon reports whether an icon is embedded for this credential's authenticator
// It is mapped onto the DTO so the frontend knows upfront whether an icon is worth requesting
func (c WebauthnCredential) HasIcon() bool {
	return utils.HasAuthenticatorIcon(c.AAGUID)
}

// VerifiedAuthenticatorName is the authenticator's model name if its attestation was verified, because the AAGUID of an unverified passkey is only a claim
func (c WebauthnCredential) VerifiedAuthenticatorName() string {
	if !c.AttestationVerified {
		return ""
	}
	return utils.GetAuthenticatorName(c.AAGUID)
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(atl, value)
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}
