//go:build unit

// This file is only imported by unit tests

package testing

import (
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	sqlitekit "github.com/italypaleale/go-sql-utils/sqlite"
	"github.com/libtnb/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	sqliteutil "github.com/pocket-id/pocket-id/backend/internal/utils/sqlite"
	"github.com/pocket-id/pocket-id/backend/resources"
)

func init() {
	sqliteutil.RegisterSqliteFunctions()
}

// NewDatabaseForTest returns a new instance of GORM connected to an in-memory SQLite database.
// Each database connection is unique for the test.
// All migrations are automatically performed.
// Note: the in-memory database is limited to a single connection, so it cannot be used to test concurrent access: use NewConcurrentDatabaseForTest for that.
func NewDatabaseForTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := openInMemoryTestDB(t)
	runMigrations(t, db, 0, nil)
	return db
}

// NewDatabaseForTestWithMigrationSeed behaves like NewDatabaseForTest, but pauses the migrations right after the migration whose version is stopAfterVersion has been applied.
// It then invokes seed, so the test can insert data into the intermediate schema, before applying the remaining migrations.
// This is meant to test data migrations, which operate on data that already exists in the database.
func NewDatabaseForTestWithMigrationSeed(t *testing.T, stopAfterVersion uint, seed func(t *testing.T, db *gorm.DB)) *gorm.DB {
	t.Helper()
	db := openInMemoryTestDB(t)
	runMigrations(t, db, stopAfterVersion, seed)
	return db
}

// NewConcurrentDatabaseForTest returns a new instance of GORM connected to a temporary file-based SQLite database
// Unlike NewDatabaseForTest, which forces a single connection to an in-memory database, this one can be used to test concurrent access to the database.
// All migrations are automatically performed.
func NewConcurrentDatabaseForTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := openFileTestDB(t)
	runMigrations(t, db, 0, nil)
	return db
}

// openInMemoryTestDB opens a GORM instance backed by an in-memory SQLite database, unique to the test.
func openInMemoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Get a name for this in-memory database that is specific to the test
	dbName := utils.CreateSha256Hash(t.Name())

	// Connect to a new in-memory SQL database
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory"), newTestGormConfig(t))
	require.NoError(t, err, "Failed to connect to test database")

	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")

	// For in-memory SQLite databases, we must limit to 1 open connection at the same time, or they won't see the whole data
	// The other workaround, of using shared caches, doesn't work well with multiple write transactions trying to happen at once
	sqlDB.SetMaxOpenConns(1)

	return db
}

// openFileTestDB opens a GORM instance backed by a temporary file-based SQLite database, configured like the production database
func openFileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	connString, _, _, err := sqlitekit.ParseConnectionString(dbPath, slog.Default())
	require.NoError(t, err, "Failed to parse connection string")

	db, err := gorm.Open(sqlite.Open(connString), newTestGormConfig(t))
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

// runMigrations applies the embedded SQLite migrations to db.
// If seed is not nil, migrations are first applied up to and including stopAfterVersion, then seed is invoked so the test can insert data into the intermediate schema, before the remaining migrations are applied.
func runMigrations(t *testing.T, db *gorm.DB, stopAfterVersion uint, seed func(t *testing.T, db *gorm.DB)) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")

	// Perform migrations with the embedded migrations
	driver, err := sqliteMigrate.WithInstance(sqlDB, &sqliteMigrate.Config{
		NoTxWrap: true,
	})
	require.NoError(t, err, "Failed to create migration driver")
	source, err := iofs.New(resources.FS, "migrations/sqlite")
	require.NoError(t, err, "Failed to create embedded migration source")
	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	require.NoError(t, err, "Failed to create migration instance")

	// If the test wants to seed data partway through the migrations, apply migrations up to and including stopAfterVersion first, then let it seed
	if seed != nil {
		err = m.Migrate(stopAfterVersion)
		require.NoErrorf(t, err, "Failed to perform migrations up to version %d", stopAfterVersion)
		seed(t, db)
	}

	// Apply all the remaining migrations
	// ErrNoChange means we were already at the latest version, which is not an error here
	err = m.Up()
	if !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err, "Failed to perform migrations")
	}
	_, err = sqlDB.Exec("PRAGMA foreign_keys = OFF;")
	require.NoError(t, err, "Failed to disable foreign keys")
}

// newTestGormConfig returns the GORM configuration shared by the test databases, wiring the logger to the test's output.
func newTestGormConfig(t *testing.T) *gorm.Config {
	t.Helper()
	return &gorm.Config{
		TranslateError: true,
		Logger: logger.New(
			testLoggerAdapter{t: t},
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: false,
				ParameterizedQueries:      false,
				Colorful:                  false,
			},
		),
	}
}

// Implements gorm's logger.Writer interface
type testLoggerAdapter struct {
	t *testing.T
}

func (l testLoggerAdapter) Printf(format string, args ...any) {
	l.t.Logf(format, args...)
}
