package common

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	sloggin "github.com/gin-contrib/slog"
	_ "github.com/joho/godotenv/autoload"
)

type (
	AppEnv                            string
	DbProvider                        string
	TrustProxyConfig                  []string
	DismissSQLiteStorageWarningConfig bool
)

const (
	// TracerName should be passed to otel.Tracer, trace.SpanFromContext when creating custom spans.
	TracerName = "github.com/pocket-id/pocket-id/backend/tracing"
	// MeterName should be passed to otel.Meter when create custom metrics.
	MeterName = "github.com/pocket-id/pocket-id/backend/metrics"
	// dismissSQLiteStorageWarningPhrase is the exact phrase DISMISS_SQLITE_STORAGE_WARNING must be set to to suppress the warning (matched case-insensitive)
	dismissSQLiteStorageWarningPhrase = "i accept the risks"
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

	// FrancisHostEmbedded is the FRANCIS_HOST value that keeps the Francis actor runtime embedded in the Pocket ID process
	FrancisHostEmbedded string = "embedded"
	// francisHostPSKMinLength is the shortest host bootstrap pre-shared key Francis accepts
	francisHostPSKMinLength int = 16
)

type EnvConfigSchema struct {
	AppEnv                    AppEnv `env:"APP_ENV" options:"toLower"`
	EncryptionKey             []byte `env:"ENCRYPTION_KEY" options:"file"`
	AppURL                    string `env:"APP_URL" options:"toLower,trimTrailingSlash"`
	DbProvider                DbProvider
	DbConnectionString        string           `env:"DB_CONNECTION_STRING" options:"file"`
	TrustProxy                TrustProxyConfig `env:"TRUST_PROXY"`
	ProxyProtocol             TrustProxyConfig `env:"PROXY_PROTOCOL"`
	TrustedPlatform           string           `env:"TRUSTED_PLATFORM"`
	AuditLogRetentionDays     int              `env:"AUDIT_LOG_RETENTION_DAYS"`
	AnalyticsDisabled         bool             `env:"ANALYTICS_DISABLED"`
	AllowDowngrade            bool             `env:"ALLOW_DOWNGRADE"`
	AllowInsecureCallbackURLs bool             `env:"ALLOW_INSECURE_CALLBACK_URLS"`
	InternalAppURL            string           `env:"INTERNAL_APP_URL"`
	UiConfigDisabled          bool             `env:"UI_CONFIG_DISABLED"`
	DisableRateLimiting       bool             `env:"DISABLE_RATE_LIMITING"`
	VersionCheckDisabled      bool             `env:"VERSION_CHECK_DISABLED"`
	StaticApiKey              string           `env:"STATIC_API_KEY" options:"file"`

	FileBackend                     string `env:"FILE_BACKEND" options:"toLower"`
	UploadPath                      string `env:"UPLOAD_PATH"`
	S3Bucket                        string `env:"S3_BUCKET"`
	S3Region                        string `env:"S3_REGION"`
	S3Endpoint                      string `env:"S3_ENDPOINT"`
	S3AccessKeyID                   string `env:"S3_ACCESS_KEY_ID"`
	S3SecretAccessKey               string `env:"S3_SECRET_ACCESS_KEY" options:"file"`
	S3ForcePathStyle                bool   `env:"S3_FORCE_PATH_STYLE"`
	S3DisableDefaultIntegrityChecks bool   `env:"S3_DISABLE_DEFAULT_INTEGRITY_CHECKS"`

	Port            string `env:"PORT"`
	Host            string `env:"HOST" options:"toLower"`
	UnixSocket      string `env:"UNIX_SOCKET"`
	UnixSocketMode  string `env:"UNIX_SOCKET_MODE"`
	SystemdSocket   bool   `env:"SYSTEMD_SOCKET"`
	LocalIPv6Ranges string `env:"LOCAL_IPV6_RANGES"`

	// TLS cert and key need special treatment with fsnotify, so we aren't using `options:"file"`
	TLSCert     string `env:"TLS_CERT"`
	TLSKey      string `env:"TLS_KEY"`
	TLSCertFile string `env:"TLS_CERT_FILE"`
	TLSKeyFile  string `env:"TLS_KEY_FILE"`

	MaxMindLicenseKey string `env:"MAXMIND_LICENSE_KEY" options:"file"`
	GeoLiteDBPath     string `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl      string `env:"GEOLITE_DB_URL"`

	ActorsPort string `env:"ACTORS_PORT"`
	ActorsHost string `env:"ACTORS_HOST" options:"toLower"`

	// FrancisHost selects where the Francis actor runtime lives
	// When empty or set to "embedded", Pocket ID runs the runtime inside its own process, which is the default
	// Any other value is the address (or a comma-separated list of addresses) of a standalone Francis runtime to connect to, and in that case Pocket ID does not start an embedded runtime
	FrancisHost string `env:"FRANCIS_HOST" options:"toLower"`
	// FrancisHostPSK is the pre-shared key Pocket ID presents to a standalone Francis runtime when joining the cluster
	// It must match the "bootstrap.hostPSK" value in the runtime's own configuration
	// It is one of the three ways to authenticate to the runtime, and exactly one of them is required whenever FrancisHost points to a standalone runtime
	FrancisHostPSK []byte `env:"FRANCIS_HOST_PSK" options:"file"`
	// FrancisHostJWT is the bearer token Pocket ID presents to a standalone Francis runtime configured for JWT bootstrap
	// Prefer FrancisHostJWTFile in production, since a token passed inline cannot be rotated without restarting Pocket ID
	FrancisHostJWT string `env:"FRANCIS_HOST_JWT"`
	// FrancisHostJWTFile is the path to a file holding the bearer token Pocket ID presents to a standalone Francis runtime
	// Unlike the other "_FILE" variables this one keeps the path rather than the contents: the file is re-read on every connection to the runtime, so a rotated token (such as a Kubernetes projected service account token) is picked up without restarting Pocket ID
	FrancisHostJWTFile string `env:"FRANCIS_HOST_JWT_FILE"`
	// FrancisCA is the PEM-encoded cluster CA of a standalone Francis runtime, which Pocket ID pins before its first connection
	// Leaving it empty makes Pocket ID trust the certificate the runtime presents on the first connection, which is vulnerable to an attacker intercepting that connection
	FrancisCA []byte `env:"FRANCIS_CA" options:"file"`
	// FrancisAddresses contains the runtime addresses parsed out of FrancisHost, and is empty when the actor runtime is embedded
	// It is derived from FrancisHost and is not bound to an environment variable of its own
	FrancisAddresses []string

	// HAEnabled turns on high-availability mode, allowing more than one replica of Pocket ID to run against the same database at once
	// It is intentionally not bound to an environment variable while HA support is still being completed
	// TODO: Add env var when HA mode is ready
	HAEnabled bool

	LogLevel string `env:"LOG_LEVEL" options:"toLower"`
	LogJSON  bool   `env:"LOG_JSON"`

	// LogQueryArgs includes the values of SQL query parameters in traces and in the query logs printed when LogLevel is "debug"
	// Note that these may can contain sensitive data
	LogQueryArgs bool `env:"LOG_QUERY_ARGS"`

	// This is true when DISMISS_SQLITE_STORAGE_WARNING is the exact confirmation phrase set in the constant above
	// Note: this is omitted from the general list of environment variables in the docs, and documented only in the SQLite-specific section
	DismissSQLiteStorageWarning DismissSQLiteStorageWarningConfig `env:"DISMISS_SQLITE_STORAGE_WARNING"`
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
		AppEnv:                    AppEnvProduction,
		LogLevel:                  "info",
		DbProvider:                "sqlite",
		FileBackend:               "filesystem",
		AuditLogRetentionDays:     90,
		AllowInsecureCallbackURLs: true, // TODO: Default to false in major v3
		AppURL:                    AppUrl,
		Port:                      "1411",
		Host:                      "0.0.0.0",
		ActorsPort:                "1414",
		ActorsHost:                "0.0.0.0",
		FrancisHost:               FrancisHostEmbedded,
		GeoLiteDBPath:             "data/GeoLite2-City.mmdb",
		GeoLiteDBUrl:              MaxMindGeoLiteCityUrl,
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

	_, err := sloggin.ParseLevel(config.LogLevel)
	if err != nil {
		return errors.New("invalid LOG_LEVEL value. Must be 'debug', 'info', 'warn' or 'error'")
	}

	// Check required properties
	if len(config.EncryptionKey) < 16 {
		return errors.New("ENCRYPTION_KEY must be at least 16 bytes long")
	}

	if config.SystemdSocket && config.UnixSocket != "" {
		return errors.New("SYSTEMD_SOCKET and UNIX_SOCKET are mutually exclusive")
	}
	if len(config.ProxyProtocol) > 0 && config.UnixSocket != "" {
		return errors.New("PROXY_PROTOCOL and UNIX_SOCKET are mutually exclusive")
	}

	if config.AuditLogRetentionDays <= 0 {
		return errors.New("AUDIT_LOG_RETENTION_DAYS must be greater than 0")
	}

	if config.StaticApiKey != "" && len(config.StaticApiKey) < 16 {
		return errors.New("when set, STATIC_API_KEY must be at least 16 characters long")
	}

	// Prepare the DB config
	prepareDbConfig(config)

	// Resolve where the Francis actor runtime lives, which decides whether Pocket ID starts an embedded one
	err = prepareFrancisConfig(config)
	if err != nil {
		return err
	}

	// Validate other required options
	err = validateAppURLs(config)
	if err != nil {
		return err
	}

	err = validateFileBackend(config)
	if err != nil {
		return err
	}

	err = validateLocalIPv6Ranges(config.LocalIPv6Ranges)
	if err != nil {
		return err
	}

	err = validateTLSConfig(config)
	if err != nil {
		return err
	}

	return nil
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

// prepareFrancisConfig resolves FRANCIS_HOST into the list of standalone runtime addresses Pocket ID connects to
// An empty value or the "embedded" constant keeps the actor runtime inside the Pocket ID process, and leaves the address list empty
func prepareFrancisConfig(config *EnvConfigSchema) error {
	config.FrancisAddresses = nil

	value := strings.TrimSpace(config.FrancisHost)
	if value == "" || value == FrancisHostEmbedded {
		return nil
	}

	// Any other value is one address, or a comma-separated list of addresses, of the standalone runtime replicas
	// Pocket ID dials them directly rather than resolving a service record, so each one must carry an explicit port
	parts := strings.Split(value, ",")
	addresses := make([]string, 0, len(parts))
	for _, address := range parts {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}

		_, port, err := net.SplitHostPort(address)
		if err != nil || port == "" {
			return fmt.Errorf("invalid address '%s' in FRANCIS_HOST: addresses must be in the 'host:port' format", address)
		}

		addresses = append(addresses, address)
	}

	if len(addresses) == 0 {
		return errors.New("FRANCIS_HOST does not contain any address")
	}

	// Credentials have to match the runtime's byte-for-byte, and reading one from a file (including a container secret) usually leaves a trailing newline behind, so surrounding whitespace is never meaningful here
	config.FrancisHostPSK = bytes.TrimSpace(config.FrancisHostPSK)
	config.FrancisHostJWT = strings.TrimSpace(config.FrancisHostJWT)

	err := validateFrancisBootstrap(config)
	if err != nil {
		return err
	}

	config.FrancisAddresses = addresses

	return nil
}

// validateFrancisBootstrap checks the credential Pocket ID presents when joining a standalone Francis runtime
// The runtime admits a host through exactly one bootstrap method, so configuring none or more than one is a configuration error rather than something to resolve by picking a winner
func validateFrancisBootstrap(config *EnvConfigSchema) error {
	configured := make([]string, 0, 3)
	if len(config.FrancisHostPSK) > 0 {
		configured = append(configured, "FRANCIS_HOST_PSK")
	}
	if config.FrancisHostJWT != "" {
		configured = append(configured, "FRANCIS_HOST_JWT")
	}
	if config.FrancisHostJWTFile != "" {
		configured = append(configured, "FRANCIS_HOST_JWT_FILE")
	}

	switch len(configured) {
	case 1:
		// Exactly one method, which is what the runtime expects
	case 0:
		return errors.New("one of FRANCIS_HOST_PSK, FRANCIS_HOST_JWT, or FRANCIS_HOST_JWT_FILE is required when FRANCIS_HOST points to a standalone Francis runtime")
	default:
		return fmt.Errorf("only one host bootstrap method may be configured, but %s are all set", strings.Join(configured, ", "))
	}

	// Francis rejects a shorter key, so checking the length here turns that into a configuration error at startup
	if len(config.FrancisHostPSK) > 0 && len(config.FrancisHostPSK) < francisHostPSKMinLength {
		return fmt.Errorf("FRANCIS_HOST_PSK must be at least %d bytes long", francisHostPSKMinLength)
	}

	// A token read on every connection is useless if the file is not there when Pocket ID starts
	if config.FrancisHostJWTFile != "" {
		_, err := os.Stat(config.FrancisHostJWTFile)
		if err != nil {
			return fmt.Errorf("FRANCIS_HOST_JWT_FILE not found: %w", err)
		}
	}

	return nil
}

// HasEmbeddedFrancisRuntime returns true when Pocket ID runs the Francis actor runtime inside its own process, which is the case unless FRANCIS_HOST points to a standalone runtime
func (config *EnvConfigSchema) HasEmbeddedFrancisRuntime() bool {
	return len(config.FrancisAddresses) == 0
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
	inlineConfigured := config.TLSCert != "" || config.TLSKey != ""
	fileConfigured := config.TLSCertFile != "" || config.TLSKeyFile != ""

	if inlineConfigured && fileConfigured {
		return errors.New("TLS_CERT and TLS_KEY cannot be combined with TLS_CERT_FILE or TLS_KEY_FILE")
	}

	switch {
	case config.TLSCert != "" && config.TLSKey == "":
		return errors.New("TLS_KEY must be set when TLS_CERT is set")
	case config.TLSCert == "" && config.TLSKey != "":
		return errors.New("TLS_CERT must be set when TLS_KEY is set")
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

	// #nosec G703 - Path is passed by the admin
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

func (config *DismissSQLiteStorageWarningConfig) UnmarshalText(text []byte) error {
	// Make lowercase, then replace all - and _ with spaces
	value := strings.ToLower(strings.TrimSpace(string(text)))
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	*config = DismissSQLiteStorageWarningConfig(value == dismissSQLiteStorageWarningPhrase)
	return nil
}

func (config *TrustProxyConfig) UnmarshalText(text []byte) error {
	value := strings.TrimSpace(string(text))

	// Support boolean values for completely enabling or disabling trust proxy
	enabled, err := strconv.ParseBool(value)
	if err == nil {
		if enabled {
			*config = TrustProxyConfig{"0.0.0.0/0", "::/0"}
		} else {
			*config = nil
		}

		return nil
	}

	// Normalize and validate each explicit proxy before the server starts
	proxies := strings.Split(value, ",")
	for i, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return fmt.Errorf("invalid proxy IP address or CIDR %q", proxy)
			}
		}
		proxies[i] = proxy
	}

	*config = proxies
	return nil
}
