package cmds

import (
	"testing"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
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

	appConfigService, err := service.NewAppConfigService(t.Context(), db)
	require.NoError(t, err)
	instanceID := appConfigService.GetDbConfig().InstanceID.Value

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

	flags := encryptionKeyRotateFlags{
		NewKey: string(newKey),
		Yes:    true,
	}
	require.NoError(t, encryptionKeyRotate(t.Context(), flags, db, envConfig))

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

	var storedToken string
	err = db.Model(&model.ScimServiceProvider{}).Where("id = ?", "scim-1").Pluck("token", &storedToken).Error
	require.NoError(t, err)

	newEncKey, err := datatype.DeriveEncryptedStringKey(newKey)
	require.NoError(t, err)

	decBytes, err := datatype.DecryptEncryptedStringWithKey(newEncKey, storedToken)
	require.NoError(t, err)
	assert.Equal(t, "scim-token-123", string(decBytes))
}
