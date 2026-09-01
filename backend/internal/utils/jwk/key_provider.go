package jwk

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type KeyProviderOpts struct {
	EnvConfig *common.EnvConfigSchema
	DB        *gorm.DB
	Kek       []byte
	DBKey     string
}

type KeyProvider interface {
	Init(opts KeyProviderOpts) error
	LoadKey(ctx context.Context) (jwk.Key, error)
	SaveKey(ctx context.Context, key jwk.Key) error
}

// GetKeyProvider returns the provider for the key used to sign tokens that are consumed externally, such as ID tokens and access tokens for apps
func GetKeyProvider(db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string) (KeyProvider, error) {
	return getKeyProvider(db, envConfig, instanceID, PrivateKeyDBKey)
}

// GetSessionKeyProvider returns the provider for the symmetric key used to sign session tokens
// This key is symmetric and separate from the one used to sign tokens for external consumption
func GetSessionKeyProvider(db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string) (KeyProvider, error) {
	return getKeyProvider(db, envConfig, instanceID, SessionKeyDBKey)
}

func getKeyProvider(db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string, dbKey string) (keyProvider KeyProvider, err error) {
	// Load the encryption key (KEK) if present
	kek, err := LoadKeyEncryptionKey(envConfig, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load encryption key: %w", err)
	}

	keyProvider = &KeyProviderDatabase{}
	err = keyProvider.Init(KeyProviderOpts{
		DB:        db,
		EnvConfig: envConfig,
		Kek:       kek,
		DBKey:     dbKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init key provider: %w", err)
	}

	return keyProvider, nil
}
