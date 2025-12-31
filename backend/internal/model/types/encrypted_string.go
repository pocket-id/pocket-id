package datatype

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

const encryptedStringAAD = "encrypted_string"

// EncryptedString stores plaintext in memory and persists encrypted data in the database.
type EncryptedString string //nolint:recvcheck

func (e *EncryptedString) Scan(value any) error {
	if value == nil {
		*e = ""
		return nil
	}

	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unexpected type for EncryptedString: %T", value)
	}

	encBytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted string: %w", err)
	}

	decBytes, err := cryptoutils.Decrypt(common.EnvConfig.EncryptionKey, encBytes, []byte(encryptedStringAAD))
	if err != nil {
		return fmt.Errorf("failed to decrypt encrypted string: %w", err)
	}

	*e = EncryptedString(decBytes)
	return nil
}

func (e EncryptedString) Value() (driver.Value, error) {
	if e == "" {
		return "", nil
	}

	encBytes, err := cryptoutils.Encrypt(common.EnvConfig.EncryptionKey, []byte(e), []byte(encryptedStringAAD))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt string: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encBytes), nil
}

func (e EncryptedString) String() string {
	return string(e)
}
