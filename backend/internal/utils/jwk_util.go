package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

const (
	// KeyUsageSigning is the usage for the private keys, for the "use" property
	KeyUsageSigning = "sig"
)

// ImportRawKey imports a crypto key in "raw" format (e.g. crypto.PrivateKey) into a jwk.Key.
// It also populates additional fields such as the key ID, usage, and alg.
func ImportRawKey(rawKey any) (jwk.Key, error) {
	key, err := jwk.Import(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to import generated private key: %w", err)
	}

	// Generate the key ID
	kid, err := generateRandomKeyID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	_ = key.Set(jwk.KeyIDKey, kid)

	// Set other required fields
	_ = key.Set(jwk.KeyUsageKey, KeyUsageSigning)
	EnsureAlgInKey(key)

	return key, nil
}

// generateRandomKeyID generates a random key ID.
func generateRandomKeyID() (string, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// EnsureAlgInKey ensures that the key contains an "alg" parameter, set depending on the key type
func EnsureAlgInKey(key jwk.Key) {
	_, ok := key.Algorithm()
	if ok {
		// Algorithm is already set
		return
	}

	switch key.KeyType() {
	case jwa.RSA():
		// Default to RS256 for RSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.RS256())
	case jwa.EC():
		// Default to ES256 for ECDSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.ES256())
	case jwa.OKP():
		// Default to EdDSA for OKP keys
		_ = key.Set(jwk.AlgorithmKey, jwa.EdDSA())
	}
}
