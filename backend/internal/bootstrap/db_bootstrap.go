package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/resources"
)

func NewDatabase() (db *gorm.DB) {
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}

	// Choose the correct driver for the database provider
	var driver database.Driver
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		driver, err = sqliteMigrate.WithInstance(sqlDb, &sqliteMigrate.Config{})
	case common.DbProviderPostgres:
		driver, err = postgresMigrate.WithInstance(sqlDb, &postgresMigrate.Config{})
	default:
		// Should never happen at this point
		log.Fatalf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}
	if err != nil {
		log.Fatalf("failed to create migration driver: %v", err)
	}

	// Run migrations
	if err := migrateDatabase(driver); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func migrateDatabase(driver database.Driver) error {
	// Use the embedded migrations
	source, err := iofs.New(resources.FS, "migrations/"+string(common.EnvConfig.DbProvider))
	if err != nil {
		return fmt.Errorf("failed to create embedded migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func connectDatabase() (db *gorm.DB, err error) {
	var dialector gorm.Dialector

	// Choose the correct database provider
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for SQLite database")
		}
		if !strings.HasPrefix(common.EnvConfig.DbConnectionString, "file:") {
			return nil, errors.New("invalid value for env var 'DB_CONNECTION_STRING': does not begin with 'file:'")
		}
		connString, err := parseSqliteConnectionString(common.EnvConfig.DbConnectionString)
		if err != nil {
			return nil, err
		}
		dialector = sqlite.Open(connString)
	case common.DbProviderPostgres:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}
		dialector = postgres.Open(common.EnvConfig.DbConnectionString)
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}

	for i := 1; i <= 3; i++ {
		db, err = gorm.Open(dialector, &gorm.Config{
			TranslateError: true,
			Logger:         getLogger(),
		})
		if err == nil {
			return db, nil
		}

		log.Printf("Attempt %d: Failed to initialize database. Retrying...", i)
		time.Sleep(3 * time.Second)
	}

	return nil, err
}

// The official C implementation of SQLite allows some additional properties in the connection string
// that are not supported in the in the modernc.org/sqlite driver, and which must be passed as PRAGMA args instead.
// To ensure that people can use similar args as in the C driver, which was also used by Pocket ID
// previously (via github.com/mattn/go-sqlite3), we are converting some options.
func parseSqliteConnectionString(connString string) (string, error) {
	if !strings.HasPrefix(connString, "file:") {
		connString = "file:" + connString
	}

	connStringUrl, err := url.Parse(connString)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQLite connection string: %w", err)
	}

	// Reference: https://github.com/mattn/go-sqlite3?tab=readme-ov-file#connection-string
	// This only includes a subset of options, excluding those that are not relevant to us
	qs := make(url.Values, len(connStringUrl.Query()))
	for k, v := range connStringUrl.Query() {
		switch k {
		case "_auto_vacuum", "_vacuum":
			qs.Add("_pragma", "auto_vacuum("+v[0]+")")
		case "_busy_timeout", "_timeout":
			qs.Add("_pragma", "busy_timeout("+v[0]+")")
		case "_case_sensitive_like", "_cslike":
			qs.Add("_pragma", "case_sensitive_like("+v[0]+")")
		case "_foreign_keys", "_fk":
			qs.Add("_pragma", "foreign_keys("+v[0]+")")
		case "_locking_mode", "_locking":
			qs.Add("_pragma", "locking_mode("+v[0]+")")
		case "_secure_delete":
			qs.Add("_pragma", "secure_delete("+v[0]+")")
		case "_synchronous", "_sync":
			qs.Add("_pragma", "synchronous("+v[0]+")")
		default:
			// Pass other query-string args as-is
			qs[k] = v
		}
	}

	connStringUrl.RawQuery = qs.Encode()

	return connStringUrl.String(), nil
}

func getLogger() logger.Interface {
	isProduction := common.EnvConfig.AppEnv == "production"

	var logLevel logger.LogLevel
	if isProduction {
		logLevel = logger.Error
	} else {
		logLevel = logger.Info
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: isProduction,
			ParameterizedQueries:      isProduction,
			Colorful:                  !isProduction,
		},
	)
}
