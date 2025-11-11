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
		t.Setenv("DB_PROVIDER", "SQLITE") // should be lowercased automatically
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "HTTP://LOCALHOST:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, DbProviderSqlite, EnvConfig.DbProvider)
		assert.Equal(t, "http://localhost:3000", EnvConfig.AppURL)
	})

	t.Run("should parse valid Postgres config correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "POSTGRES")
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

	t.Run("should fail when ENCRYPTION_KEY is too short", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("ENCRYPTION_KEY", "short")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "ENCRYPTION_KEY must be at least 16 bytes long")
	})

	t.Run("should set default SQLite connection string when DB_CONNECTION_STRING is empty", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
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

	t.Run("should fail with invalid INTERNAL_APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("INTERNAL_APP_URL", "€://not-a-valid-url")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL is not a valid URL")
	})

	t.Run("should fail when INTERNAL_APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("INTERNAL_APP_URL", "http://localhost:3000/path")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL must not contain a path")
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
		t.Setenv("APP_ENV", "STAGING")
		t.Setenv("UPLOAD_PATH", "/custom/uploads")
		t.Setenv("KEYS_PATH", "/custom/keys")
		t.Setenv("PORT", "8080")
		t.Setenv("HOST", "LOCALHOST")
		t.Setenv("UNIX_SOCKET", "/tmp/app.sock")
		t.Setenv("MAXMIND_LICENSE_KEY", "test-license")
		t.Setenv("GEOLITE_DB_PATH", "/custom/geolite.mmdb")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "staging", EnvConfig.AppEnv) // lowercased
		assert.Equal(t, "/custom/uploads", EnvConfig.UploadPath)
		assert.Equal(t, "8080", EnvConfig.Port)
		assert.Equal(t, "localhost", EnvConfig.Host) // lowercased
	})

	t.Run("should normalize file backend and default upload path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "FS")
		t.Setenv("UPLOAD_PATH", "")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "fs", EnvConfig.FileBackend)
		assert.Equal(t, defaultFsUploadPath, EnvConfig.UploadPath)
	})

	t.Run("should fail when FILE_BACKEND is s3 but keys are stored on filesystem", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "s3")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "KEYS_STORAGE cannot be 'file' when FILE_BACKEND is 's3'")
	})

	t.Run("should fail with invalid FILE_BACKEND value", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "invalid")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid FILE_BACKEND value")
	})
}

func TestPrepareEnvConfig_FileBasedAndToLower(t *testing.T) {
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

	binaryKeyFile := tempDir + "/binary_key.bin"
	binaryKeyContent := []byte{0x01, 0x02, 0x03, 0x04}
	err = os.WriteFile(binaryKeyFile, binaryKeyContent, 0600)
	require.NoError(t, err)

	t.Run("should process toLower and file options", func(t *testing.T) {
		config := defaultConfig()
		config.AppEnv = "STAGING"
		config.Host = "LOCALHOST"

		t.Setenv("ENCRYPTION_KEY_FILE", encryptionKeyFile)
		t.Setenv("DB_CONNECTION_STRING_FILE", dbConnFile)

		err := prepareEnvConfig(&config)
		require.NoError(t, err)

		assert.Equal(t, "staging", config.AppEnv)
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, []byte(encryptionKeyContent), config.EncryptionKey)
		assert.Equal(t, dbConnContent, config.DbConnectionString)
	})

	t.Run("should handle binary data correctly", func(t *testing.T) {
		config := defaultConfig()
		t.Setenv("ENCRYPTION_KEY_FILE", binaryKeyFile)

		err := prepareEnvConfig(&config)
		require.NoError(t, err)
		assert.Equal(t, binaryKeyContent, config.EncryptionKey)
	})
}
