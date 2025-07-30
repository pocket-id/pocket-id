package jwk

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha3"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

const (
	// KeyUsageSigning is the usage for the private keys, for the "use" property
	KeyUsageSigning = "sig"
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

// LoadKeyEncryptionKey loads the key encryption key for JWKs
func LoadKeyEncryptionKey(envConfig *common.EnvConfigSchema, instanceID string) (kek []byte, err error) {
	// If there's no key, return
	if len(envConfig.EncryptionKey) == 0 {
		return nil, nil
	}

	// We need a 256-bit key for encryption with AES-GCM-256
	// We use HMAC with SHA3-256 here to derive the key from the one passed as input
	// The key is tied to a specific instance of Pocket ID
	h := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(envConfig.EncryptionKey))
	fmt.Fprint(h, "pocketid/"+instanceID+"/jwk-kek")
	kek = h.Sum(nil)

	return kek, nil
}

// ImportRawKey imports a crypto key in "raw" format (e.g. crypto.PrivateKey) into a jwk.Key.
// It also populates additional fields such as the key ID, usage, and alg.
func ImportRawKey(rawKey any, alg string, crv string) (jwk.Key, error) {
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
	EnsureAlgInKey(key, alg, crv)

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

// EnsureAlgInKey ensures that the key contains an "alg" parameter (and "crv", if needed), set depending on the key type
func EnsureAlgInKey(key jwk.Key, alg string, crv string) {
	_, ok := key.Algorithm()
	if ok {
		// Algorithm is already set
		return
	}

	if alg != "" {
		_ = key.Set(jwk.AlgorithmKey, alg)
		if crv != "" {
			eca, ok := jwa.LookupEllipticCurveAlgorithm(crv)
			if ok {
				switch key.KeyType() {
				case jwa.EC():
					_ = key.Set(jwk.ECDSACrvKey, eca)
				case jwa.OKP():
					_ = key.Set(jwk.OKPCrvKey, eca)
				}
			}
		}
		return
	}

	// If we don't have an algorithm, set the default for the key type
	switch key.KeyType() {
	case jwa.RSA():
		// Default to RS256 for RSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.RS256())
	case jwa.EC():
		// Default to ES256 for ECDSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.ES256())
		_ = key.Set(jwk.ECDSACrvKey, jwa.P256())
	case jwa.OKP():
		// Default to EdDSA and Ed25519 for OKP keys
		_ = key.Set(jwk.AlgorithmKey, jwa.EdDSA())
		_ = key.Set(jwk.OKPCrvKey, jwa.Ed25519())
	}
}

// GenerateKey generates a new jwk.Key
func GenerateKey(alg string, crv string) (key jwk.Key, err error) {
	var rawKey any
	switch alg {
	case jwa.RS256().String():
		rawKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case jwa.RS384().String():
		rawKey, err = rsa.GenerateKey(rand.Reader, 3072)
	case jwa.RS512().String():
		rawKey, err = rsa.GenerateKey(rand.Reader, 4096)
	case jwa.ES256().String():
		rawKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case jwa.ES384().String():
		rawKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case jwa.ES512().String():
		rawKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	case jwa.EdDSA().String():
		switch crv {
		case jwa.Ed25519().String():
			_, rawKey, err = ed25519.GenerateKey(rand.Reader)
		default:
			return nil, errors.New("unsupported curve for EdDSA algorithm")
		}
	default:
		return nil, errors.New("unsupported key algorithm")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Import the raw key
	return ImportRawKey(rawKey, alg, crv)
}
