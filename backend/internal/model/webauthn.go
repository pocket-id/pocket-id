package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type WebauthnSession struct {
	Base

	Challenge        string
	ExpiresAt        datatype.DateTime
	UserVerification string
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

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value interface{}) error {

	if v, ok := value.([]byte); ok {
		return json.Unmarshal(v, atl)
	} else {
		return errors.New("type assertion to []byte failed")
	}
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}
