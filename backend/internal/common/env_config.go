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

type AppEnv string
type DbProvider string

const (
	// TracerName should be passed to otel.Tracer, trace.SpanFromContext when creating custom spans.
	TracerName = "github.com/pocket-id/pocket-id/backend/tracing"
	// MeterName should be passed to otel.Meter when create custom metrics.
	MeterName = "github.com/pocket-id/pocket-id/backend/metrics"
)

const (
	AppEnvProduction        AppEnv     = "production"
	AppEnvDevelopment       AppEnv     = "development"
	AppEnvTest              AppEnv     = "test"
	DbProviderSqlite        DbProvider = "sqlite"
	DbProviderPostgres      DbProvider = "postgres"
	MaxMindGeoLiteCityUrl   string     = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
	defaultSqliteConnString string     = "data/pocket-id.db"
	defaultFsUploadPath     string     = "data/uploads"
	AppUrl                  string     = "http://localhost:1411"
)

type EnvConfigSchema struct {
	AppEnv                AppEnv `env:"APP_ENV" options:"toLower"`
	EncryptionKey         []byte `env:"ENCRYPTION_KEY" options:"file"`
	AppURL                string `env:"APP_URL" options:"toLower,trimTrailingSlash"`
	DbProvider            DbProvider
	DbConnectionString    string `env:"DB_CONNECTION_STRING" options:"file"`
	TrustProxy            bool   `env:"TRUST_PROXY"`
	TrustedPlatform       string `env:"TRUSTED_PLATFORM"`
	AuditLogRetentionDays int    `env:"AUDIT_LOG_RETENTION_DAYS"`
	AnalyticsDisabled     bool   `env:"ANALYTICS_DISABLED"`
	AllowDowngrade        bool   `env:"ALLOW_DOWNGRADE"`
	InternalAppURL        string `env:"INTERNAL_APP_URL"`
	UiConfigDisabled      bool   `env:"UI_CONFIG_DISABLED"`
	DisableRateLimiting   bool   `env:"DISABLE_RATE_LIMITING"`
	VersionCheckDisabled  bool   `env:"VERSION_CHECK_DISABLED"`
	StaticApiKey          string `env:"STATIC_API_KEY" options:"file"`

	FileBackend                     string `env:"FILE_BACKEND" options:"toLower"`
	UploadPath                      string `env:"UPLOAD_PATH"`
	S3Bucket                        string `env:"S3_BUCKET"`
	S3Region                        string `env:"S3_REGION"`
	S3Endpoint                      string `env:"S3_ENDPOINT"`
	S3AccessKeyID                   string `env:"S3_ACCESS_KEY_ID"`
	S3SecretAccessKey               string `env:"S3_SECRET_ACCESS_KEY"`
	S3ForcePathStyle                bool   `env:"S3_FORCE_PATH_STYLE"`
	S3DisableDefaultIntegrityChecks bool   `env:"S3_DISABLE_DEFAULT_INTEGRITY_CHECKS"`

	Port            string `env:"PORT"`
	Host            string `env:"HOST" options:"toLower"`
	UnixSocket      string `env:"UNIX_SOCKET"`
	UnixSocketMode  string `env:"UNIX_SOCKET_MODE"`
	LocalIPv6Ranges string `env:"LOCAL_IPV6_RANGES"`

	TLSCertFile string `env:"TLS_CERT" options:"file"`
	TLSKeyFile  string `env:"TLS_KEY" options:"file"`

	MaxMindLicenseKey string `env:"MAXMIND_LICENSE_KEY" options:"file"`
	GeoLiteDBPath     string `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl      string `env:"GEOLITE_DB_URL"`

	LogLevel       string `env:"LOG_LEVEL" options:"toLower"`
	MetricsEnabled bool   `env:"METRICS_ENABLED"`
	TracingEnabled bool   `env:"TRACING_ENABLED"`
	LogJSON        bool   `env:"LOG_JSON"`
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
		AppEnv:                AppEnvProduction,
		LogLevel:              "info",
		DbProvider:            "sqlite",
		FileBackend:           "filesystem",
		AuditLogRetentionDays: 90,
		AppURL:                AppUrl,
		Port:                  "1411",
		Host:                  "0.0.0.0",
		GeoLiteDBPath:         "data/GeoLite2-City.mmdb",
		GeoLiteDBUrl:          MaxMindGeoLiteCityUrl,
	}
}

func parseEnvConfig() error {
	parsers := map[reflect.Type]env.ParserFunc{
		reflect.TypeFor[[]byte](): func(value string) (any, error) {
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

	return nil

}

// ValidateEnvConfig checks the EnvConfig for required fields and valid values
func ValidateEnvConfig(config *EnvConfigSchema) error {
	if shouldSkipEnvValidation(os.Args) {
		return nil
	}

	if _, err := sloggin.ParseLevel(config.LogLevel); err != nil {
		return errors.New("invalid LOG_LEVEL value. Must be 'debug', 'info', 'warn' or 'error'")
	}

	if len(config.EncryptionKey) < 16 {
		return errors.New("ENCRYPTION_KEY must be at least 16 bytes long")
	}

	prepareDbConfig(config)

	if err := validateAppURLs(config); err != nil {
		return err
	}
	if err := validateFileBackend(config); err != nil {
		return err
	}
	if err := validateLocalIPv6Ranges(config.LocalIPv6Ranges); err != nil {
		return err
	}

	if config.AuditLogRetentionDays <= 0 {
		return errors.New("AUDIT_LOG_RETENTION_DAYS must be greater than 0")
	}

	if config.StaticApiKey != "" && len(config.StaticApiKey) < 16 {
		return errors.New("STATIC_API_KEY must be at least 16 characters long")
	}

	return validateTLSConfig(config)

}

func prepareDbConfig(config *EnvConfigSchema) {
	switch {
	case config.DbConnectionString == "":
		config.DbProvider = DbProviderSqlite
		config.DbConnectionString = defaultSqliteConnString
	case strings.HasPrefix(config.DbConnectionString, "postgres://") || strings.HasPrefix(config.DbConnectionString, "postgresql://"):
		config.DbProvider = DbProviderPostgres
	default:
		config.DbProvider = DbProviderSqlite
	}
}

func validateAppURLs(config *EnvConfigSchema) error {
	if err := validateURLWithoutPath(config.AppURL, "APP_URL"); err != nil {
		return err
	}

	// Derive INTERNAL_APP_URL from APP_URL if not set; validate only when provided
	if config.InternalAppURL == "" {
		config.InternalAppURL = config.AppURL
		return nil
	}

	return validateURLWithoutPath(config.InternalAppURL, "INTERNAL_APP_URL")
}

func validateURLWithoutPath(rawURL, envName string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL", envName)
	}
	if parsedURL.Path != "" {
		return fmt.Errorf("%s must not contain a path", envName)
	}

	return nil
}

func validateFileBackend(config *EnvConfigSchema) error {
	switch config.FileBackend {
	case "s3", "database":
		return nil
	case "", "filesystem":
		if config.UploadPath == "" {
			config.UploadPath = defaultFsUploadPath
		}
		return nil
	default:
		return errors.New("invalid FILE_BACKEND value. Must be 'filesystem', 'database', or 's3'")
	}
}

func validateLocalIPv6Ranges(localIPv6Ranges string) error {
	ranges := strings.SplitSeq(localIPv6Ranges, ",")
	for rangeStr := range ranges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}

		if err := validateLocalIPv6Range(rangeStr); err != nil {
			return err
		}
	}

	return nil
}

func validateLocalIPv6Range(rangeStr string) error {
	_, ipNet, err := net.ParseCIDR(rangeStr)
	if err != nil {
		return fmt.Errorf("invalid LOCAL_IPV6_RANGES '%s': %w", rangeStr, err)
	}

	if ipNet.IP.To4() != nil {
		return fmt.Errorf("range '%s' is not a valid IPv6 range", rangeStr)
	}

	return nil
}

func validateTLSConfig(config *EnvConfigSchema) error {
	switch {
	case config.TLSCertFile != "" && config.TLSKeyFile == "":
		return errors.New("TLS_KEY_FILE must be set when TLS_CERT_FILE is set")
	case config.TLSCertFile == "" && config.TLSKeyFile != "":
		return errors.New("TLS_CERT_FILE must be set when TLS_KEY_FILE is set")
	}

	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		if _, err := os.Stat(config.TLSCertFile); err != nil {
			return fmt.Errorf("TLS_CERT_FILE not found: %w", err)
		}
	}

	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		if _, err := os.Stat(config.TLSKeyFile); err != nil {
			return fmt.Errorf("TLS_KEY_FILE not found: %w", err)
		}
	}

	return nil

}

func shouldSkipEnvValidation(args []string) bool {
	for _, arg := range args[1:] {
		switch arg {
		case "-h", "--help", "help", "version":
			return true
		}
	}

	return false
}

// prepareEnvConfig processes special options for EnvConfig fields
func prepareEnvConfig(config *EnvConfigSchema) error {
	val := reflect.ValueOf(config).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		optionsTag := fieldType.Tag.Get("options")
		options := strings.SplitSeq(optionsTag, ",")

		for option := range options {
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

func (a AppEnv) IsProduction() bool {
	return a == AppEnvProduction
}

func (a AppEnv) IsTest() bool {
	return a == AppEnvTest
}
