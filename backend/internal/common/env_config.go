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
	AppEnv             string     `env:"APP_ENV"`
	AppURL             string     `env:"APP_URL"`
	DbProvider         DbProvider `env:"DB_PROVIDER"`
	DbConnectionString string     `env:"DB_CONNECTION_STRING"`
	UploadPath         string     `env:"UPLOAD_PATH"`
	KeysPath           string     `env:"KEYS_PATH"`
	Port               string     `env:"PORT"`
	Host               string     `env:"HOST"`
	UnixSocket         string     `env:"UNIX_SOCKET"`
	MaxMindLicenseKey  string     `env:"MAXMIND_LICENSE_KEY"`
	GeoLiteDBPath      string     `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl       string     `env:"GEOLITE_DB_URL"`
	UiConfigDisabled   bool       `env:"UI_CONFIG_DISABLED"`
	MetricsEnabled     bool       `env:"METRICS_ENABLED"`
	TracingEnabled     bool       `env:"TRACING_ENABLED"`
	TrustProxy         bool       `env:"TRUST_PROXY"`
	AnalyticsDisabled  bool       `env:"ANALYTICS_DISABLED"`
}

var EnvConfig = &EnvConfigSchema{
	AppEnv:             "production",
	DbProvider:         "sqlite",
	DbConnectionString: "file:data/pocket-id.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(2500)&_txlock=immediate",
	UploadPath:         "data/uploads",
	KeysPath:           "data/keys",
	AppURL:             "http://localhost:1411",
	Port:               "1411",
	Host:               "0.0.0.0",
	UnixSocket:         "",
	MaxMindLicenseKey:  "",
	GeoLiteDBPath:      "data/GeoLite2-City.mmdb",
	GeoLiteDBUrl:       MaxMindGeoLiteCityUrl,
	UiConfigDisabled:   false,
	MetricsEnabled:     false,
	TracingEnabled:     false,
	TrustProxy:         false,
	AnalyticsDisabled:  false,
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
		log.Fatal("APP_URL is not a valid URL")
	}
	if parsedAppUrl.Path != "" {
		log.Fatal("APP_URL must not contain a path")
	}
}
