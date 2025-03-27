package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/resources"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDatabase() (db *gorm.DB) {
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
		dialector = sqlite.Open(parseSqliteConnString(common.EnvConfig.SqliteDBPath))
	case common.DbProviderPostgres:
		dialector = postgres.Open(common.EnvConfig.PostgresConnectionString)
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}

	for i := 1; i <= 3; i++ {
		db, err = gorm.Open(dialector, &gorm.Config{
			TranslateError: true,
			Logger:         getLogger(),
		})
		if err == nil {
			break
		} else {
			log.Printf("Attempt %d: Failed to initialize database. Retrying...", i)
			time.Sleep(3 * time.Second)
		}
	}

	return db, err
}

func parseSqliteConnString(connString string) string {
	// Inspired by, and adapted from: https://github.com/dapr/components-contrib/blob/v1.15.1/common/authentication/sqlite/metadata.go
	// Copyright (C) The Dapr Authors.
	// License: Apache2 http://www.apache.org/licenses/LICENSE-2.0

	// Default value for busy_timeout in ms
	const defaultBusyTimeout = 2500

	// Extract the query string if present
	idx := strings.IndexRune(connString, '?')
	var qs url.Values
	if idx > 0 {
		qs, _ = url.ParseQuery(connString[(idx + 1):])
	}
	if len(qs) == 0 {
		qs = make(url.Values, 2)
	}

	// Check if the database is read-only or immutable
	isReadOnly := false
	if len(qs["mode"]) > 0 {
		// Keep the first value only
		val := qs.Get("mode")
		qs.Set("mode", val)
		if val == "ro" {
			isReadOnly = true
		}
	}
	if len(qs["immutable"]) > 0 {
		// Keep the first value only
		val := qs.Get("immutable")
		qs.Set("immutable", val)
		if val == "1" {
			isReadOnly = true
		}
	}

	// We do not want to override a _txlock if set, but we'll show a warning if it's not "immediate"
	if len(qs["_txlock"]) > 0 {
		// Keep the first value only
		val := qs.Get("_txlock")
		qs.Set("_txlock", val)
		if val != "immediate" {
			log.Println("SQLite connection string is being created with a _txlock different from the recommended value 'immediate'")
		}
	} else {
		qs.Set("_txlock", "immediate")
	}

	// Add busy timeout if not present
	if len(qs["_busy_timeout"]) == 0 {
		qs.Set("_busy_timeout", strconv.Itoa(defaultBusyTimeout))
	}

	if len(qs["_journal_mode"]) == 0 {
		if isReadOnly {
			// Set the journaling mode to "DELETE" (the default) if the database is read-only
			qs.Set("_journal_mode", "DELETE")
		} else {
			// Enable WAL
			qs.Set("_journal_mode", "WAL")
		}
	}

	// Build the final connection string
	if idx > 0 {
		connString = connString[:idx]
	}
	connString += "?" + qs.Encode()

	// If the connection string doesn't begin with "file:", add the prefix
	if !strings.HasPrefix(strings.ToLower(connString), "file:") {
		connString = "file:" + connString
	}

	return connString
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
