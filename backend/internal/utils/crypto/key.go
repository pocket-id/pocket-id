package crypto

import (
	"crypto/hmac"
	"crypto/sha3"
	"errors"
	"fmt"
	"hash"
)

// DeriveKey derives a key using HMAC from the configured masterKey (envConfig.EncryptionKey) and a seed
// Note: changing this function is considered a breaking change
func DeriveKey(masterKey []byte, seed string) (key []byte, err error) {
	if len(masterKey) == 0 {
		return nil, errors.New("encryption key is empty in the configuration")
	}

	// We use HMAC with SHA3-256 here to derive a 256-bit key from the one passed as input
	h := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(masterKey))
	fmt.Fprint(h, seed)
	key = h.Sum(nil)

	return key, nil
}
