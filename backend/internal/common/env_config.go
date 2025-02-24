package common

import (
	"log"
	"net/url"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

type DbProvider string

const (
	DbProviderSqlite      DbProvider = "sqlite"
	DbProviderPostgres    DbProvider = "postgres"
	MaxMindGeoLiteCityUrl string     = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
)

type EnvConfigSchema struct {
	AppEnv                   string     `env:"APP_ENV"`
	AppURL                   string     `env:"PUBLIC_APP_URL"`
	TrustedProxies           []string   `env:"TRUSTED_PROXIES"`
	DbProvider               DbProvider `env:"DB_PROVIDER"`
	SqliteDBPath             string     `env:"SQLITE_DB_PATH"`
	PostgresConnectionString string     `env:"POSTGRES_CONNECTION_STRING"`
	UploadPath               string     `env:"UPLOAD_PATH"`
	Port                     string     `env:"BACKEND_PORT"`
	Host                     string     `env:"HOST"`
	MaxMindLicenseKey        string     `env:"MAXMIND_LICENSE_KEY"`
	GeoLiteDBPath            string     `env:"GEOLITE_DB_PATH"`
	GeoLiteDBUrl             string     `env:"GEOLITE_DB_URL"`
	UiConfigDisabled         bool       `env:"PUBLIC_UI_CONFIG_DISABLED"`
}

var EnvConfig = &EnvConfigSchema{
	AppEnv:                   "production",
	DbProvider:               "sqlite",
	SqliteDBPath:             "data/pocket-id.db",
	PostgresConnectionString: "",
	UploadPath:               "data/uploads",
	AppURL:                   "http://localhost",
	Port:                     "8080",
	Host:                     "0.0.0.0",
	MaxMindLicenseKey:        "",
	GeoLiteDBPath:            "data/GeoLite2-City.mmdb",
	GeoLiteDBUrl:             MaxMindGeoLiteCityUrl,
	UiConfigDisabled:         false,
}

func init() {
	if err := env.ParseWithOptions(EnvConfig, env.Options{}); err != nil {
		log.Fatal(err)
	}
	// Validate the environment variables
	if EnvConfig.DbProvider != DbProviderSqlite && EnvConfig.DbProvider != DbProviderPostgres {
		log.Fatal("Invalid DB_PROVIDER value. Must be 'sqlite' or 'postgres'")
	}

	if EnvConfig.DbProvider == DbProviderPostgres && EnvConfig.PostgresConnectionString == "" {
		log.Fatal("Missing POSTGRES_CONNECTION_STRING environment variable")
	}

	if EnvConfig.DbProvider == DbProviderSqlite && EnvConfig.SqliteDBPath == "" {
		log.Fatal("Missing SQLITE_DB_PATH environment variable")
	}

	parsedAppUrl, err := url.Parse(EnvConfig.AppURL)
	if err != nil {
		log.Fatal("PUBLIC_APP_URL is not a valid URL")
	}
	if parsedAppUrl.Path != "" {
		log.Fatal("PUBLIC_APP_URL must not contain a path")
	}

	// No need to validate TrustedProxies, it will do Gin
}
