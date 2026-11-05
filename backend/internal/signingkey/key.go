// Package signingkey owns the instance's asymmetric key, which signs the tokens that are verified outside Pocket ID, such as ID tokens, access tokens and logout tokens
package signingkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type Key struct {
	db         *gorm.DB
	envConfig  *common.EnvConfigSchema
	privateKey jwk.Key
	jwks       []byte
}

// Load loads the key stored for the instance, generating and storing an RS256 key when there is none yet
func Load(ctx context.Context, db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string) (*Key, error) {
	k := &Key{db: db, envConfig: envConfig}
	err := k.Reload(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// Reload replaces the key with the one stored for the instance
// The stored key is encrypted with a key derived from the instance ID, which the E2E reset switches to a fixed value
func (k *Key) Reload(ctx context.Context, instanceID string) error {
	provider, err := jwkutils.GetKeyProvider(k.db, k.envConfig, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	privateKey, err := jwkutils.LoadOrGenerateKey(ctx, provider, func() (jwk.Key, error) {
		return jwkutils.GenerateKey(jwa.RS256().String(), "")
	})
	if err != nil {
		return fmt.Errorf("error loading signing key: %w", err)
	}

	return k.set(privateKey)
}

func (k *Key) set(privateKey jwk.Key) error {
	err := validate(privateKey)
	if err != nil {
		return fmt.Errorf("private key is not valid: %w", err)
	}

	// Encode the JWKS once, since it only changes with the key
	publicKey, err := privateKey.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}
	jwkutils.EnsureAlgInKey(publicKey, "", "")
	set := jwk.NewSet()
	err = set.AddKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to add public key to JWKS: %w", err)
	}
	jwks, err := json.Marshal(set)
	if err != nil {
		return fmt.Errorf("failed to encode JWKS to JSON: %w", err)
	}

	k.privateKey = privateKey
	k.jwks = jwks
	return nil
}

func validate(privateKey jwk.Key) error {
	err := privateKey.Validate()
	if err != nil {
		return fmt.Errorf("key object is invalid: %w", err)
	}
	keyID, ok := privateKey.KeyID()
	if !ok || keyID == "" {
		return errors.New("key object does not contain a key ID")
	}
	usage, ok := privateKey.KeyUsage()
	if !ok || usage != jwkutils.KeyUsageSigning {
		return errors.New("key object is not valid for signing")
	}
	ok, err = jwk.IsPrivateKey(privateKey)
	if err != nil || !ok {
		return errors.New("key object is not a private key")
	}
	if _, ok = privateKey.Algorithm(); !ok {
		return errors.New("key object does not contain an algorithm")
	}
	return nil
}

// GetPrivateKey returns the raw private key
func (k *Key) GetPrivateKey() any {
	privateKey, _ := jwk.Export[any](k.privateKey)
	return privateKey
}

// GetKeyAlg returns the algorithm the key signs with
func (k *Key) GetKeyAlg() (jwa.KeyAlgorithm, error) {
	alg, _ := k.privateKey.Algorithm()
	return alg, nil
}

func (k *Key) GetKeyID() (string, bool) {
	return k.privateKey.KeyID()
}

// PublicJWKS returns the JSON-encoded JWKS that publishes the public key
func (k *Key) PublicJWKS() []byte {
	return k.jwks
}

// Sign signs the token with extra protected headers, and the key ID header that lets verifiers pick the key from the JWKS
func (k *Key) Sign(token jwt.Token, headers jws.Headers) (string, error) {
	alg, _ := k.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, k.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return string(signed), nil
}
