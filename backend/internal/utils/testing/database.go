// This file is only imported by unit tests

package testing

import (
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/glebarez/sqlite"
	"github.com/golang-migrate/migrate/v4"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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
func NewDatabaseForTest(t *testing.T) *gorm.DB {
	t.Helper()

	// Get a name for this in-memory database that is specific to the test
	dbName := utils.CreateSha256Hash(t.Name())

	// Connect to a new in-memory SQL database
	db, err := gorm.Open(
		sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"),
		&gorm.Config{
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
		})
	require.NoError(t, err, "Failed to connect to test database")

	// Perform migrations with the embedded migrations
	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")
	driver, err := sqliteMigrate.WithInstance(sqlDB, &sqliteMigrate.Config{})
	require.NoError(t, err, "Failed to create migration driver")
	source, err := iofs.New(resources.FS, "migrations/sqlite")
	require.NoError(t, err, "Failed to create embedded migration source")
	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	require.NoError(t, err, "Failed to create migration instance")
	err = m.Up()
	require.NoError(t, err, "Failed to perform migrations")

	return db
}

// Implements gorm's logger.Writer interface
type testLoggerAdapter struct {
	t *testing.T
}

func (l testLoggerAdapter) Printf(format string, args ...any) {
	l.t.Logf(format, args...)
}
