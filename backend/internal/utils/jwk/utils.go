package jwk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha3"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// EncodeJWK encodes a jwk.Key to a writable stream.
func EncodeJWK(w io.Writer, key jwk.Key) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(key)
}

// EncodeJWKBytes encodes a jwk.Key to a byte slice.
func EncodeJWKBytes(key jwk.Key) ([]byte, error) {
	b := &bytes.Buffer{}
	err := EncodeJWK(b, key)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func LoadKeyEncryptionKey(envConfig *common.EnvConfigSchema, instanceID string) (kek []byte, err error) {
	// Try getting the key from the env var as string
	kekInput := []byte(envConfig.EncryptionKey)

	// If there's nothing in the env, try loading from file
	if len(kekInput) == 0 && envConfig.EncryptionKeyFile != "" {
		kekInput, err = os.ReadFile(envConfig.EncryptionKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file '%s': %w", envConfig.EncryptionKeyFile, err)
		}
	}

	// If there's still no key, return
	if len(kekInput) == 0 {
		return nil, nil
	}

	// We need a 512-bit key for encryption with AEAD_AES_256_CBC_HMAC_SHA_512: 256 for the content encryption key, and 256 for the HMAC
	// We use HMAC with SHA3-512 here to derive the key from the one passed as input
	// The key is tied to a specific instance of Pocket ID
	h := hmac.New(func() hash.Hash { return sha3.New256() }, kekInput)
	fmt.Fprint(h, "pocketid/"+instanceID+"/jwt-kek")
	kek = h.Sum(nil)

	return kek, nil
}
