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

	if raw == "" {
		*e = ""
		return nil
	}

	decBytes, err := DecryptEncryptedStringWithKey(encStringKey, raw)
	if err != nil {
		return err
	}

	*e = EncryptedString(decBytes)
	return nil
}

func (e EncryptedString) Value() (driver.Value, error) {
	if e == "" {
		return "", nil
	}

	encValue, err := EncryptEncryptedStringWithKey(encStringKey, []byte(e))
	if err != nil {
		return nil, err
	}

	return encValue, nil
}

func (e EncryptedString) String() string {
	return string(e)
}

// DeriveEncryptedStringKey derives a key for encrypting EncryptedString values from the master key.
func DeriveEncryptedStringKey(master []byte) ([]byte, error) {
	const info = "pocketid/encrypted_string"
	r := hkdf.New(sha256.New, master, nil, []byte(info))

	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// DecryptEncryptedStringWithKey decrypts an EncryptedString value using the derived key.
func DecryptEncryptedStringWithKey(key []byte, encoded string) ([]byte, error) {
	encBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted string: %w", err)
	}

	decBytes, err := cryptoutils.Decrypt(key, encBytes, []byte(encryptedStringAAD))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt encrypted string: %w", err)
	}

	return decBytes, nil
}

// EncryptEncryptedStringWithKey encrypts an EncryptedString value using the derived key.
func EncryptEncryptedStringWithKey(key []byte, plaintext []byte) (string, error) {
	encBytes, err := cryptoutils.Encrypt(key, plaintext, []byte(encryptedStringAAD))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt string: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encBytes), nil
}

func init() {
	key, err := DeriveEncryptedStringKey(common.EnvConfig.EncryptionKey)
	if err != nil {
		panic(fmt.Sprintf("failed to derive encrypted string key: %v", err))
	}
	encStringKey = key
}
