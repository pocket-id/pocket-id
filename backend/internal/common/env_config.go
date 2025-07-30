package common

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

func resolveStringOrFile(directValue string, filePath string, varName string, trim bool) (string, error) {
	if directValue != "" {
		return directValue, nil
	}
	if filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read secret '%s' from file '%s': %w", varName, filePath, err)
		}

		if trim {
			return strings.TrimSpace(string(content)), nil
		}
		return string(content), nil
	}
	return "", nil
}

type DbProvider string

const (
	// TracerName should be passed to otel.Tracer, trace.SpanFromContext when creating custom spans.
	TracerName = "github.com/pocket-id/pocket-id/backend/tracing"
	// MeterName should be passed to otel.Meter when create custom metrics.
	MeterName = "github.com/pocket-id/pocket-id/backend/metrics"
)

const (
	DbProviderSqlite        DbProvider = "sqlite"
	DbProviderPostgres      DbProvider = "postgres"
	MaxMindGeoLiteCityUrl   string     = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
	defaultSqliteConnString string     = "file:data/pocket-id.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(2500)&_txlock=immediate"
)

type EnvConfigSchema struct {
	AppEnv                 string     `env:"APP_ENV"`
	AppURL                 string     `env:"APP_URL"`
	DbProvider             DbProvider `env:"DB_PROVIDER"`
	DbConnectionString     string     `env:"DB_CONNECTION_STRING"`
	DbConnectionStringFile string     `env:"DB_CONNECTION_STRING_FILE"`
	UploadPath             string     `env:"UPLOAD_PATH"`
	KeysPath               string     `env:"KEYS_PATH"`
	KeysStorage            string     `env:"KEYS_STORAGE"`
	EncryptionKey          string     `env:"ENCRYPTION_KEY"`
	EncryptionKeyFile      string     `env:"ENCRYPTION_KEY_FILE"`
	Port                   string     `env:"PORT"`
	Host                   string     `env:"HOST"`
	UnixSocket             string     `env:"UNIX_SOCKET"`
	UnixSocketMode         string     `env:"UNIX_SOCKET_MODE"`
	MaxMindLicenseKey      string     `env:"MAXMIND_LICENSE_KEY"`
	MaxMindLicenseKeyFile  string     `env:"MAXMIND_LICENSE_KEY_FILE"`
	GeoLiteDBPath          string     `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl           string     `env:"GEOLITE_DB_URL"`
	LocalIPv6Ranges        string     `env:"LOCAL_IPV6_RANGES"`
	UiConfigDisabled       bool       `env:"UI_CONFIG_DISABLED"`
	MetricsEnabled         bool       `env:"METRICS_ENABLED"`
	TracingEnabled         bool       `env:"TRACING_ENABLED"`
	LogJSON                bool       `env:"LOG_JSON"`
	TrustProxy             bool       `env:"TRUST_PROXY"`
	AnalyticsDisabled      bool       `env:"ANALYTICS_DISABLED"`
}

var EnvConfig = defaultConfig()

func init() {
	err := parseEnvConfig()
	if err != nil {
		slog.Error("Configuration error", slog.Any("error", err))
		os.Exit(1)
	}
}

func defaultConfig() EnvConfigSchema {
	return EnvConfigSchema{
		AppEnv:             "production",
		DbProvider:         "sqlite",
		DbConnectionString: "",
		UploadPath:         "data/uploads",
		KeysPath:           "data/keys",
		KeysStorage:        "", // "database" or "file"
		EncryptionKey:      "",
		AppURL:             "http://localhost:1411",
		Port:               "1411",
		Host:               "0.0.0.0",
		UnixSocket:         "",
		UnixSocketMode:     "",
		MaxMindLicenseKey:  "",
		GeoLiteDBPath:      "data/GeoLite2-City.mmdb",
		GeoLiteDBUrl:       MaxMindGeoLiteCityUrl,
		LocalIPv6Ranges:    "",
		UiConfigDisabled:   false,
		MetricsEnabled:     false,
		TracingEnabled:     false,
		TrustProxy:         false,
		AnalyticsDisabled:  false,
	}
}

func parseEnvConfig() error {
	err := env.ParseWithOptions(&EnvConfig, env.Options{})
	if err != nil {
		return fmt.Errorf("error parsing env config: %w", err)
	}

	// Resolve string/file environment variables
	EnvConfig.DbConnectionString, err = resolveStringOrFile(
		EnvConfig.DbConnectionString,
		EnvConfig.DbConnectionStringFile,
		"DB_CONNECTION_STRING",
		true,
	)
	if err != nil {
		return err
	}
	EnvConfig.DbConnectionStringFile = ""

	EnvConfig.MaxMindLicenseKey, err = resolveStringOrFile(
		EnvConfig.MaxMindLicenseKey,
		EnvConfig.MaxMindLicenseKeyFile,
		"MAXMIND_LICENSE_KEY",
		true,
	)
	if err != nil {
		return err
	}
	EnvConfig.MaxMindLicenseKeyFile = ""

	// Validate the environment variables
	switch EnvConfig.DbProvider {
	case DbProviderSqlite:
		if EnvConfig.DbConnectionString == "" {
			EnvConfig.DbConnectionString = defaultSqliteConnString
		}
	case DbProviderPostgres:
		if EnvConfig.DbConnectionString == "" {
			return errors.New("missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}
	default:
		return errors.New("invalid DB_PROVIDER value. Must be 'sqlite' or 'postgres'")
	}

	parsedAppUrl, err := url.Parse(EnvConfig.AppURL)
	if err != nil {
		return errors.New("APP_URL is not a valid URL")
	}
	if parsedAppUrl.Path != "" {
		return errors.New("APP_URL must not contain a path")
	}

	switch EnvConfig.KeysStorage {
	// KeysStorage defaults to "file" if empty
	case "":
		EnvConfig.KeysStorage = "file"
	case "database":
		// Resolve encryption key using the same pattern
		encryptionKey, err := resolveStringOrFile(
			EnvConfig.EncryptionKey,
			EnvConfig.EncryptionKeyFile,
			"ENCRYPTION_KEY",
			// Do not trim spaces because the file should be interpreted as binary
			false,
		)
		if err != nil {
			return err
		}
		if encryptionKey == "" {
			return errors.New("ENCRYPTION_KEY or ENCRYPTION_KEY_FILE must be non-empty when KEYS_STORAGE is database")
		}
		// Update the config with resolved value
		EnvConfig.EncryptionKey = encryptionKey
		EnvConfig.EncryptionKeyFile = ""
	case "file":
		// All good, these are valid values
	default:
		return fmt.Errorf("invalid value for KEYS_STORAGE: %s", EnvConfig.KeysStorage)
	}

	return nil
}
