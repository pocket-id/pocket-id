package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	sqliteutil "github.com/pocket-id/pocket-id/backend/internal/utils/sqlite"
)

func NewDatabase() (db *gorm.DB, err error) {
	db, err = ConnectDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Run migrations
	if err := utils.MigrateDatabase(sqlDb); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func ConnectDatabase() (db *gorm.DB, err error) {
	var dialector gorm.Dialector

	// Choose the correct database provider
	var onConnFn func(conn *sql.DB)
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for SQLite database")
		}

		sqliteutil.RegisterSqliteFunctions()

		connString, dbPath, isMemoryDB, err := parseSqliteConnectionString(common.EnvConfig.DbConnectionString)
		if err != nil {
			return nil, err
		}

		// Before we connect, also make sure that there's a temporary folder for SQLite to write its data
		err = ensureSqliteTempDir(filepath.Dir(dbPath))
		if err != nil {
			return nil, err
		}

		if isMemoryDB {
			// For in-memory SQLite databases, we must limit to 1 open connection at the same time, or they won't see the whole data
			// The other workaround, of using shared caches, doesn't work well with multiple write transactions trying to happen at once
			onConnFn = func(conn *sql.DB) {
				conn.SetMaxOpenConns(1)
			}
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

			if onConnFn != nil {
				conn, err := db.DB()
				if err != nil {
					slog.Warn("Failed to get database connection, will retry in 3s", slog.Int("attempt", i), slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))
					time.Sleep(3 * time.Second)
				}
				onConnFn(conn)
			}

			return db, nil
		}

		slog.Warn("Failed to connect to database, will retry in 3s", slog.Int("attempt", i), slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))
		time.Sleep(3 * time.Second)
	}

	slog.Error("Failed to connect to database after 3 attempts", slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))

	return nil, err
}

func parseSqliteConnectionString(connString string) (parsedConnString string, dbPath string, isMemoryDB bool, err error) {
	if !strings.HasPrefix(connString, "file:") {
		connString = "file:" + connString
	}

	// Check if we're using an in-memory database
	isMemoryDB = isSqliteInMemory(connString)

	// Parse the connection string
	connStringUrl, err := url.Parse(connString)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to parse SQLite connection string: %w", err)
	}

	// Convert options for the old SQLite driver to the new one
	convertSqlitePragmaArgs(connStringUrl)

	// Add the default and required params
	err = addSqliteDefaultParameters(connStringUrl, isMemoryDB)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid SQLite connection string: %w", err)
	}

	// Get the absolute path to the database
	// Here, we know for a fact that the ? is present
	parsedConnString = connStringUrl.String()
	idx := strings.IndexRune(parsedConnString, '?')
	dbPath, err = filepath.Abs(parsedConnString[len("file:"):idx])
	if err != nil {
		return "", "", false, fmt.Errorf("failed to determine absolute path to the database: %w", err)
	}

	return parsedConnString, dbPath, isMemoryDB, nil
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

// ensureSqliteTempDir ensures that SQLite has a directory where it can write temporary files if needed
// The default directory may not be writable when using a container with a read-only root file system
// See: https://www.sqlite.org/tempfiles.html
func ensureSqliteTempDir(dbPath string) error {
	// Per docs, SQLite tries these folders in order (excluding those that aren't applicable to us):
	//
	// - The SQLITE_TMPDIR environment variable
	// - The TMPDIR environment variable
	// - /var/tmp
	// - /usr/tmp
	// - /tmp
	//
	// Source: https://www.sqlite.org/tempfiles.html#temporary_file_storage_locations
	//
	// First, let's check if SQLITE_TMPDIR or TMPDIR are set, in which case we trust the user has taken care of the problem already
	if os.Getenv("SQLITE_TMPDIR") != "" || os.Getenv("TMPDIR") != "" {
		return nil
	}

	// Now, let's check if /var/tmp, /usr/tmp, or /tmp exist and are writable
	for _, dir := range []string{"/var/tmp", "/usr/tmp", "/tmp"} {
		ok, err := utils.IsWritableDir(dir)
		if err != nil {
			return fmt.Errorf("failed to check if %s is writable: %w", dir, err)
		}
		if ok {
			// We found a folder that's writable
			return nil
		}
	}

	// If we're here, there's no temporary directory that's writable (not unusual for containers with a read-only root file system), so we set SQLITE_TMPDIR to the folder where the SQLite database is set
	err := os.Setenv("SQLITE_TMPDIR", dbPath)
	if err != nil {
		return fmt.Errorf("failed to set SQLITE_TMPDIR environmental variable: %w", err)
	}

	slog.Debug("Set SQLITE_TMPDIR to the database directory", "path", dbPath)

	return nil
}

func getGormLogger() gormLogger.Interface {
	loggerOpts := make([]slogGorm.Option, 0, 5)
	loggerOpts = append(loggerOpts,
		slogGorm.WithSlowThreshold(200*time.Millisecond),
		slogGorm.WithErrorField("error"),
	)

	if common.EnvConfig.LogLevel == "debug" {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelDebug),
			slogGorm.WithRecordNotFoundError(),
			slogGorm.WithTraceAll(),
		)

	} else {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelWarn),
			slogGorm.WithIgnoreTrace(),
		)
	}

	return slogGorm.New(loggerOpts...)
}
