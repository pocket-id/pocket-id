package jwk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lestrrat-go/jwx/v4/jwk"
)

var (
	// ErrPublicKeyNotAsymmetric is returned for keys that can't be used to verify a signature from a third party, such as symmetric ones
	ErrPublicKeyNotAsymmetric = errors.New("key is not an asymmetric public key")
	// ErrPublicKeyIsPrivate is returned for keys that contain private key material, which must never be uploaded to Pocket ID
	ErrPublicKeyIsPrivate = errors.New("key contains private key material")
	// ErrPublicKeyMissingKeyID is returned for keys without a "kid" property
	ErrPublicKeyMissingKeyID = errors.New(`key is missing the "kid" property`)
	// ErrPublicKeyNotForSigning is returned for keys whose "use" property restricts them to something other than verifying signatures
	ErrPublicKeyNotForSigning = errors.New(`key is not meant to verify signatures, its "use" is not "sig"`)
)

// ParsePublicKey parses a single JWK that is trusted to verify signatures, such as one of the public keys configured on a federated client credential.
func ParsePublicKey(raw []byte) (jwk.Key, error) {
	key, err := jwk.ParseKey(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	// Keys must be asymmetric and not have the private part
	isPrivate, err := jwk.IsPrivateKey(key)
	if err != nil {
		// An error indicates the key isn't asymmetric at all
		return nil, ErrPublicKeyNotAsymmetric
	} else if isPrivate {
		return nil, ErrPublicKeyIsPrivate
	}

	// Keys must have a "kid", which is required by Pocket ID to select the correct signing key
	kid, ok := key.KeyID()
	if !ok || kid == "" {
		return nil, ErrPublicKeyMissingKeyID
	}

	// A key restricted to encryption can never verify a signature
	use, ok := key.KeyUsage()
	if ok && use != "" && use != KeyUsageSigning {
		return nil, ErrPublicKeyNotForSigning
	}

	return key, nil
}

// ParsePublicKeySet parses a list of JWKs into a key set, validating each key with ParsePublicKey.
func ParsePublicKeySet(raw []json.RawMessage) (jwk.Set, error) {
	set := jwk.NewSet()

	for i, rawKey := range raw {
		key, err := ParsePublicKey(rawKey)
		if err != nil {
			return nil, fmt.Errorf("key %d is invalid: %w", i+1, err)
		}

		// Key IDs must be unique within the set
		kid, _ := key.KeyID()
		_, ok := set.LookupKeyID(kid)
		if ok {
			return nil, fmt.Errorf("key %d has the same key ID %q as an earlier key", i+1, kid)
		}

		err = set.AddKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to add key %d to the key set: %w", i+1, err)
		}
	}

	return set, nil
}

// NormalizePublicKeys validates a list of JWKs with ParsePublicKeySet and returns them re-encoded, so only well-formed keys are ever persisted.
func NormalizePublicKeys(raw []json.RawMessage) ([]json.RawMessage, error) {
	set, err := ParsePublicKeySet(raw)
	if err != nil {
		return nil, err
	}
	if set.Len() == 0 {
		return nil, nil
	}

	normalized := make([]json.RawMessage, set.Len())
	for i := range set.Len() {
		key, _ := set.Key(i)
		encoded, err := EncodeJWKBytes(key)
		if err != nil {
			return nil, fmt.Errorf("failed to encode key %d: %w", i+1, err)
		}
		normalized[i] = json.RawMessage(bytes.TrimSpace(encoded))
	}

	return normalized, nil
}
