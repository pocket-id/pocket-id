package common

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnvConfig(t *testing.T) {
	// Store original config to restore later
	originalConfig := EnvConfig
	t.Cleanup(func() {
		EnvConfig = originalConfig
	})

	t.Run("should parse valid SQLite config correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, DbProviderSqlite, EnvConfig.DbProvider)
	})

	t.Run("should parse valid Postgres config correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "postgres")
		t.Setenv("DB_CONNECTION_STRING", "postgres://user:pass@localhost/db")
		t.Setenv("APP_URL", "https://example.com")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, DbProviderPostgres, EnvConfig.DbProvider)
	})

	t.Run("should fail with invalid DB_PROVIDER", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "invalid")
		t.Setenv("DB_CONNECTION_STRING", "test")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid DB_PROVIDER value")
	})

	t.Run("should set default SQLite connection string when DB_CONNECTION_STRING is empty", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "") // Explicitly empty
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, defaultSqliteConnString, EnvConfig.DbConnectionString)
	})

	t.Run("should fail when Postgres DB_CONNECTION_STRING is missing", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "postgres")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "missing required env var 'DB_CONNECTION_STRING' for Postgres")
	})

	t.Run("should fail with invalid APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "€://not-a-valid-url")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL is not a valid URL")
	})

	t.Run("should fail when APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000/path")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL must not contain a path")
	})

	t.Run("should default KEYS_STORAGE to 'file' when empty", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "file", EnvConfig.KeysStorage)
	})

	t.Run("should fail when KEYS_STORAGE is 'database' but no encryption key", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("KEYS_STORAGE", "database")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "ENCRYPTION_KEY must be non-empty when KEYS_STORAGE is database")
	})

	t.Run("should accept valid KEYS_STORAGE values", func(t *testing.T) {
		validStorageTypes := []string{"file", "database"}

		for _, storage := range validStorageTypes {
			EnvConfig = defaultConfig()
			t.Setenv("DB_PROVIDER", "sqlite")
			t.Setenv("DB_CONNECTION_STRING", "file:test.db")
			t.Setenv("APP_URL", "http://localhost:3000")
			t.Setenv("KEYS_STORAGE", storage)
			if storage == "database" {
				t.Setenv("ENCRYPTION_KEY", "test-key")
			}

			err := parseEnvConfig()
			require.NoError(t, err)
			assert.Equal(t, storage, EnvConfig.KeysStorage)
		}
	})

	t.Run("should fail with invalid KEYS_STORAGE value", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("KEYS_STORAGE", "invalid")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid value for KEYS_STORAGE")
	})

	t.Run("should parse boolean environment variables correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("UI_CONFIG_DISABLED", "true")
		t.Setenv("METRICS_ENABLED", "true")
		t.Setenv("TRACING_ENABLED", "false")
		t.Setenv("TRUST_PROXY", "true")
		t.Setenv("ANALYTICS_DISABLED", "false")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.True(t, EnvConfig.UiConfigDisabled)
		assert.True(t, EnvConfig.MetricsEnabled)
		assert.False(t, EnvConfig.TracingEnabled)
		assert.True(t, EnvConfig.TrustProxy)
		assert.False(t, EnvConfig.AnalyticsDisabled)
	})

	t.Run("should parse string environment variables correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "postgres")
		t.Setenv("DB_CONNECTION_STRING", "postgres://test")
		t.Setenv("APP_URL", "https://prod.example.com")
		t.Setenv("APP_ENV", "staging")
		t.Setenv("UPLOAD_PATH", "/custom/uploads")
		t.Setenv("KEYS_PATH", "/custom/keys")
		t.Setenv("PORT", "8080")
		t.Setenv("HOST", "127.0.0.1")
		t.Setenv("UNIX_SOCKET", "/tmp/app.sock")
		t.Setenv("MAXMIND_LICENSE_KEY", "test-license")
		t.Setenv("GEOLITE_DB_PATH", "/custom/geolite.mmdb")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "staging", EnvConfig.AppEnv)
		assert.Equal(t, "/custom/uploads", EnvConfig.UploadPath)
		assert.Equal(t, "8080", EnvConfig.Port)
		assert.Equal(t, "127.0.0.1", EnvConfig.Host)
	})
}

func TestResolveFileBasedEnvVariables(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create test files
	encryptionKeyFile := tempDir + "/encryption_key.txt"
	encryptionKeyContent := "test-encryption-key-123"
	err := os.WriteFile(encryptionKeyFile, []byte(encryptionKeyContent), 0600)
	require.NoError(t, err)

	dbConnFile := tempDir + "/db_connection.txt"
	dbConnContent := "postgres://user:pass@localhost/testdb"
	err = os.WriteFile(dbConnFile, []byte(dbConnContent), 0600)
	require.NoError(t, err)

	// Create a binary file for testing binary data handling
	binaryKeyFile := tempDir + "/binary_key.bin"
	binaryKeyContent := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	err = os.WriteFile(binaryKeyFile, binaryKeyContent, 0600)
	require.NoError(t, err)

	t.Run("should read file content for fields with options:file tag", func(t *testing.T) {
		config := defaultConfig()

		// Set environment variables pointing to files
		t.Setenv("ENCRYPTION_KEY_FILE", encryptionKeyFile)
		t.Setenv("DB_CONNECTION_STRING_FILE", dbConnFile)

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// Verify file contents were read correctly
		assert.Equal(t, []byte(encryptionKeyContent), config.EncryptionKey)
		assert.Equal(t, dbConnContent, config.DbConnectionString)
	})

	t.Run("should skip fields without options:file tag", func(t *testing.T) {
		config := defaultConfig()
		originalAppURL := config.AppURL

		// Set a file for a field that doesn't have options:file tag
		t.Setenv("APP_URL_FILE", "/tmp/nonexistent.txt")

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// AppURL should remain unchanged
		assert.Equal(t, originalAppURL, config.AppURL)
	})

	t.Run("should skip non-string fields", func(t *testing.T) {
		// This test verifies that non-string fields are skipped
		// We test this indirectly by ensuring the function doesn't error
		// when processing the actual EnvConfigSchema which has bool fields
		config := defaultConfig()

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)
	})

	t.Run("should skip when _FILE environment variable is not set", func(t *testing.T) {
		config := defaultConfig()
		originalEncryptionKey := config.EncryptionKey

		// Don't set ENCRYPTION_KEY_FILE environment variable

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// EncryptionKey should remain unchanged
		assert.Equal(t, originalEncryptionKey, config.EncryptionKey)
	})

	t.Run("should handle multiple file-based variables simultaneously", func(t *testing.T) {
		config := defaultConfig()

		// Set multiple file environment variables
		t.Setenv("ENCRYPTION_KEY_FILE", encryptionKeyFile)
		t.Setenv("DB_CONNECTION_STRING_FILE", dbConnFile)

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// All should be resolved correctly
		assert.Equal(t, []byte(encryptionKeyContent), config.EncryptionKey)
		assert.Equal(t, dbConnContent, config.DbConnectionString)
	})

	t.Run("should handle mixed file and non-file environment variables", func(t *testing.T) {
		config := defaultConfig()

		// Set both file and non-file environment variables
		t.Setenv("ENCRYPTION_KEY_FILE", encryptionKeyFile)

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// File-based should be resolved, others should remain as set by env parser
		assert.Equal(t, []byte(encryptionKeyContent), config.EncryptionKey)
		assert.Equal(t, "http://localhost:1411", config.AppURL)
	})

	t.Run("should handle binary data correctly", func(t *testing.T) {
		config := defaultConfig()

		// Set environment variable pointing to binary file
		t.Setenv("ENCRYPTION_KEY_FILE", binaryKeyFile)

		err := resolveFileBasedEnvVariables(&config)
		require.NoError(t, err)

		// Verify binary data was read correctly without corruption
		assert.Equal(t, binaryKeyContent, config.EncryptionKey)
	})
}
