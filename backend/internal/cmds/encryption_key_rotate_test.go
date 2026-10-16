package cmds

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testingutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestEncryptionKeyRotate(t *testing.T) {
	oldKey := []byte("old-encryption-key-123456")
	newKey := []byte("new-encryption-key-654321")

	envConfig := &common.EnvConfigSchema{
		EncryptionKey: oldKey,
	}

	db := testingutils.NewDatabaseForTest(t)

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	oldKek, err := jwkutils.LoadKeyEncryptionKey(envConfig, instanceID)
	require.NoError(t, err)

	oldProvider := &jwkutils.KeyProviderDatabase{}
	require.NoError(t, oldProvider.Init(jwkutils.KeyProviderOpts{
		DB:  db,
		Kek: oldKek,
	}))

	signingKey, err := jwkutils.GenerateKey("RS256", "")
	require.NoError(t, err)
	require.NoError(t, oldProvider.SaveKey(t.Context(), signingKey))

	oldSessionKeyProvider := &jwkutils.KeyProviderDatabase{}
	err = oldSessionKeyProvider.Init(jwkutils.KeyProviderOpts{
		DB:    db,
		Kek:   oldKek,
		DBKey: jwkutils.SessionKeyDBKey,
	})
	require.NoError(t, err)

	sessionKey, err := jwkutils.GenerateSessionKey()
	require.NoError(t, err)
	err = oldSessionKeyProvider.SaveKey(t.Context(), sessionKey)
	require.NoError(t, err)

	oldEncKey, err := datatype.DeriveEncryptedStringKey(oldKey)
	require.NoError(t, err)
	encToken, err := datatype.EncryptEncryptedStringWithKey(oldEncKey, []byte("scim-token-123"))
	require.NoError(t, err)

	err = db.Exec(
		`INSERT INTO scim_service_providers (id, created_at, endpoint, token, oidc_client_id) VALUES (?, ?, ?, ?, ?)`,
		"scim-1",
		time.Now(),
		"https://example.com/scim",
		encToken,
		"client-1",
	).Error
	require.NoError(t, err)

	// Seed an S/MIME key encrypted with the old master key to exercise the same rotation as existing secrets
	signingCiphertext, err := datatype.EncryptEncryptedStringWithKey(oldEncKey, []byte("test-smime-private-key"))
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO email_signing_settings (id, private_key) VALUES (1, ?)", signingCiphertext).Error)

	flags := encryptionKeyRotateFlags{
		NewKey: string(newKey),
		Yes:    true,
	}
	err = encryptionKeyRotate(t.Context(), flags, db, instanceID, envConfig)
	require.NoError(t, err)

	newKek, err := jwkutils.LoadKeyEncryptionKey(&common.EnvConfigSchema{EncryptionKey: newKey}, instanceID)
	require.NoError(t, err)

	newProvider := &jwkutils.KeyProviderDatabase{}
	require.NoError(t, newProvider.Init(jwkutils.KeyProviderOpts{
		DB:  db,
		Kek: newKek,
	}))

	rotatedKey, err := newProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rotatedKey)

	// The session key must be re-encrypted with the new encryption key too, so sessions survive the rotation
	newSessionKeyProvider := &jwkutils.KeyProviderDatabase{}
	err = newSessionKeyProvider.Init(jwkutils.KeyProviderOpts{
		DB:    db,
		Kek:   newKek,
		DBKey: jwkutils.SessionKeyDBKey,
	})
	require.NoError(t, err)

	rotatedSessionKey, err := newSessionKeyProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rotatedSessionKey)

	sessionKeyID, _ := sessionKey.KeyID()
	rotatedSessionKeyID, _ := rotatedSessionKey.KeyID()
	assert.Equal(t, sessionKeyID, rotatedSessionKeyID, "The session key should be unchanged, only re-encrypted")

	var storedToken string
	err = db.Model(&scimsync.ServiceProvider{}).
		Where("id = ?", "scim-1").
		Pluck("token", &storedToken).
		Error
	require.NoError(t, err)

	newEncKey, err := datatype.DeriveEncryptedStringKey(newKey)
	require.NoError(t, err)

	decBytes, err := datatype.DecryptEncryptedStringWithKey(newEncKey, storedToken)
	require.NoError(t, err)
	assert.Equal(t, "scim-token-123", string(decBytes))
	var storedSigningKey string
	require.NoError(t, db.Raw("SELECT private_key FROM email_signing_settings WHERE id = 1").Scan(&storedSigningKey).Error)
	plainSigningKey, err := datatype.DecryptEncryptedStringWithKey(newEncKey, storedSigningKey)
	require.NoError(t, err)
	assert.Equal(t, "test-smime-private-key", string(plainSigningKey))
	_, err = datatype.DecryptEncryptedStringWithKey(oldEncKey, storedSigningKey)
	require.Error(t, err, "old key must no longer decrypt S/MIME material")
}
