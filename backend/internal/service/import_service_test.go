package service

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/libtnb/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// importResetTargetVersion is a real migration version the import resets the schema to
// It only has to be at or beyond 20240817191051 (the rename_config_table migration that toggles foreign keys), so the reset exercises a foreign-key-sensitive migration
const importResetTargetVersion = 20260708120000

// openImportTestDB opens a Gorm SQLite pool on a file database using the same pragmas the application configures in production, most importantly foreign_keys(1) on every connection (normalize() is already registered by the service package's test setup)
func openImportTestDB(t *testing.T, dbPath string, cfg func(*sql.DB)) *gorm.DB {
	t.Helper()
	common.EnvConfig.DbProvider = common.DbProviderSqlite

	dsn := "file:" + dbPath + "?_txlock=immediate&_pragma=busy_timeout(2500)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	require.NoError(t, err)

	if cfg != nil {
		sqlDB, _ := db.DB()
		cfg(sqlDB)
	}

	return db
}

func closeImportTestDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

// seedActorHostSchema simulates the actor host's own tables and views (the "francis_" prefix), including a view over a table, which an import must preserve.
func seedActorHostSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE francis_active_actors (id TEXT PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE VIEW francis_host_active_actor_count AS SELECT count(*) AS n FROM francis_active_actors`).Error)
	require.NoError(t, db.Exec(`INSERT INTO francis_active_actors (id) VALUES ('actor-1')`).Error)
}

// requireActorHostSchemaPreserved asserts the actor host's tables/views and their data survived an import.
func requireActorHostSchemaPreserved(t *testing.T, db *gorm.DB) {
	t.Helper()
	var tableRows int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM francis_active_actors`).Scan(&tableRows).Error)
	require.Equal(t, int64(1), tableRows, "francis_ tables and their rows must be preserved by an import")

	// The view is only valid if its backing table was preserved as well
	var viewCount int64
	require.NoError(t, db.Raw(`SELECT n FROM francis_host_active_actor_count`).Scan(&viewCount).Error)
	require.Equal(t, int64(1), viewCount, "francis_ views must be preserved by an import")
}

// importAndRestart mirrors the CLI import flow (tests/specs/cli.spec.ts): a running instance is stopped, `pocket-id import` replaces the schema and data, and the instance is started again.
// Each step uses a fresh pool to emulate the separate processes involved. importCfg configures the pool the import step uses.
func importAndRestart(importCfg func(*sql.DB)) func(t *testing.T) {
	return func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "pocket-id.db")

		// Initial boot: connect, migrate, and let the actor host create its own schema.
		boot := openImportTestDB(t, dbPath, nil)
		sqlDB, _ := boot.DB()
		err := utils.MigrateDatabase(t.Context(), sqlDB)
		require.NoError(t, err)
		seedActorHostSchema(t, boot)
		closeImportTestDB(boot)

		// Import: reset the schema (drop + migrate) and load data. Empty tables keep the test focused
		// on the schema reset.
		imp := openImportTestDB(t, dbPath, importCfg)
		err = NewImportService(imp, nil).ImportDatabase(t.Context(), DatabaseExport{
			Provider: "sqlite",
			Version:  importResetTargetVersion,
			Tables:   map[string][]map[string]any{},
		})
		closeImportTestDB(imp)
		require.NoError(t, err)

		// Restart: migrating again must be a clean no-op, not fail on a dirty database, and the actor
		// host's own schema must still be intact
		restart := openImportTestDB(t, dbPath, nil)
		defer closeImportTestDB(restart)
		sqlDB, _ = restart.DB()
		err = utils.MigrateDatabase(t.Context(), sqlDB)
		require.NoError(t, err)
		requireActorHostSchemaPreserved(t, restart)
	}
}

func TestImportResetSchema(t *testing.T) {
	importAndRestart(nil)(t)
}

// TestImportResetSchemaFreshConnections guards the schema reset against connection pooling.
// resetSchema must drop tables with foreign keys disabled on the connection that performs the drops.
// With MaxIdleConns(0) every statement runs on a fresh connection (which the DSN opens with foreign_keys(1)); before dropPocketIDTablesSQLite this reproduced the CI failure where DROP TABLE tripped foreign-key cascades/triggers and left the database dirty, breaking the container restart.
func TestImportResetSchemaFreshConnections(t *testing.T) {
	importAndRestart(func(db *sql.DB) { db.SetMaxIdleConns(0) })(t)
}
