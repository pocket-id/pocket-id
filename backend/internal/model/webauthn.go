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

	UserID string
}

func (c WebauthnCredential) HasIcon() bool {
	return utils.HasAuthenticatorIcon(c.AAGUID)
}

type AuthenticatorTransportList []protocol.AuthenticatorTransport //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (atl *AuthenticatorTransportList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(atl, value)
}

func (atl AuthenticatorTransportList) Value() (driver.Value, error) {
	return json.Marshal(atl)
}
