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
		t.Setenv("PROXY_PROTOCOL", "true")
		t.Setenv("ANALYTICS_DISABLED", "false")
		t.Setenv("ALLOW_INSECURE_CALLBACK_URLS", "false")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.True(t, EnvConfig.UiConfigDisabled)
		assert.Equal(t, TrustProxyConfig{"0.0.0.0/0", "::/0"}, EnvConfig.TrustProxy)
		assert.Equal(t, TrustProxyConfig{"0.0.0.0/0", "::/0"}, EnvConfig.ProxyProtocol)
		assert.False(t, EnvConfig.AnalyticsDisabled)
		assert.False(t, EnvConfig.AllowInsecureCallbackURLs)
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
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, 90, EnvConfig.AuditLogRetentionDays)
	})

	t.Run("should parse audit log retention days override", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("AUDIT_LOG_RETENTION_DAYS", "365")

		err := parseEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, 365, EnvConfig.AuditLogRetentionDays)
	})

	t.Run("should fail when AUDIT_LOG_RETENTION_DAYS is non-positive", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_PROVIDER", "sqlite")
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
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
		t.Setenv("ACTORS_PORT", "9999")
		t.Setenv("ACTORS_HOST", "LOCALHOST")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, AppEnvProduction, EnvConfig.AppEnv) // lowercased
		assert.Equal(t, "/custom/uploads", EnvConfig.UploadPath)
		assert.Equal(t, "8080", EnvConfig.Port)
		assert.Equal(t, "localhost", EnvConfig.Host) // lowercased
		assert.Equal(t, "9999", EnvConfig.ActorsPort)
		assert.Equal(t, "localhost", EnvConfig.ActorsHost) // lowercased
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

	t.Run("should fail when TLS cert is set without key", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT", "certificate")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_KEY must be set when TLS_CERT is set")
	})

	t.Run("should fail when TLS key is set without cert", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_KEY", "private key")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT must be set when TLS_KEY is set")
	})

	t.Run("should fail when TLS cert file is set without key file", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_KEY_FILE must be set when TLS_CERT_FILE is set")
	})

	t.Run("should fail when TLS key file is set without cert file", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_KEY_FILE", "/path/to/key.pem")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT_FILE must be set when TLS_KEY_FILE is set")
	})

	t.Run("should fail when inline and file TLS configuration are combined", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
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
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
		t.Setenv("TLS_CERT_FILE", "/nonexistent/cert.pem")

		keyFile := t.TempDir() + "/key.pem"
		require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0600))
		t.Setenv("TLS_KEY_FILE", keyFile)

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "TLS_CERT_FILE not found")
	})

	t.Run("should fail when TLS key file does not exist", func(t *testing.T) {
		EnvConfig = defaultConfig()
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
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
		t.Setenv("DB_CONNECTION_STRING", "file:test.db")
		t.Setenv("APP_URL", "http://localhost:3000")
	}

	t.Run("should default to the embedded runtime", func(t *testing.T) {
		setBaseEnv(t)

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, FrancisHostEmbedded, EnvConfig.FrancisHost)
		assert.Empty(t, EnvConfig.FrancisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should keep the embedded runtime when FRANCIS_HOST is empty", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Empty(t, EnvConfig.FrancisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should keep the embedded runtime when FRANCIS_HOST is 'embedded'", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "EMBEDDED")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, FrancisHostEmbedded, EnvConfig.FrancisHost) // lowercased
		assert.Empty(t, EnvConfig.FrancisAddresses)
		assert.True(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should parse a single remote runtime address", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "francis.example.com:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, []string{"francis.example.com:8443"}, EnvConfig.FrancisAddresses)
		assert.False(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should parse a comma-separated list of remote runtime addresses", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "one.example.com:8443, two.example.com:8443 ,[2001:db8::1]:8443")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.NoError(t, err)
		assert.Equal(t, []string{"one.example.com:8443", "two.example.com:8443", "[2001:db8::1]:8443"}, EnvConfig.FrancisAddresses)
		assert.False(t, EnvConfig.HasEmbeddedFrancisRuntime())
	})

	t.Run("should fail when a remote runtime address has no port", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "francis.example.com")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid address 'francis.example.com' in FRANCIS_HOST")
	})

	t.Run("should fail when FRANCIS_HOST only contains separators", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", " , ")
		t.Setenv("FRANCIS_HOST_PSK", "bootstrap-psk-that-is-long-enough")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FRANCIS_HOST does not contain any address")
	})

	t.Run("should fail when the bootstrap PSK is missing", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "francis.example.com:8443")

		err := parseAndValidateEnvConfig(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "FRANCIS_HOST_PSK is required")
	})

	t.Run("should fail when the bootstrap PSK is too short", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("FRANCIS_HOST", "francis.example.com:8443")
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

		t.Setenv("FRANCIS_HOST", "francis.example.com:8443")
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
