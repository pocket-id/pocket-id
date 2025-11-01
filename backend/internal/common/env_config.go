package common

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
	sloggin "github.com/gin-contrib/slog"
	_ "github.com/joho/godotenv/autoload"
)

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
	defaultSqliteConnString string     = "data/pocket-id.db"
	AppUrl                  string     = "http://localhost:1411"
)

type EnvConfigSchema struct {
	AppEnv             string     `env:"APP_ENV" options:"toLower"`
	LogLevel           string     `env:"LOG_LEVEL" options:"toLower"`
	AppURL             string     `env:"APP_URL" options:"toLower,trimTrailingSlash"`
	DbProvider         DbProvider `env:"DB_PROVIDER" options:"toLower"`
	DbConnectionString string     `env:"DB_CONNECTION_STRING" options:"file"`
	UploadPath         string     `env:"UPLOAD_PATH"`
	KeysPath           string     `env:"KEYS_PATH"`
	KeysStorage        string     `env:"KEYS_STORAGE"`
	EncryptionKey      []byte     `env:"ENCRYPTION_KEY" options:"file"`
	Port               string     `env:"PORT"`
	Host               string     `env:"HOST" options:"toLower"`
	UnixSocket         string     `env:"UNIX_SOCKET"`
	UnixSocketMode     string     `env:"UNIX_SOCKET_MODE"`
	MaxMindLicenseKey  string     `env:"MAXMIND_LICENSE_KEY" options:"file"`
	GeoLiteDBPath      string     `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl       string     `env:"GEOLITE_DB_URL"`
	LocalIPv6Ranges    string     `env:"LOCAL_IPV6_RANGES"`
	UiConfigDisabled   bool       `env:"UI_CONFIG_DISABLED"`
	MetricsEnabled     bool       `env:"METRICS_ENABLED"`
	TracingEnabled     bool       `env:"TRACING_ENABLED"`
	LogJSON            bool       `env:"LOG_JSON"`
	TrustProxy         bool       `env:"TRUST_PROXY"`
	AnalyticsDisabled  bool       `env:"ANALYTICS_DISABLED"`
	AllowDowngrade     bool       `env:"ALLOW_DOWNGRADE"`
	InternalAppURL     string     `env:"INTERNAL_APP_URL"`
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
		LogLevel:           "info",
		DbProvider:         "sqlite",
		DbConnectionString: "",
		UploadPath:         "data/uploads",
		KeysPath:           "data/keys",
		KeysStorage:        "", // "database" or "file"
		EncryptionKey:      nil,
		AppURL:             AppUrl,
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
		AllowDowngrade:     false,
		InternalAppURL:     "",
	}
}

func parseEnvConfig() error {
	parsers := map[reflect.Type]env.ParserFunc{
		reflect.TypeOf([]byte{}): func(value string) (interface{}, error) {
			return []byte(value), nil
		},
	}

	err := env.ParseWithOptions(&EnvConfig, env.Options{
		FuncMap: parsers,
	})
	if err != nil {
		return fmt.Errorf("error parsing env config: %w", err)
	}

	err = prepareEnvConfig(&EnvConfig)
	if err != nil {
		return fmt.Errorf("error preparing env config: %w", err)
	}

	err = validateEnvConfig(&EnvConfig)
	if err != nil {
		return err
	}

	return nil

}

// validateEnvConfig checks the EnvConfig for required fields and valid values
func validateEnvConfig(config *EnvConfigSchema) error {
	if _, err := sloggin.ParseLevel(config.LogLevel); err != nil {
		return errors.New("invalid LOG_LEVEL value. Must be 'debug', 'info', 'warn' or 'error'")
	}

	switch config.DbProvider {
	case DbProviderSqlite:
		if config.DbConnectionString == "" {
			config.DbConnectionString = defaultSqliteConnString
		}
	case DbProviderPostgres:
		if config.DbConnectionString == "" {
			return errors.New("missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}
	default:
		return errors.New("invalid DB_PROVIDER value. Must be 'sqlite' or 'postgres'")
	}

	parsedAppUrl, err := url.Parse(config.AppURL)
	if err != nil {
		return errors.New("APP_URL is not a valid URL")
	}
	if parsedAppUrl.Path != "" {
		return errors.New("APP_URL must not contain a path")
	}

	// Derive INTERNAL_APP_URL from APP_URL if not set; validate only when provided
	if config.InternalAppURL == "" {
		config.InternalAppURL = config.AppURL
	} else {
		parsedInternalAppUrl, err := url.Parse(config.InternalAppURL)
		if err != nil {
			return errors.New("INTERNAL_APP_URL is not a valid URL")
		}
		if parsedInternalAppUrl.Path != "" {
			return errors.New("INTERNAL_APP_URL must not contain a path")
		}
	}

	switch config.KeysStorage {
	// KeysStorage defaults to "file" if empty
	case "":
		config.KeysStorage = "file"
	case "database":
		if config.EncryptionKey == nil {
			return errors.New("ENCRYPTION_KEY must be non-empty when KEYS_STORAGE is database")
		}
	case "file":
		// All good, these are valid values
	default:
		return fmt.Errorf("invalid value for KEYS_STORAGE: %s", config.KeysStorage)
	}

	// Validate LOCAL_IPV6_RANGES
	ranges := strings.Split(config.LocalIPv6Ranges, ",")
	for _, rangeStr := range ranges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}

		_, ipNet, err := net.ParseCIDR(rangeStr)
		if err != nil {
			return fmt.Errorf("invalid LOCAL_IPV6_RANGES '%s': %w", rangeStr, err)
		}

		if ipNet.IP.To4() != nil {
			return fmt.Errorf("range '%s' is not a valid IPv6 range", rangeStr)
		}

	}

	return nil

}

// prepareEnvConfig processes special options for EnvConfig fields
func prepareEnvConfig(config *EnvConfigSchema) error {
	val := reflect.ValueOf(config).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		optionsTag := fieldType.Tag.Get("options")
		options := strings.Split(optionsTag, ",")

		for _, option := range options {
			switch option {
			case "toLower":
				if field.Kind() == reflect.String {
					field.SetString(strings.ToLower(field.String()))
				}
			case "file":
				err := resolveFileBasedEnvVariable(field, fieldType)
				if err != nil {
					return err
				}
			case "trimTrailingSlash":
				if field.Kind() == reflect.String {
					field.SetString(strings.TrimRight(field.String(), "/"))
				}
			}
		}
	}

	return nil
}

// resolveFileBasedEnvVariable checks if an environment variable with the suffix "_FILE" is set,
// reads the content of the file specified by that variable, and sets the corresponding field's value.
func resolveFileBasedEnvVariable(field reflect.Value, fieldType reflect.StructField) error {
	// Only process string and []byte fields
	isString := field.Kind() == reflect.String
	isByteSlice := field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8
	if !isString && !isByteSlice {
		return nil
	}

	// Only process fields with the "env" tag
	envTag := fieldType.Tag.Get("env")
	if envTag == "" {
		return nil
	}

	envVarName := envTag
	if commaIndex := len(envTag); commaIndex > 0 {
		envVarName = envTag[:commaIndex]
	}

	// If the file environment variable is not set, skip
	envVarFileName := envVarName + "_FILE"
	envVarFileValue := os.Getenv(envVarFileName)
	if envVarFileValue == "" {
		return nil
	}

	fileContent, err := os.ReadFile(envVarFileValue)
	if err != nil {
		return fmt.Errorf("failed to read file for env var %s: %w", envVarFileName, err)
	}

	if isString {
		field.SetString(strings.TrimSpace(string(fileContent)))
	} else {
		field.SetBytes(fileContent)
	}

	return nil
}
