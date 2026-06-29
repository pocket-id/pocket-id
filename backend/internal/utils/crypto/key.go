package crypto

import (
	"crypto/hmac"
	"crypto/sha3"
	"errors"
	"fmt"
	"hash"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// DeriveKey derives a key using HMAC from the configured EncryptionKey and a seed
func DeriveKey(envConfig *common.EnvConfigSchema, seed string) (key []byte, err error) {
	if len(envConfig.EncryptionKey) == 0 {
		return nil, errors.New("encryption key is empty in the configuration")
	}

	// We use HMAC with SHA3-256 here to derive a 256-bit key from the one passed as input
	h := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(envConfig.EncryptionKey))
	fmt.Fprint(h, seed)
	key = h.Sum(nil)

	return key, nil
}
