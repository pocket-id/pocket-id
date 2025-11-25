package resources

import (
	"embed"
	"slices"
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
	// We can ignore migrations with version below 20251115000000
	const ignoreBefore = 20251115000000

	// Scan postgres migrations
	postgresMigrations := scanMigrations(t, FS, "migrations/postgres", ignoreBefore)

	// Scan sqlite migrations
	sqliteMigrations := scanMigrations(t, FS, "migrations/sqlite", ignoreBefore)

	// Sort both lists for consistent comparison
	slices.Sort(postgresMigrations)
	slices.Sort(sqliteMigrations)

	// Compare the lists
	assert.EqualValues(t, postgresMigrations, sqliteMigrations, "Migration versions must match between Postgres and SQLite")
}

// scanMigrations scans a directory for migration files and returns a list of versions
func scanMigrations(t *testing.T, fs embed.FS, dir string, ignoreBefore int64) []int64 {
	t.Helper()

	entries, err := fs.ReadDir(dir)
	require.NoErrorf(t, err, "Failed to read directory '%s'", dir)

	// Divide by 2 because of up and down files
	versions := make([]int64, 0, len(entries)/2)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Only consider .up.sql files
		if !strings.HasSuffix(filename, ".up.sql") {
			continue
		}

		// Extract version from filename (format: <version>_<anything>.up.sql)
		versionString, _, ok := strings.Cut(filename, "_")
		require.Truef(t, ok, "Migration file has unexpected format: %s", filename)

		version, err := strconv.ParseInt(versionString, 10, 64)
		require.NoErrorf(t, err, "Failed to parse version from filename '%s'", filename)

		// Exclude migrations with version below ignoreBefore
		if version < ignoreBefore {
			continue
		}

		versions = append(versions, version)
	}

	return versions
}
