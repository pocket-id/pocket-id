package jwk

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

const PrivateKeyDBKey = "jwt_private_key.json"

type KeyProviderDatabase struct {
	db  *gorm.DB
	kek []byte
}

func (f *KeyProviderDatabase) Init(opts KeyProviderOpts) error {
	if len(opts.Kek) == 0 {
		return errors.New("an encryption key is required when using the 'database' key provider")
	}

	f.db = opts.DB
	f.kek = opts.Kek

	return nil
}

func (f *KeyProviderDatabase) LoadKey() (key jwk.Key, err error) {
	row := model.KV{
		Key: PrivateKeyDBKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = f.db.WithContext(ctx).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Key not present in the database - return nil so a new one can be generated
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve private key from the database: %w", err)
	}

	if row.Value == nil || *row.Value == "" {
		// Key not present in the database - return nil so a new one can be generated
		return nil, nil
	}

	// Decode from base64
	enc, err := base64.StdEncoding.DecodeString(*row.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted private key: not a valid base64-encoded value: %w", err)
	}

	// Decrypt the data
	data, err := cryptoutils.Decrypt(f.kek, enc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Parse the key
	key, err = jwk.ParseKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse encrypted private key: %w", err)
	}

	return key, nil
}

func (f *KeyProviderDatabase) SaveKey(key jwk.Key) error {
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
	encB64 := base64.StdEncoding.EncodeToString(enc)

	// Save to database
	row := model.KV{
		Key:   PrivateKeyDBKey,
		Value: &encB64,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = f.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		// There's one scenario where if Pocket ID is started fresh with more than 1 replica, they both could be trying to create the private key in the database at the same time
		// In this case, only one of the replicas will succeed; the other one(s) will return an error here, which will cascade down and cause the replica(s) to crash and be restarted (at that point they'll load the then-existing key from the database)
		return fmt.Errorf("failed to store private key in database: %w", err)
	}

	return nil
}

// Compile-time interface check
var _ KeyProvider = (*KeyProviderDatabase)(nil)
