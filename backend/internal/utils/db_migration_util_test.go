package utils_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestMigrateDatabaseWithNewerMigrations(t *testing.T) {
	t.Run("an older replica keeps the newer migrations", func(t *testing.T) {
		sqlDb := newDatabaseWithNewerMigration(t, "-- +goose Up\nCREATE TABLE newer_feature (id TEXT);\n-- +goose Down\nDROP TABLE newer_feature;\n")

		require.NoError(t, utils.MigrateDatabase(t.Context(), sqlDb, common.DbProviderSqlite))
		exists, err := utils.TableExists(t.Context(), sqlDb, common.DbProviderSqlite, "newer_feature")
		require.NoError(t, err)
		require.True(t, exists, "the newer migration must not be rolled back")
	})

	t.Run("a downgrade rolls back the newer migrations", func(t *testing.T) {
		sqlDb := newDatabaseWithNewerMigration(t, "-- +goose Up\nCREATE TABLE newer_feature (id TEXT);\n-- +goose Down\nDROP TABLE newer_feature;\n")
		previous := common.EnvConfig.AllowDowngrade
		common.EnvConfig.AllowDowngrade = true
		t.Cleanup(func() {
			common.EnvConfig.AllowDowngrade = previous
		})

		require.NoError(t, utils.MigrateDatabase(t.Context(), sqlDb, common.DbProviderSqlite))
		exists, err := utils.TableExists(t.Context(), sqlDb, common.DbProviderSqlite, "newer_feature")
		require.NoError(t, err)
		require.False(t, exists)
	})
}

// newDatabaseWithNewerMigration returns a database where a newer Pocket ID version applied a migration this version doesn't know, and stored its file
func newDatabaseWithNewerMigration(t *testing.T, content string) *sql.DB {
	sqlDb, err := testutils.NewDatabaseForTest(t).DB()
	require.NoError(t, err)
	for _, query := range []string{
		`CREATE TABLE newer_feature (id TEXT)`,
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES (99990101000000, true)`,
	} {
		_, err = sqlDb.ExecContext(t.Context(), query)
		require.NoError(t, err)
	}
	_, err = sqlDb.ExecContext(t.Context(), `INSERT INTO goose_migration_files (version_id, file_name, content) VALUES (99990101000000, '99990101000000_newer_feature.sql', $1)`, content)
	require.NoError(t, err)
	return sqlDb
}
