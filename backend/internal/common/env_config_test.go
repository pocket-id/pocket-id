package common

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseAndValidateEnvConfig(t *testing.T) error {
	t.Helper()

	if _, exists := os.LookupEnv("ENCRYPTION_KEY"); !exists {
		t.Setenv("ENCRYPTION_KEY", "0123456789abcdef")
	}

	if err := parseEnvConfig(); err != nil {
		return err
	}

	return ValidateEnvConfig(&EnvConfig)
}

func TestParseEnvConfig(t *testing.T) {
	// Store original config to restore later
	originalConfig := EnvConfig
	t.Cleanup(func() {
		EnvConfig = originalConfig
	})

	t.Run("should parse valid SQLite config correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "HTTP://LOCALHOST:3000")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, DbProviderSqlite, EnvConfig.DbProvider)
		assert.Equal(t, "http://localhost:3000", EnvConfig.AppURL)
	})

	t.Run("should parse valid Postgres config correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "postgres://user:pass@localhost/db")
		t.Setenv("APP_URL", "https://example.com")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, DbProviderPostgres, EnvConfig.DbProvider)
	})

	t.Run("should fail when ENCRYPTION_KEY is too short", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("ENCRYPTION_KEY", "short")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "ENCRYPTION_KEY must be at least 16 bytes long")
	})

	t.Run("should set default SQLite connection string when DB_CONNECTION_STRING is empty", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, defaultSqliteConnString, EnvConfig.DbConnectionString)
	})

	t.Run("should fail with invalid APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "€://not-a-valid-url")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL is not a valid URL")
	})

	t.Run("should fail when APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000/path")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL must not contain a path")
	})

	t.Run("should fail with invalid INTERNAL_APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("INTERNAL_APP_URL", "€://not-a-valid-url")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL is not a valid URL")
	})

	t.Run("should fail when INTERNAL_APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("INTERNAL_APP_URL", "http://localhost:3000/path")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL must not contain a path")
	})

	t.Run("should parse boolean environment variables correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("UI_CONFIG_DISABLED", "true")
		t.Setenv("METRICS_ENABLED", "true")
		t.Setenv("TRACING_ENABLED", "false")
		t.Setenv("TRUST_PROXY", "true")
		t.Setenv("ANALYTICS_DISABLED", "false")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.True(t, EnvConfig.UiConfigDisabled)
		assert.True(t, EnvConfig.MetricsEnabled)
		assert.False(t, EnvConfig.TracingEnabled)
		assert.True(t, EnvConfig.TrustProxy)
		assert.False(t, EnvConfig.AnalyticsDisabled)
	})

	t.Run("should parse string environment variables correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "postgres://test")
		t.Setenv("APP_URL", "https://prod.example.com")
		t.Setenv("APP_ENV", "PRODUCTION")
		t.Setenv("UPLOAD_PATH", "/custom/uploads")
		t.Setenv("PORT", "8080")
		t.Setenv("HOST", "LOCALHOST")
		t.Setenv("UNIX_SOCKET", "/tmp/app.sock")
		t.Setenv("MAXMIND_LICENSE_KEY", "test-license")
		t.Setenv("GEOLITE_DB_PATH", "/custom/geolite.mmdb")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, AppEnvProduction, EnvConfig.AppEnv) // lowercased
		assert.Equal(t, "/custom/uploads", EnvConfig.UploadPath)
		assert.Equal(t, "8080", EnvConfig.Port)
		assert.Equal(t, "localhost", EnvConfig.Host) // lowercased
	})

	t.Run("should normalize file backend and default upload path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "FILESYSTEM")
		t.Setenv("UPLOAD_PATH", "")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, "filesystem", EnvConfig.FileBackend)
		assert.Equal(t, defaultFsUploadPath, EnvConfig.UploadPath)
	})

	t.Run("should fail with invalid FILE_BACKEND value", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "invalid")

		err := parseAndValidateEnvConfig(t)
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

		assert.Equal(t, AppEnv("staging"), config.AppEnv)
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
