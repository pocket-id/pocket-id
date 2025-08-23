package bootstrap

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	sqliteutil "github.com/pocket-id/pocket-id/backend/internal/utils/sqlite"
	"github.com/pocket-id/pocket-id/backend/resources"
)

func NewDatabase() (db *gorm.DB, err error) {
	db, err = connectDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
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
		return nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Run migrations
	if err := migrateDatabase(driver); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
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
		sqliteutil.RegisterSqliteFunctions()
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
			Logger:         getGormLogger(),
		})
		if err == nil {
			slog.Info("Connected to database", slog.String("provider", string(common.EnvConfig.DbProvider)))
			return db, nil
		}

		slog.Warn("Failed to connect to database, will retry in 3s", slog.Int("attempt", i), slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))
		time.Sleep(3 * time.Second)
	}

	slog.Error("Failed to connect to database after 3 attempts", slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))

	return nil, err
}

func parseSqliteConnectionString(connString string) (string, error) {
	if !strings.HasPrefix(connString, "file:") {
		connString = "file:" + connString
	}

	// Check if we're using an in-memory database
	isMemoryDB := isSqliteInMemory(connString)

	// Parse the connection string
	connStringUrl, err := url.Parse(connString)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQLite connection string: %w", err)
	}

	// Convert options for the old SQLite driver to the new one
	convertSqlitePragmaArgs(connStringUrl)

	// Add the default and required params
	err = addSqliteDefaultParameters(connStringUrl, isMemoryDB)
	if err != nil {
		return "", fmt.Errorf("invalid SQLite connection string: %w", err)
	}

	return connStringUrl.String(), nil
}

// The official C implementation of SQLite allows some additional properties in the connection string
// that are not supported in the in the modernc.org/sqlite driver, and which must be passed as PRAGMA args instead.
// To ensure that people can use similar args as in the C driver, which was also used by Pocket ID
// previously (via github.com/mattn/go-sqlite3), we are converting some options.
// Note this function updates connStringUrl.
func convertSqlitePragmaArgs(connStringUrl *url.URL) {
	// Reference: https://github.com/mattn/go-sqlite3?tab=readme-ov-file#connection-string
	// This only includes a subset of options, excluding those that are not relevant to us
	qs := make(url.Values, len(connStringUrl.Query()))
	for k, v := range connStringUrl.Query() {
		switch strings.ToLower(k) {
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

	// Update the connStringUrl object
	connStringUrl.RawQuery = qs.Encode()
}

// Adds the default (and some required) parameters to the SQLite connection string.
// Note this function updates connStringUrl.
func addSqliteDefaultParameters(connStringUrl *url.URL, isMemoryDB bool) error {
	// This function include code adapted from https://github.com/dapr/components-contrib/blob/v1.14.6/
	// Copyright (C) 2023 The Dapr Authors
	// License: Apache2
	const defaultBusyTimeout = 2500 * time.Millisecond

	// Get the "query string" from the connection string if present
	qs := connStringUrl.Query()
	if len(qs) == 0 {
		qs = make(url.Values, 2)
	}

	// If the database is in-memory, we must ensure that cache=shared is set
	if isMemoryDB {
		qs["cache"] = []string{"shared"}
	}

	// Check if the database is read-only or immutable
	isReadOnly := false
	if len(qs["mode"]) > 0 {
		// Keep the first value only
		qs["mode"] = []string{
			strings.ToLower(qs["mode"][0]),
		}
		if qs["mode"][0] == "ro" {
			isReadOnly = true
		}
	}
	if len(qs["immutable"]) > 0 {
		// Keep the first value only
		qs["immutable"] = []string{
			strings.ToLower(qs["immutable"][0]),
		}
		if qs["immutable"][0] == "1" {
			isReadOnly = true
		}
	}

	// We do not want to override a _txlock if set, but we'll show a warning if it's not "immediate"
	if len(qs["_txlock"]) > 0 {
		// Keep the first value only
		qs["_txlock"] = []string{
			strings.ToLower(qs["_txlock"][0]),
		}
		if qs["_txlock"][0] != "immediate" {
			slog.Warn("SQLite connection is being created with a _txlock different from the recommended value 'immediate'")
		}
	} else {
		qs["_txlock"] = []string{"immediate"}
	}

	// Add pragma values
	var hasBusyTimeout, hasJournalMode bool
	if len(qs["_pragma"]) == 0 {
		qs["_pragma"] = make([]string, 0, 3)
	} else {
		for _, p := range qs["_pragma"] {
			p = strings.ToLower(p)
			switch {
			case strings.HasPrefix(p, "busy_timeout"):
				hasBusyTimeout = true
			case strings.HasPrefix(p, "journal_mode"):
				hasJournalMode = true
			case strings.HasPrefix(p, "foreign_keys"):
				return errors.New("found forbidden option '_pragma=foreign_keys' in the connection string")
			}
		}
	}
	if !hasBusyTimeout {
		qs["_pragma"] = append(qs["_pragma"], fmt.Sprintf("busy_timeout(%d)", defaultBusyTimeout.Milliseconds()))
	}
	if !hasJournalMode {
		switch {
		case isMemoryDB:
			// For in-memory databases, set the journal to MEMORY, the only allowed option besides OFF (which would make transactions ineffective)
			qs["_pragma"] = append(qs["_pragma"], "journal_mode(MEMORY)")
		case isReadOnly:
			// Set the journaling mode to "DELETE" (the default) if the database is read-only
			qs["_pragma"] = append(qs["_pragma"], "journal_mode(DELETE)")
		default:
			// Enable WAL
			qs["_pragma"] = append(qs["_pragma"], "journal_mode(WAL)")
		}
	}

	// Forcefully enable foreign keys
	qs["_pragma"] = append(qs["_pragma"], "foreign_keys(1)")

	// Update the connStringUrl object
	connStringUrl.RawQuery = qs.Encode()

	return nil
}

// isSqliteInMemory returns true if the connection string is for an in-memory database.
func isSqliteInMemory(connString string) bool {
	lc := strings.ToLower(connString)

	// First way to define an in-memory database is to use ":memory:" or "file::memory:" as connection string
	if strings.HasPrefix(lc, ":memory:") || strings.HasPrefix(lc, "file::memory:") {
		return true
	}

	// Another way is to pass "mode=memory" in the "query string"
	idx := strings.IndexRune(lc, '?')
	if idx < 0 {
		return false
	}
	qs, _ := url.ParseQuery(lc[(idx + 1):])

	return len(qs["mode"]) > 0 && qs["mode"][0] == "memory"
}

func getGormLogger() gormLogger.Interface {
	loggerOpts := make([]slogGorm.Option, 0, 5)
	loggerOpts = append(loggerOpts,
		slogGorm.WithSlowThreshold(200*time.Millisecond),
		slogGorm.WithErrorField("error"),
	)

	if common.EnvConfig.AppEnv == "production" {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelWarn),
			slogGorm.WithIgnoreTrace(),
		)
	} else {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelDebug),
			slogGorm.WithRecordNotFoundError(),
			slogGorm.WithTraceAll(),
		)
	}

	return slogGorm.New(loggerOpts...)
}
