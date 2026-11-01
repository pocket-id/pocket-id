package jwk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
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
	ReplaceKey(ctx context.Context, key jwk.Key) error
}

// GetKeyProvider returns the provider for the key used to sign tokens that are consumed externally, such as ID tokens and access tokens for apps
func GetKeyProvider(db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string) (KeyProvider, error) {
	return getKeyProvider(db, envConfig, instanceID, PrivateKeyDBKey)
}

// GetSessionKeyProvider returns the provider for the symmetric key that signs the known-browser cookie
// The key is stored as the session key because it signed the browser session before sessions became stateful
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

// LoadOrGenerateKey loads the provider's key, generating and saving one when none is stored yet
// A key that another replica saved first wins, so a conflicting save restarts the whole sequence up to three times
func LoadOrGenerateKey(ctx context.Context, provider KeyProvider, generate func() (jwk.Key, error)) (jwk.Key, error) {
	for retries := 0; ; retries++ {
		key, err := loadOrGenerateKeyOnce(ctx, provider, generate)
		if !errors.Is(err, ErrRetryKeyStorage) || retries == 3 {
			return key, err
		}

		// Wait briefly before reloading so the competing database transaction has time to finish
		slog.WarnContext(ctx, "Failed to store key, retrying", slog.Int("retry", retries+1), slog.Any("error", err))
		err = utils.SleepWithContext(ctx, 200*time.Millisecond)
		if err != nil {
			return nil, err
		}
	}
}

func loadOrGenerateKeyOnce(ctx context.Context, provider KeyProvider, generate func() (jwk.Key, error)) (jwk.Key, error) {
	key, err := provider.LoadKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load key: %w", err)
	}
	if key != nil {
		return key, nil
	}

	key, err = generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	err = provider.SaveKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}
	return key, nil
}
