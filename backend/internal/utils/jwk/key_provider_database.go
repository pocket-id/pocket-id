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
	// Save to database
	row := model.KV{
		Key:   f.dbKey,
		Value: new(base64.StdEncoding.EncodeToString(enc)),
	}

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
		// There's one scenario where if Pocket ID is started fresh with more than 1 replica, they both could be trying to create the key in the database at the same time
		// In this case, only one of the replicas will succeed and the other one(s) will return an error here, which will cascade down and cause the replica(s) to crash and be restarted (at that point they'll load the then-existing key from the database)
		return fmt.Errorf("failed to store key in database: %w", err)
	}

	return nil
}

// Compile-time interface check
var _ KeyProvider = (*KeyProviderDatabase)(nil)
