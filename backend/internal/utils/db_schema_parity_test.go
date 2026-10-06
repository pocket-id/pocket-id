//go:build unit

package utils_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestDBSchemaParity ensures the SQLite and Postgres migrations produce schemas that encode every column the same way in a data export
// Without this, an export taken from one database provider fails to import into the other (see https://github.com/pocket-id/pocket-id/issues/1603 and https://github.com/pocket-id/pocket-id/issues/1815)
// If this test fails after adding a migration, pick column types for both databases that map to the same utils.DBExportKind
func TestDBSchemaParity(t *testing.T) {
	sqliteSchema, err := utils.LoadDBSchemaTypes(testutils.NewDatabaseForTest(t))
	require.NoError(t, err)
	postgresSchema, err := utils.LoadDBSchemaTypes(testutils.NewPostgresDatabaseForTest(t))
	require.NoError(t, err)

	// The migration bookkeeping table is managed by golang-migrate and never exported
	delete(sqliteSchema, "schema_migrations")
	delete(postgresSchema, "schema_migrations")

	// Both databases must have the same tables
	require.ElementsMatch(t, slices.Collect(maps.Keys(sqliteSchema)), slices.Collect(maps.Keys(postgresSchema)), "SQLite and Postgres must have the same tables")

	for table, sqliteColumns := range sqliteSchema {
		postgresColumns := postgresSchema[table]

		// Both databases must have the same columns in each table
		if !assert.ElementsMatchf(t, slices.Collect(maps.Keys(sqliteColumns)), slices.Collect(maps.Keys(postgresColumns)), "SQLite and Postgres must have the same columns in table %q", table) {
			continue
		}

		// Each column must be encoded the same way in an export and accept the same values
		for column, sqliteColumn := range sqliteColumns {
			postgresColumn := postgresColumns[column]
			assert.Equalf(t, sqliteColumn.ExportKind(), postgresColumn.ExportKind(),
				"column %s.%s is exported differently: SQLite type %q is %q, Postgres type %q is %q",
				table, column, sqliteColumn.Name, sqliteColumn.ExportKind(), postgresColumn.Name, postgresColumn.ExportKind())
			assert.Equalf(t, sqliteColumn.Nullable, postgresColumn.Nullable,
				"column %s.%s has a different nullability: SQLite nullable=%t, Postgres nullable=%t",
				table, column, sqliteColumn.Nullable, postgresColumn.Nullable)
		}
	}
}
