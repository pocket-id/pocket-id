package webauthn

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// WebauthnSession holds the transient challenge state for an in-progress registration or login
type WebauthnSession struct {
	model.Base

	Challenge        string
	ExpiresAt        datatype.DateTime
	UserVerification string
	CredentialParams CredentialParameters
	Extensions       protocol.SessionExtensions `gorm:"serializer:json"`
}

type WebauthnCredential struct {
	model.Base

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
	return hasAuthenticatorIcon(c.AAGUID)
}

// VerifiedAuthenticatorName is the authenticator's model name if its attestation was verified, because the AAGUID of an unverified passkey is only a claim
func (c WebauthnCredential) VerifiedAuthenticatorName() string {
	if !c.AttestationVerified {
		return ""
	}
	return getAuthenticatorName(c.AAGUID)
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(atl, value)
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}

// passkeyUser presents a user together with their passkeys to go-webauthn
type passkeyUser struct {
	model.User
	credentials []WebauthnCredential
}

func (u *passkeyUser) WebAuthnID() []byte { return []byte(u.ID) }

func (u *passkeyUser) WebAuthnName() string { return u.Username }

func (u *passkeyUser) WebAuthnDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.FullName()
}

func (u *passkeyUser) WebAuthnCredentials() []gowebauthn.Credential {
	credentials := make([]gowebauthn.Credential, len(u.credentials))
	for i, credential := range u.credentials {
		credentials[i] = gowebauthn.Credential{
			ID:              credential.CredentialID,
			AttestationType: credential.AttestationType,
			PublicKey:       credential.PublicKey,
			Transport:       credential.Transport,
			Flags: gowebauthn.CredentialFlags{
				BackupState:    credential.BackupState,
				BackupEligible: credential.BackupEligible,
			},
		}
	}
	return credentials
}

// ReauthenticationToken is a short-lived token proving a user recently re-verified themselves
type ReauthenticationToken struct {
	model.Base
	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   model.User
}

// PublicKeyCredentialCreationOptions is the registration challenge returned to the browser
type PublicKeyCredentialCreationOptions struct {
	Response  protocol.PublicKeyCredentialCreationOptions
	SessionID string
	Timeout   time.Duration
}

// PublicKeyCredentialRequestOptions is the login challenge returned to the browser
type PublicKeyCredentialRequestOptions struct {
	Response  protocol.PublicKeyCredentialRequestOptions
	SessionID string
	Timeout   time.Duration
}

type CredentialParameters []protocol.CredentialParameter //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (cp *CredentialParameters) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(cp, value)
}

func (cp CredentialParameters) Value() (driver.Value, error) {
	return json.Marshal(cp)
}
