package bootstrap

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	privateKeyFilePem = "jwt_private_key.pem"
)

func migrateKey() {
	err := migrateKeyInternal(common.EnvConfig.KeysPath)
	if err != nil {
		log.Fatalf("failed to perform migration of keys: %v", err)
	}
}

func migrateKeyInternal(basePath string) error {
	// First, check if there's already a JWK stored
	jwkPath := filepath.Join(basePath, service.PrivateKeyFile)
	ok, err := utils.FileExists(jwkPath)
	if err != nil {
		return fmt.Errorf("failed to check if private key file (JWK) exists at path '%s': %w", jwkPath, err)
	}
	if ok {
		// There's already a key as JWK, so we don't do anything else here
		return nil
	}

	// Check if there's a PEM file
	pemPath := filepath.Join(basePath, privateKeyFilePem)
	ok, err = utils.FileExists(pemPath)
	if err != nil {
		return fmt.Errorf("failed to check if private key file (PEM) exists at path '%s': %w", pemPath, err)
	}
	if !ok {
		// No file to migrate, return
		return nil
	}

	// Load and validate the key
	key, err := loadKeyPEM(pemPath)
	if err != nil {
		return fmt.Errorf("failed to load private key file (PEM) at path '%s': %w", pemPath, err)
	}
	err = service.ValidateKey(key)
	if err != nil {
		return fmt.Errorf("key object is invalid: %w", err)
	}

	// Save the key as JWK
	err = service.SaveKeyJWK(key, jwkPath)
	if err != nil {
		return fmt.Errorf("failed to save private key file at path '%s': %w", jwkPath, err)
	}

	// Finally, delete the PEM file
	err = os.Remove(pemPath)
	if err != nil {
		return fmt.Errorf("failed to remove migrated key at path '%s': %w", pemPath, err)
	}

	return nil
}

func loadKeyPEM(path string) (jwk.Key, error) {
	// Load the key from disk and parse it
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key data: %w", err)
	}

	key, err := jwk.ParseKey(data, jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	// Populate the key ID using the "legacy" algorithm
	keyId, err := generateKeyID(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	key.Set(jwk.KeyIDKey, keyId)

	// Populate other required fields
	_ = key.Set(jwk.KeyUsageKey, service.KeyUsageSigning)
	service.EnsureAlgInKey(key)

	return key, nil
}

// generateKeyID generates a Key ID for the public key using the first 8 bytes of the SHA-256 hash of the public key's PKIX-serialized structure.
// This is used for legacy keys, imported from PEM.
func generateKeyID(key jwk.Key) (string, error) {
	// Export the public key and serialize it to PKIX (not in a PEM block)
	// This is for backwards-compatibility with the algorithm used before the switch to JWK
	pubKey, err := key.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}
	var pubKeyRaw any
	err = jwk.Export(pubKey, &pubKeyRaw)
	if err != nil {
		return "", fmt.Errorf("failed to export public key: %w", err)
	}
	pubASN1, err := x509.MarshalPKIXPublicKey(pubKeyRaw)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Compute SHA-256 hash of the public key
	hash := sha256.New()
	hash.Write(pubASN1)
	hashed := hash.Sum(nil)

	// Truncate the hash to the first 8 bytes for a shorter Key ID
	shortHash := hashed[:8]

	// Return Base64 encoded truncated hash as Key ID
	return base64.RawURLEncoding.EncodeToString(shortHash), nil
}
