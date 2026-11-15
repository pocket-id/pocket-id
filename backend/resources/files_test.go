package resources

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test is meant to enforce that for every new migration added, a file with the same migration number exists for all supported databases
// This is necessary to ensure import/export works correctly
// Note: if a migration is not needed for a database, ensure there's a file with an empty (no-op) migration (e.g. even just a comment)
func TestMigrationsMatchingVersions(t *testing.T) {
	postgresMigrations := scanMigrations(t, "migrations/postgres")
	sqliteMigrations := scanMigrations(t, "migrations/sqlite")

	assert.Equal(t, postgresMigrations, sqliteMigrations, "Migration versions must match between Postgres and SQLite")
}

// scanMigrations returns the versions of the migration files in a directory, which are named <version>_<name>.sql
func scanMigrations(t *testing.T, dir string) []int64 {
	t.Helper()

	entries, err := FS.ReadDir(dir)
	require.NoErrorf(t, err, "Failed to read directory '%s'", dir)

	versions := make([]int64, 0, len(entries))
	for _, entry := range entries {
		versionString, _, ok := strings.Cut(entry.Name(), "_")
		require.Truef(t, ok, "Migration file has unexpected format: %s", entry.Name())

		version, err := strconv.ParseInt(versionString, 10, 64)
		require.NoErrorf(t, err, "Failed to parse version from filename '%s'", entry.Name())

		versions = append(versions, version)
	}

	return versions
}
