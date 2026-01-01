package datatype

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
	"golang.org/x/crypto/hkdf"
)

const encryptedStringAAD = "encrypted_string"

var encStringKey []byte

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

	decBytes, err := cryptoutils.Decrypt(encStringKey, encBytes, []byte(encryptedStringAAD))
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

	encBytes, err := cryptoutils.Encrypt(encStringKey, []byte(e), []byte(encryptedStringAAD))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt string: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encBytes), nil
}

func (e EncryptedString) String() string {
	return string(e)
}

func deriveEncryptedStringKey(master []byte) ([]byte, error) {
	const info = "pocketid/encrypted_string"
	r := hkdf.New(sha256.New, master, nil, []byte(info))

	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func init() {
	key, err := deriveEncryptedStringKey(common.EnvConfig.EncryptionKey)
	if err != nil {
		panic(fmt.Sprintf("failed to derive encrypted string key: %v", err))
	}
	encStringKey = key
}
