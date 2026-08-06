package service

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// TestExportExcludesActorHostTables verifies that an export does not dump the actor host's own "francis_" tables
// They hold volatile runtime state (host registrations, alarms, …), are not part of a Pocket ID export, and including them made the CLI export/import comparison tests fail
func TestExportExcludesActorHostTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pocket-id.db")
	db := openImportTestDB(t, dbPath, nil)
	defer closeImportTestDB(db)

	sqlDB, _ := db.DB()
	require.NoError(t, utils.MigrateDatabase(t.Context(), sqlDB))
	seedActorHostSchema(t, db) // creates francis_active_actors (with a row) and a view over it

	export, err := NewExportService(db, nil, nil).extractDatabase(t.Context())
	require.NoError(t, err)

	for table := range export.Tables {
		require.Falsef(t, strings.HasPrefix(table, "francis_"), "export must not include actor host table %q", table)
	}
}

func TestExportSnapshotTxOptions(t *testing.T) {
	t.Run("postgres pins one snapshot for the whole transaction", func(t *testing.T) {
		require.Equal(t, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, snapshotTxOptions("postgres"))
	})

	// SQLite must be left at the driver's default isolation, and read-only, or the driver would honor the DSN's "_txlock=immediate" and take a write lock for the whole export
	t.Run("sqlite reads without taking a write lock", func(t *testing.T) {
		require.Equal(t, &sql.TxOptions{ReadOnly: true}, snapshotTxOptions("sqlite"))
	})
}

// TestExportRunsWhileAnotherConnectionWrites covers an export taken while Pocket ID is running and writing to the database.
// The DSN sets "_txlock=immediate", so an export that did not open its transaction read-only would take a write lock, block on the concurrent writer, and fail once the busy timeout expired.
func TestExportRunsWhileAnotherConnectionWrites(t *testing.T) {
	db := newExportTestDB(t)

	require.NoError(t, db.Exec(`INSERT INTO kv ("key", "value") VALUES ('committed', 'v')`).Error)

	// Hold the write lock on another connection for the duration of the export
	writer := db.Begin()
	require.NoError(t, writer.Error)
	defer writer.Rollback()
	require.NoError(t, writer.Exec(`INSERT INTO kv ("key", "value") VALUES ('uncommitted', 'v')`).Error)

	export, err := NewExportService(db, nil, nil).extractDatabase(t.Context())
	require.NoError(t, err)

	// Only the committed row belongs in the export
	require.Len(t, export.Tables["kv"], 1)
	require.Equal(t, "committed", *(export.Tables["kv"][0]["key"].(*string)))
}

// TestExportSnapshotIgnoresWritesDuringTheExport covers a write that is committed after the export has started.
// Each table is read with its own query, so without a single snapshot a row inserted midway would land in some tables but not others.
func TestExportSnapshotIgnoresWritesDuringTheExport(t *testing.T) {
	db := newExportTestDB(t)

	tx := db.Begin(snapshotTxOptions(db.Name()))
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	// A deferred read transaction takes its snapshot at the first read, which is what the export does when it loads the schema
	var before int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM kv`).Scan(&before).Error)

	// A committed write on another connection must neither be blocked nor visible to the open snapshot
	require.NoError(t, db.Exec(`INSERT INTO kv ("key", "value") VALUES ('added-mid-export', 'v')`).Error)

	var after int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM kv`).Scan(&after).Error)
	require.Equal(t, before, after, "the export's snapshot must not include rows written after it started")
}

// newExportTestDB opens a migrated SQLite database using the same DSN the application uses, on a file so the pool can hand out more than one connection
func newExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := openImportTestDB(t, filepath.Join(t.TempDir(), "pocket-id.db"), nil)
	t.Cleanup(func() {
		closeImportTestDB(db)
	})

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, utils.MigrateDatabase(t.Context(), sqlDB))

	return db
}
