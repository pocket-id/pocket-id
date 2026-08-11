package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// versionBeforeMultipleClientSecrets is the migration version right before client secrets moved into the credentials document
const versionBeforeMultipleClientSecrets = 20260802120000

// TestMigrateClientSecretsToCredentials checks that the secret of every existing client is preserved as an entry of the credentials document
func TestMigrateClientSecretsToCredentials(t *testing.T) {
	const legacySecret = "legacy-client-secret"

	legacyHash, err := bcrypt.GenerateFromPassword([]byte(legacySecret), bcrypt.MinCost)
	require.NoError(t, err)

	createdAt := time.Now().Add(-72 * time.Hour).Truncate(time.Second)

	db := testutils.NewDatabaseForTestWithMigrationSeed(t, versionBeforeMultipleClientSecrets, func(t *testing.T, db *gorm.DB) {
		// A client with a secret and no other credentials
		err := db.Exec(
			`INSERT INTO oidc_clients (id, created_at, name, secret, callback_urls, is_public, pkce_enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"client-with-secret", createdAt.Unix(), "With Secret", string(legacyHash), `["https://example.com/callback"]`, false, false,
		).Error
		require.NoError(t, err)

		// A client whose credentials document already holds a federated identity, which the migration must preserve
		err = db.Exec(
			`INSERT INTO oidc_clients (id, created_at, name, secret, callback_urls, is_public, pkce_enabled, credentials) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"client-with-federated-identity", createdAt.Unix(), "With Federated Identity", string(legacyHash), `["https://example.com/callback"]`, false, false,
			`{"federatedIdentities":[{"issuer":"https://issuer.example.com"}]}`,
		).Error
		require.NoError(t, err)

		// A public client, which never had a secret
		err = db.Exec(
			`INSERT INTO oidc_clients (id, created_at, name, secret, callback_urls, is_public, pkce_enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"public-client", createdAt.Unix(), "Public", "", `["https://example.com/callback"]`, true, true,
		).Error
		require.NoError(t, err)
	})

	// The legacy column is gone once the migration has run
	ok := db.Migrator().HasColumn(&model.OidcClient{}, "secret")
	assert.False(t, ok)

	var clients []model.OidcClient
	err = db.Find(&clients).Error
	require.NoError(t, err)
	byID := make(map[string]model.OidcClient, len(clients))
	for _, client := range clients {
		byID[client.ID] = client
	}
	require.Len(t, byID, 3)

	// The migrated secret keeps its bcrypt hash, has no expiration, and carries no prefix because the value was never stored
	migrated := byID["client-with-secret"].Credentials.Secrets
	require.Len(t, migrated, 1)
	assert.NotEmpty(t, migrated[0].ID)
	assert.Equal(t, model.OidcClientSecretHashBcrypt, migrated[0].Algorithm)
	assert.Equal(t, string(legacyHash), migrated[0].Hash)
	assert.Empty(t, migrated[0].Prefix)
	assert.Nil(t, migrated[0].ExpiresAt)
	assert.True(t, migrated[0].IsActive())
	assert.Equal(t, createdAt.UTC(), migrated[0].CreatedAt.UTC())

	// Existing federated identities survive the migration alongside the new secret
	withFederated := byID["client-with-federated-identity"].Credentials
	require.Len(t, withFederated.Secrets, 1)
	assert.Equal(t, string(legacyHash), withFederated.Secrets[0].Hash)
	require.Len(t, withFederated.FederatedIdentities, 1)
	assert.Equal(t, "https://issuer.example.com", withFederated.FederatedIdentities[0].Issuer)

	// Clients that never had a secret do not get an empty one
	assert.Empty(t, byID["public-client"].Credentials.Secrets)

	// Every migrated secret gets its own identifier
	assert.NotEqual(t, migrated[0].ID, withFederated.Secrets[0].ID)
}
