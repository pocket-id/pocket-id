package common

import (
	"net"
	"os"
	"path/filepath"
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
		t.Setenv("DB_CONNECTION_STRING", "file:test")
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
		t.Setenv("DB_CONNECTION_STRING", "file:test")
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

	t.Run("should reject a SQLite connection string that names a database file", func(t *testing.T) {
		existing := filepath.Join(t.TempDir(), "auth")
		require.NoError(t, os.WriteFile(existing, nil, 0o600))

		for _, connString := range []string{"data/pocket-id.db", "file:/app/data/pocket-id.sqlite?_pragma=busy_timeout(5000)", existing} {
			EnvConfig = defaultConfig()
			t.Setenv("DB_CONNECTION_STRING", connString)
			t.Setenv("APP_URL", "http://localhost:3000")

			err := parseAndValidateEnvConfig(t)
			require.ErrorContains(t, err, "must be the folder of the SQLite databases", connString)
		}
	})

	t.Run("should derive both SQLite files from the folder and keep its query parameters", func(t *testing.T) {
		// The file is joined with the separator of the OS, which SQLite accepts on Windows too
		config := EnvConfigSchema{DbConnectionString: "file:/app/data?_pragma=busy_timeout(5000)"}
		assert.Equal(t, "file:"+filepath.Join("/app/data", "pocket-id.db")+"?_pragma=busy_timeout(5000)", config.SqliteConnString("pocket-id.db"))
		assert.Equal(t, "file:"+filepath.Join("/app/data", "francis.db")+"?_pragma=busy_timeout(5000)", config.SqliteConnString("francis.db"))

		config.DbConnectionString = "data"
		assert.Equal(t, "file:"+filepath.Join("data", "francis.db"), config.SqliteConnString("francis.db"))
	})

	t.Run("should fail with invalid APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "€://not-a-valid-url")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL is not a valid URL")
	})

	t.Run("should fail when APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000/path")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "APP_URL must not contain a path")
	})

	t.Run("should fail with invalid INTERNAL_APP_URL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("INTERNAL_APP_URL", "€://not-a-valid-url")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL is not a valid URL")
	})

	t.Run("should fail when INTERNAL_APP_URL contains path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("INTERNAL_APP_URL", "http://localhost:3000/path")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "INTERNAL_APP_URL must not contain a path")
	})

	t.Run("should parse boolean environment variables correctly", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("UI_CONFIG_DISABLED", "true")
		t.Setenv("METRICS_ENABLED", "true")
		t.Setenv("TRACING_ENABLED", "false")
		t.Setenv("TRUST_PROXY", "true")
		t.Setenv("PROXY_PROTOCOL", "true")
		t.Setenv("ANALYTICS_DISABLED", "false")
		t.Setenv("ALLOW_INSECURE_CALLBACK_URLS", "false")
		t.Setenv("CLOUDFLARE_LOCATION_HEADERS", "true")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.True(t, EnvConfig.UiConfigDisabled)
		assert.Equal(t, TrustProxyConfig{"0.0.0.0/0", "::/0"}, EnvConfig.TrustProxy)
		assert.Equal(t, TrustProxyConfig{"0.0.0.0/0", "::/0"}, EnvConfig.ProxyProtocol)
		assert.False(t, EnvConfig.AnalyticsDisabled)
		assert.False(t, EnvConfig.AllowInsecureCallbackURLs)
		assert.True(t, EnvConfig.CloudflareLocationHeaders)
	})

	t.Run("should parse trusted proxy IP addresses and CIDR ranges", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("TRUST_PROXY", "10.0.0.0/8, 192.168.1.10, ::1/128")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, TrustProxyConfig{"10.0.0.0/8", "192.168.1.10", "::1/128"}, EnvConfig.TrustProxy)
	})

	t.Run("should disable trusted proxies when set to false", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("TRUST_PROXY", "false")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Nil(t, EnvConfig.TrustProxy)
	})

	t.Run("should parse PROXY protocol trusted proxy IP addresses and CIDR ranges", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("PROXY_PROTOCOL", "10.0.0.0/8, 192.168.1.10, ::1/128")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, TrustProxyConfig{"10.0.0.0/8", "192.168.1.10", "::1/128"}, EnvConfig.ProxyProtocol)
	})

	t.Run("should reject an invalid PROXY protocol trusted proxy", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("PROXY_PROTOCOL", "not-an-ip")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid proxy IP address or CIDR")
	})

	t.Run("should normalize allowed frame ancestors to origins", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("ALLOWED_FRAME_ANCESTORS", "HTTPS://Dashboard.Example.com/, http://localhost:8080")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, FrameAncestorsConfig{"https://dashboard.example.com", "http://localhost:8080"}, EnvConfig.AllowedFrameAncestors)
	})

	t.Run("should reject an allowed frame ancestor that isn't an origin", func(t *testing.T) {
		for _, value := range []string{"dashboard.example.com", "*", "https://*.example.com", "https://example.com/dashboard", "https://example.com;report-uri", "https://example.com,", "javascript://example.com"} {
			EnvConfig = defaultConfig()
			t.Setenv("ALLOWED_FRAME_ANCESTORS", value)

			err := parseAndValidateEnvConfig(t)
			assert.ErrorContains(t, err, "invalid frame ancestor", value)
		}
	})

	t.Run("should reject PROXY protocol with a UNIX socket", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("PROXY_PROTOCOL", "true")
		t.Setenv("UNIX_SOCKET", "/tmp/pocket-id.sock")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "PROXY_PROTOCOL and UNIX_SOCKET are mutually exclusive")
	})

	t.Run("should allow insecure callback URLs by default", func(t *testing.T) {
		assert.True(t, defaultConfig().AllowInsecureCallbackURLs)
	})

	t.Run("should default audit log retention days to 90", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, 90, EnvConfig.AuditLogRetentionDays)
	})

	t.Run("should parse audit log retention days override", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("AUDIT_LOG_RETENTION_DAYS", "365")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, 365, EnvConfig.AuditLogRetentionDays)
	})

	t.Run("should fail when AUDIT_LOG_RETENTION_DAYS is non-positive", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("AUDIT_LOG_RETENTION_DAYS", "0")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "AUDIT_LOG_RETENTION_DAYS must be greater than 0")
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
		t.Setenv("CLUSTER_PORT", "9999")
		t.Setenv("CLUSTER_ADDRESS", "LOCALHOST")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, AppEnvProduction, EnvConfig.AppEnv) // lowercased
		assert.Equal(t, "/custom/uploads", EnvConfig.UploadPath)
		assert.Equal(t, "8080", EnvConfig.Port)
		assert.Equal(t, "localhost", EnvConfig.Host) // lowercased
		assert.Equal(t, "9999", EnvConfig.ClusterPort)
		assert.Equal(t, "localhost", EnvConfig.ClusterAddress) // lowercased
		assert.Equal(t, "/custom/geolite.mmdb", EnvConfig.GeoLiteDBPath)
	})

	t.Run("should keep the GeoLite database in the cache folder unless a path is set", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("CACHE_PATH", "/custom/cache")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/custom/cache", "GeoLite2-City.mmdb"), EnvConfig.GeoLiteDBPath)
	})

	t.Run("should normalize file backend and default upload path", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
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
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("FILE_BACKEND", "invalid")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid FILE_BACKEND value")
	})

	t.Run("should fail when TLS cert is set without key", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT", "certificate")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_KEY must be set when TLS_CERT is set")
	})

	t.Run("should fail when TLS key is set without cert", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_KEY", "private key")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT must be set when TLS_KEY is set")
	})

	t.Run("should fail when TLS cert file is set without key file", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_KEY_FILE must be set when TLS_CERT_FILE is set")
	})

	t.Run("should fail when TLS key file is set without cert file", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_KEY_FILE", "/path/to/key.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT_FILE must be set when TLS_KEY_FILE is set")
	})

	t.Run("should fail when inline and file TLS configuration are combined", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT", "certificate")
		t.Setenv("TLS_KEY", "private key")
		t.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")
		t.Setenv("TLS_KEY_FILE", "/path/to/key.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT and TLS_KEY cannot be combined with TLS_CERT_FILE or TLS_KEY_FILE")
	})

	t.Run("should fail when TLS cert file does not exist", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT_FILE", "/nonexistent/cert.pem")

		keyFile := t.TempDir() + "/key.pem"
		require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0600))
		t.Setenv("TLS_KEY_FILE", keyFile)

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT_FILE not found")
	})

	t.Run("should not dismiss the SQLite storage warning by default", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.False(t, bool(EnvConfig.DismissSQLiteStorageWarning))
	})

	t.Run("should not dismiss the SQLite storage warning with an unrelated value", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("DISMISS_SQLITE_STORAGE_WARNING", "true")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.False(t, bool(EnvConfig.DismissSQLiteStorageWarning))
	})

	t.Run("should dismiss the SQLite storage warning with the confirmation phrase", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("DISMISS_SQLITE_STORAGE_WARNING", "  I Accept The Risks  ")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.True(t, bool(EnvConfig.DismissSQLiteStorageWarning))
	})

	t.Run("should dismiss the SQLite storage warning with the confirmation phrase with dashes", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("DISMISS_SQLITE_STORAGE_WARNING", " I-Accept_The_Risks  ")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.True(t, bool(EnvConfig.DismissSQLiteStorageWarning))
	})

	t.Run("should fail when TLS key file does not exist", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")

		certFile := t.TempDir() + "/cert.pem"
		require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0600))
		t.Setenv("TLS_CERT_FILE", certFile)
		t.Setenv("TLS_KEY_FILE", "/nonexistent/key.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_KEY_FILE not found")
	})
}

func TestFrancisHostConfig(t *testing.T) {
	originalConfig := EnvConfig
	t.Cleanup(func() {
		EnvConfig = originalConfig
	})

	// setBaseEnv sets the variables every valid configuration needs, so each subtest only sets what it is about
	setBaseEnv := func(t *testing.T) {
		t.Helper()
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("CLUSTER_ADDRESS", "pocket-id.example.com")
	}

	t.Run("should default to the embedded runtime", func(t *testing.T) {
		setBaseEnv(t)

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, FrancisRuntimeEmbedded, EnvConfig.FrancisRuntimeAddress)
		assert.Empty(t, EnvConfig.francisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should keep the embedded runtime when FRANCIS_RUNTIME_ADDRESS is empty", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Empty(t, EnvConfig.francisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should keep the embedded runtime when FRANCIS_RUNTIME_ADDRESS is 'embedded'", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "EMBEDDED")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, FrancisRuntimeEmbedded, EnvConfig.FrancisRuntimeAddress) // lowercased
		assert.Empty(t, EnvConfig.francisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should parse a single remote runtime address", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, []string{"francis.example.com:8443"}, EnvConfig.francisAddresses)
		assert.False(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should parse a comma-separated list of remote runtime addresses", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "one.example.com:8443, two.example.com:8443 ,[2001:db8::1]:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, []string{"one.example.com:8443", "two.example.com:8443", "[2001:db8::1]:8443"}, EnvConfig.francisAddresses)
		assert.False(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should fail when a remote runtime address has no port", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid address 'francis.example.com' in FRANCIS_RUNTIME_ADDRESS")
	})

	t.Run("should fail when FRANCIS_RUNTIME_ADDRESS only contains separators", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", " , ")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FRANCIS_RUNTIME_ADDRESS does not contain any address")
	})

	t.Run("should fail when no bootstrap method is configured", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "one of FRANCIS_HOST_PSK or FRANCIS_HOST_JWT_FILE is required")
	})

	t.Run("should accept a bootstrap JWT file and keep its path", func(t *testing.T) {
		setBaseEnv(t)

		// The path is what gets stored, not the contents: Francis re-reads the file on every connection so a rotated token is picked up
		jwtFile := t.TempDir() + "/token"
		require.NoError(t, os.WriteFile(jwtFile, []byte("header.payload.signature"), 0600))

		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_JWT_FILE", jwtFile)

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, jwtFile, EnvConfig.FrancisHostJWTFile)
		assert.False(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should fail when the bootstrap JWT file does not exist", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_JWT_FILE", "/nonexistent/token")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FRANCIS_HOST_JWT_FILE not found")
	})

	t.Run("should fail when more than one bootstrap method is configured", func(t *testing.T) {
		setBaseEnv(t)

		jwtFile := t.TempDir() + "/token"
		require.NoError(t, os.WriteFile(jwtFile, []byte("header.payload.signature"), 0600))

		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")
		t.Setenv("FRANCIS_HOST_JWT_FILE", jwtFile)

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "only one host bootstrap method may be configured, but FRANCIS_HOST_PSK and FRANCIS_HOST_JWT_FILE are both set")
	})

	t.Run("should fail when the bootstrap PSK is too short", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK", "too-short")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FRANCIS_HOST_PSK must be at least 16 bytes long")
	})

	t.Run("should read the bootstrap PSK and the CA from files", func(t *testing.T) {
		setBaseEnv(t)

		tempDir := t.TempDir()
		pskFile := tempDir + "/psk"
		caFile := tempDir + "/ca.pem"
		caContent := "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
		// The trailing newline is what writing a secret to a file usually leaves behind, and it must not become part of the key
		require.NoError(t, os.WriteFile(pskFile, []byte("bootstrap-psk-that-is-long-enough\n"), 0600))
		require.NoError(t, os.WriteFile(caFile, []byte(caContent), 0600))

		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK_FILE", pskFile)
		t.Setenv("FRANCIS_CA_FILE", caFile)

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, []byte("bootstrap-psk-that-is-long-enough"), EnvConfig.FrancisHostPSK)
		assert.Equal(t, []byte(caContent), EnvConfig.FrancisCA)
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
	dbConnContent := "postgres://user:pass@localhost/testdb" // #nosec G101
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

	t.Run("should preserve TLS cert and key file paths", func(t *testing.T) {
		originalConfig := EnvConfig
		t.Cleanup(func() {
			EnvConfig = originalConfig
		})
		EnvConfig = defaultConfig()

		certFile := tempDir + "/cert.pem"
		certContent := "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
		err := os.WriteFile(certFile, []byte(certContent), 0600)
		require.NoError(t, err)

		keyFile := tempDir + "/key.pem"
		keyContent := "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
		err = os.WriteFile(keyFile, []byte(keyContent), 0600)
		require.NoError(t, err)

		t.Setenv("TLS_CERT_FILE", certFile)
		t.Setenv("TLS_KEY_FILE", keyFile)

		err = parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, certFile, EnvConfig.TLSCertFile)
		assert.Equal(t, keyFile, EnvConfig.TLSKeyFile)
	})

	t.Run("should preserve inline TLS cert and key data", func(t *testing.T) {
		originalConfig := EnvConfig
		t.Cleanup(func() {
			EnvConfig = originalConfig
		})
		EnvConfig = defaultConfig()

		certContent := "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
		keyContent := "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
		t.Setenv("TLS_CERT", certContent)
		t.Setenv("TLS_KEY", keyContent)

		err = parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, certContent, EnvConfig.TLSCert)
		assert.Equal(t, keyContent, EnvConfig.TLSKey)
	})
}

func TestIconLibraryConfig(t *testing.T) {
	originalConfig := EnvConfig
	t.Cleanup(func() {
		EnvConfig = originalConfig
	})

	tests := []struct {
		name        string
		value       *string
		wantURL     string
		wantEnabled bool
		wantErr     string
	}{
		{name: "defaults to the selfh.st collection", wantURL: DefaultIconLibraryURL, wantEnabled: true},
		{name: "empty value falls back to the default", value: new(""), wantURL: DefaultIconLibraryURL, wantEnabled: true},
		{name: "custom URL without the trailing slash", value: new("http://192.168.1.10:4050/icons/"), wantURL: "http://192.168.1.10:4050/icons", wantEnabled: true},
		{name: "disabled is case-insensitive", value: new("Disabled"), wantURL: URLDisabled, wantEnabled: false},
		{name: "rejects other schemes", value: new("ftp://mirror.example.com/icons"), wantErr: "ICON_LIBRARY_URL must be an HTTP or HTTPS URL"},
		{name: "rejects values without a host", value: new("icons"), wantErr: "ICON_LIBRARY_URL must be an HTTP or HTTPS URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			EnvConfig = defaultConfig()
			if tt.value != nil {
				t.Setenv("ICON_LIBRARY_URL", *tt.value)
			}

			err := parseAndValidateEnvConfig(t)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, EnvConfig.IconLibraryURL)
			assert.Equal(t, tt.wantEnabled, EnvConfig.IconLibraryEnabled())
		})
	}
}

func TestHAConfig(t *testing.T) {
	originalConfig := EnvConfig
	t.Cleanup(func() {
		EnvConfig = originalConfig
	})

	t.Run("rejects SQLite", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("HA_ENABLED", "true")
		t.Setenv("DB_CONNECTION_STRING", "file:test")

		err := parseAndValidateEnvConfig(t)
		require.ErrorContains(t, err, "high-availability mode requires a PostgreSQL database")
	})

	t.Run("accepts PostgreSQL", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("HA_ENABLED", "true")
		t.Setenv("DB_CONNECTION_STRING", "postgres://localhost/db")
		t.Setenv("CLUSTER_ADDRESS", "10.0.0.5")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
	})

	t.Run("rejects a loopback cluster address", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("HA_ENABLED", "true")
		t.Setenv("DB_CONNECTION_STRING", "postgres://localhost/db")
		t.Setenv("CLUSTER_ADDRESS", "127.0.0.1")

		err := parseAndValidateEnvConfig(t)
		require.ErrorContains(t, err, "but it is the loopback address '127.0.0.1'")
	})

	t.Run("detects the IPv4 address of the replica", func(t *testing.T) {
		setInterfaceAddrs(t, "127.0.0.1/8", "fe80::1/64", "2001:db8::5/64", "10.0.0.5/24")
		EnvConfig = defaultConfig()
		t.Setenv("HA_ENABLED", "true")
		t.Setenv("DB_CONNECTION_STRING", "postgres://localhost/db")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5", EnvConfig.ClusterAddress)
	})

	t.Run("refuses to guess between several addresses", func(t *testing.T) {
		setInterfaceAddrs(t, "10.0.0.5/24", "172.17.0.1/16")
		EnvConfig = defaultConfig()
		t.Setenv("HA_ENABLED", "true")
		t.Setenv("DB_CONNECTION_STRING", "postgres://localhost/db")

		err := parseAndValidateEnvConfig(t)
		require.ErrorContains(t, err, "this machine has several (10.0.0.5, 172.17.0.1): set CLUSTER_ADDRESS to one of them")
	})

	t.Run("detects the address with a standalone runtime", func(t *testing.T) {
		setInterfaceAddrs(t, "10.0.0.5/24")
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test")
		t.Setenv("FRANCIS_RUNTIME_ADDRESS", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5", EnvConfig.ClusterAddress)
	})
}

func setInterfaceAddrs(t *testing.T, cidrs ...string) {
	t.Helper()

	addrs := make([]net.Addr, len(cidrs))
	for i, cidr := range cidrs {
		ip, ipNet, err := net.ParseCIDR(cidr)
		require.NoError(t, err)
		ipNet.IP = ip
		addrs[i] = ipNet
	}

	original := interfaceAddrs
	interfaceAddrs = func() ([]net.Addr, error) { return addrs, nil }
	t.Cleanup(func() {
		interfaceAddrs = original
	})
}
