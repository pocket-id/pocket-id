package jwk

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

const (
	// PrivateKeyDBKey is the row in the "kv" table containing the key used to sign tokens that are consumed externally
	PrivateKeyDBKey = "jwt_private_key.json"

	// SessionKeyDBKey is the row in the "kv" table containing the symmetric key used to sign session tokens
	SessionKeyDBKey = "session_key.json"
)

// ErrRetryKeyStorage signals that the caller should reload the key before trying to store it again
var ErrRetryKeyStorage = errors.New("retry key storage")

type KeyProviderDatabase struct {
	db    *gorm.DB
	kek   []byte
	dbKey string
}

func (f *KeyProviderDatabase) Init(opts KeyProviderOpts) error {
	if len(opts.Kek) == 0 {
		return errors.New("an encryption key is required when using the 'database' key provider")
	}

	f.db = opts.DB
	f.kek = opts.Kek

	// Callers that don't ask for a specific row get the token signing key, which is the key most of the codebase deals with
	f.dbKey = opts.DBKey
	if f.dbKey == "" {
		f.dbKey = PrivateKeyDBKey
	}

	return nil
}

func (f *KeyProviderDatabase) LoadKey(ctx context.Context) (key jwk.Key, err error) {
	row := model.KV{
		Key: f.dbKey,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = f.db.WithContext(ctx).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Key not present in the database - return nil so a new one can be generated
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve key from the database: %w", err)
	}

	if row.Value == nil || *row.Value == "" {
		// Key not present in the database - return nil so a new one can be generated
		return nil, nil
	}

	// Decode from base64
	enc, err := base64.StdEncoding.DecodeString(*row.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted key: not a valid base64-encoded value: %w", err)
	}

	// Decrypt the data
	data, err := cryptoutils.Decrypt(f.kek, enc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	// Parse the key
	key, err = jwk.ParseKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse encrypted key: %w", err)
	}

	return key, nil
}

func (f *KeyProviderDatabase) SaveKey(ctx context.Context, key jwk.Key) error {
	// Prepare the encrypted database value before attempting the insert
	row, err := f.prepareKeyRow(key)
	if err != nil {
		return err
	}

	// Insert only if the key doesn't exist yet
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result := f.db.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).
		Create(&row)
	if result.Error != nil {
		// Preserve ordinary database failures because only an existing row can be resolved by reloading
		return fmt.Errorf("failed to store key in database: %w", result.Error)
	}

	// Ask the caller to reload the winning key when another writer created the row first
	if result.RowsAffected == 0 {
		return ErrRetryKeyStorage
	}

	return nil
}

func (f *KeyProviderDatabase) ReplaceKey(ctx context.Context, key jwk.Key) error {
	// Prepare the encrypted database value before attempting the replacement
	row, err := f.prepareKeyRow(key)
	if err != nil {
		return err
	}

	// Upsert explicitly because key rotation must replace an existing key and can also recover a missing row
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = f.db.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		Create(&row).
		Error
	if err != nil {
		return fmt.Errorf("failed to replace key in database: %w", err)
	}

	return nil
}

func (f *KeyProviderDatabase) prepareKeyRow(key jwk.Key) (model.KV, error) {
	// Encode the key to JSON
	data, err := EncodeJWKBytes(key)
	if err != nil {
		return model.KV{}, fmt.Errorf("failed to encode key to JSON: %w", err)
	}

	// Encrypt the key then encode to Base64
	enc, err := cryptoutils.Encrypt(f.kek, data, nil)
	if err != nil {
		return model.KV{}, fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Build the row once so inserts and explicit replacements encode keys identically
	return model.KV{
		Key:   f.dbKey,
		Value: new(base64.StdEncoding.EncodeToString(enc)),
	}, nil
}

// Compile-time interface check
var _ KeyProvider = (*KeyProviderDatabase)(nil)
