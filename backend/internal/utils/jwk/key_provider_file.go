package jwk

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

const (
	// PrivateKeyFile is the path in the data/keys folder where the key is stored
	// This is a JSON file containing a key encoded as JWK
	PrivateKeyFile = "jwt_private_key.json"

	// PrivateKeyFileEncrypted is the path in the data/keys folder where the encrypted key is stored
	// This is a encrypted JSON file containing a key encoded as JWK
	PrivateKeyFileEncrypted = "jwt_private_key.json.enc"
)

type KeyProviderFile struct {
	envConfig *common.EnvConfigSchema
	kek       []byte
}

func (f *KeyProviderFile) Init(opts KeyProviderOpts) error {
	f.envConfig = opts.EnvConfig
	f.kek = opts.Kek

	return nil
}

func (f *KeyProviderFile) LoadKey() (jwk.Key, error) {
	if len(f.kek) > 0 {
		return f.loadEncryptedKey()
	}
	return f.loadKey()
}

func (f *KeyProviderFile) SaveKey(key jwk.Key) error {
	if len(f.kek) > 0 {
		return f.saveKeyEncrypted(key)
	}
	return f.saveKey(key)
}

func (f *KeyProviderFile) loadKey() (jwk.Key, error) {
	var key jwk.Key

	// First, check if we have a JWK file
	// If we do, then we just load that
	jwkPath := f.jwkPath()
	ok, err := utils.FileExists(jwkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if private key file exists at path '%s': %w", jwkPath, err)
	}
	if !ok {
		// File doesn't exist, no key was loaded
		return nil, nil
	}

	data, err := os.ReadFile(jwkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file at path '%s': %w", jwkPath, err)
	}

	key, err = jwk.ParseKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key file at path '%s': %w", jwkPath, err)
	}

	return key, nil
}

func (f *KeyProviderFile) loadEncryptedKey() (key jwk.Key, err error) {
	// First, check if we have an encrypted JWK file
	// If we do, then we just load that
	encJwkPath := f.encJwkPath()
	ok, err := utils.FileExists(encJwkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if encrypted private key file exists at path '%s': %w", encJwkPath, err)
	}
	if ok {
		encB64, err := os.ReadFile(encJwkPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read encrypted private key file at path '%s': %w", encJwkPath, err)
		}

		// Decode from base64
		enc := make([]byte, base64.StdEncoding.DecodedLen(len(encB64)))
		n, err := base64.StdEncoding.Decode(enc, encB64)
		if err != nil {
			return nil, fmt.Errorf("failed to read encrypted private key file at path '%s': not a valid base64-encoded file: %w", encJwkPath, err)
		}

		// Decrypt the data
		data, err := cryptoutils.Decrypt(f.kek, enc[:n], nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key file at path '%s': %w", encJwkPath, err)
		}

		// Parse the key
		key, err = jwk.ParseKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse encrypted private key file at path '%s': %w", encJwkPath, err)
		}

		return key, nil
	}

	// Check if we have an un-encrypted JWK file
	key, err = f.loadKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load un-encrypted key file: %w", err)
	}
	if key == nil {
		// No key exists, encrypted or un-encrypted
		return nil, nil
	}

	// If we are here, we have loaded a key that was un-encrypted
	// We need to replace the plaintext key with the encrypted one before we return
	err = f.saveKeyEncrypted(key)
	if err != nil {
		return nil, fmt.Errorf("failed to save encrypted key file: %w", err)
	}
	jwkPath := f.jwkPath()
	err = os.Remove(jwkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to remove un-encrypted key file at path '%s': %w", jwkPath, err)
	}

	return key, nil
}

func (f *KeyProviderFile) saveKey(key jwk.Key) error {
	err := os.MkdirAll(f.envConfig.KeysPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory '%s' for key file: %w", f.envConfig.KeysPath, err)
	}

	jwkPath := f.jwkPath()
	keyFile, err := os.OpenFile(jwkPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file at path '%s': %w", jwkPath, err)
	}
	defer keyFile.Close()

	// Write the JSON file to disk
	err = EncodeJWK(keyFile, key)
	if err != nil {
		return fmt.Errorf("failed to write key file at path '%s': %w", jwkPath, err)
	}

	return nil
}

func (f *KeyProviderFile) saveKeyEncrypted(key jwk.Key) error {
	err := os.MkdirAll(f.envConfig.KeysPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory '%s' for encrypted key file: %w", f.envConfig.KeysPath, err)
	}

	// Encode the key to JSON
	data, err := EncodeJWKBytes(key)
	if err != nil {
		return fmt.Errorf("failed to encode key to JSON: %w", err)
	}

	// Encrypt the key then encode to Base64
	enc, err := cryptoutils.Encrypt(f.kek, data, nil)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}
	encB64 := make([]byte, base64.StdEncoding.EncodedLen(len(enc)))
	base64.StdEncoding.Encode(encB64, enc)

	// Write to disk
	encJwkPath := f.encJwkPath()
	err = os.WriteFile(encJwkPath, encB64, 0600)
	if err != nil {
		return fmt.Errorf("failed to write encrypted key file at path '%s': %w", encJwkPath, err)
	}

	return nil
}

func (f *KeyProviderFile) jwkPath() string {
	return filepath.Join(f.envConfig.KeysPath, PrivateKeyFile)
}

func (f *KeyProviderFile) encJwkPath() string {
	return filepath.Join(f.envConfig.KeysPath, PrivateKeyFileEncrypted)
}

// Compile-time interface check
var _ KeyProvider = (*KeyProviderFile)(nil)
