package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type WebauthnSession struct {
	Base

	Challenge        string
	ExpiresAt        datatype.DateTime
	UserVerification string
	CredentialParams CredentialParameters
}

type WebauthnCredential struct {
	Base

	Name            string
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transport       AuthenticatorTransportList

	BackupEligible bool `json:"backupEligible"`
	BackupState    bool `json:"backupState"`

	UserID string
}

type PublicKeyCredentialCreationOptions struct {
	Response  protocol.PublicKeyCredentialCreationOptions
	SessionID string
	Timeout   time.Duration
}

type PublicKeyCredentialRequestOptions struct {
	Response  protocol.PublicKeyCredentialRequestOptions
	SessionID string
	Timeout   time.Duration
}

type ReauthenticationToken struct {
	Base
	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   User
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(atl, value)
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}

type CredentialParameters []protocol.CredentialParameter //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (cp *CredentialParameters) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(cp, value)
}

func (cp CredentialParameters) Value() (driver.Value, error) {
	return json.Marshal(cp)
}
