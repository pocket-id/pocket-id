package common

import (
	"log"
	"net/url"

	"github.com/caarlos0/env/v11"
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
	DbProviderSqlite      DbProvider = "sqlite"
	DbProviderPostgres    DbProvider = "postgres"
	MaxMindGeoLiteCityUrl string     = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
)

type EnvConfigSchema struct {
	AppEnv                   string     `env:"APP_ENV"`
	AppURL                   string     `env:"PUBLIC_APP_URL"`
	DbProvider               DbProvider `env:"DB_PROVIDER"`
	DbConnectionString       string     `env:"DB_CONNECTION_STRING"`
	SqliteDBPath             string     `env:"SQLITE_DB_PATH"`             // Deprecated: use "DB_CONNECTION_STRING" instead
	PostgresConnectionString string     `env:"POSTGRES_CONNECTION_STRING"` // Deprecated: use "DB_CONNECTION_STRING" instead
	UploadPath               string     `env:"UPLOAD_PATH"`
	KeysPath                 string     `env:"KEYS_PATH"`
	Port                     string     `env:"PORT"`
	Host                     string     `env:"HOST"`
	MaxMindLicenseKey        string     `env:"MAXMIND_LICENSE_KEY"`
	GeoLiteDBPath            string     `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl             string     `env:"GEOLITE_DB_URL"`
	UiConfigDisabled         bool       `env:"UI_CONFIG_DISABLED"`
	MetricsEnabled           bool       `env:"METRICS_ENABLED"`
	TracingEnabled           bool       `env:"TRACING_ENABLED"`
	TrustProxy               bool       `env:"TRUST_PROXY"`
}

var EnvConfig = &EnvConfigSchema{
	AppEnv:                   "production",
	DbProvider:               "sqlite",
	DbConnectionString:       "file:data/pocket-id.db?_journal_mode=WAL&_busy_timeout=2500&_txlock=immediate",
	SqliteDBPath:             "",
	PostgresConnectionString: "",
	UploadPath:               "data/uploads",
	KeysPath:                 "data/keys",
	AppURL:                   "http://localhost",
	Port:                     "80",
	Host:                     "0.0.0.0",
	MaxMindLicenseKey:        "",
	GeoLiteDBPath:            "data/GeoLite2-City.mmdb",
	GeoLiteDBUrl:             MaxMindGeoLiteCityUrl,
	UiConfigDisabled:         false,
	MetricsEnabled:           false,
	TracingEnabled:           false,
	TrustProxy:               false,
}

func init() {
	if err := env.ParseWithOptions(EnvConfig, env.Options{}); err != nil {
		log.Fatal(err)
	}

	// Validate the environment variables
	switch EnvConfig.DbProvider {
	case DbProviderSqlite:
		if EnvConfig.DbConnectionString == "" {
			log.Fatal("Missing required env var 'DB_CONNECTION_STRING' for SQLite database")
		}
	case DbProviderPostgres:
		if EnvConfig.DbConnectionString == "" {
			log.Fatal("Missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}
	default:
		log.Fatal("Invalid DB_PROVIDER value. Must be 'sqlite' or 'postgres'")
	}

	parsedAppUrl, err := url.Parse(EnvConfig.AppURL)
	if err != nil {
		log.Fatal("PUBLIC_APP_URL is not a valid URL")
	}
	if parsedAppUrl.Path != "" {
		log.Fatal("PUBLIC_APP_URL must not contain a path")
	}
}
