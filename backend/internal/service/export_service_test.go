package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/haconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
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

// TestExportOmitsHAModeKV verifies that an export omits the "ha_enabled" row from the kv table
// The HA mode is a per-cluster setting re-established from config on import, so carrying it in the export would pin an imported cluster to the exporter's mode
func TestExportOmitsHAModeKV(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	// Seed the kv table with the HA mode row (must be omitted) and another row (must be kept)
	haValue := "true"
	instanceValue := "instance-123"
	require.NoError(t, db.Create(&model.KV{Key: haconfig.KVKey, Value: &haValue}).Error)
	require.NoError(t, db.Create(&model.KV{Key: "instance_id", Value: &instanceValue}).Error)

	export, err := NewExportService(db, nil).extractDatabase()
	require.NoError(t, err)

	keys := make(map[string]bool)
	for _, row := range export.Tables["kv"] {
		keyPtr, ok := row["key"].(*string)
		require.True(t, ok)
		require.NotNil(t, keyPtr)
		keys[*keyPtr] = true
	}

	require.NotContains(t, keys, haconfig.KVKey, "export must omit the HA mode row")
	require.Contains(t, keys, "instance_id", "export must keep other kv rows")
}
