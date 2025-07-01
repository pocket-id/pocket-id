package common

import (
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

	t.Run("should default KEYS_STORAGE to 'file' when no encryption key", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "file", EnvConfig.KeysStorage)
	})

	t.Run("should default KEYS_STORAGE to 'database' when encryption key provided", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("ENCRYPTION_KEY", "test-key")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "database", EnvConfig.KeysStorage)
	})

	t.Run("should default KEYS_STORAGE to 'database' when encryption key file provided", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("ENCRYPTION_KEY_FILE", "/path/to/key")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, "database", EnvConfig.KeysStorage)
	})

	t.Run("should fail when KEYS_STORAGE is 'database' but no encryption key", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("KEYS_STORAGE", "database")

		err := parseEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "ENCRYPTION_KEY or ENCRYPTION_KEY_FILE must be non-empty")
	})

	t.Run("should accept valid KEYS_STORAGE values", func(t *testing.T) {
		validStorageTypes := []string{"file", "memory", "database"}

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
