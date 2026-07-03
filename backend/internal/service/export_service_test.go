package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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

	export, err := NewExportService(db, nil).extractDatabase()
	require.NoError(t, err)

	for table := range export.Tables {
		require.Falsef(t, strings.HasPrefix(table, "francis_"), "export must not include actor host table %q", table)
	}
}
