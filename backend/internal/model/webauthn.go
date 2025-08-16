package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
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

type ReauthenticationToken struct {
	Base
	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   User
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, atl)
	case string:
		return json.Unmarshal([]byte(v), atl)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}
