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

	UserID string
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(atl, value)
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}
