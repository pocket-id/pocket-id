package jwk

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type KeyProviderOpts struct {
	EnvConfig *common.EnvConfigSchema
	DB        *gorm.DB
	Kek       []byte
}

type KeyProvider interface {
	Init(opts KeyProviderOpts) error
	LoadKey(ctx context.Context) (jwk.Key, error)
	SaveKey(ctx context.Context, key jwk.Key) error
}

func GetKeyProvider(db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string) (keyProvider KeyProvider, err error) {
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init key provider: %w", err)
	}

	return keyProvider, nil
}
