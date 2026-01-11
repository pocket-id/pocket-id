package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// MigrateDatabase applies database migrations using embedded migration files or fetches them from GitHub if a downgrade is detected.
func MigrateDatabase(sqlDb *sql.DB) error {
	m, err := GetEmbeddedMigrateInstance(sqlDb)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	path := "migrations/" + string(common.EnvConfig.DbProvider)
	requiredVersion, err := getRequiredMigrationVersion(path)
	if err != nil {
		return fmt.Errorf("failed to get last migration version: %w", err)
	}

	currentVersion, _, _ := m.Version()
	if currentVersion > requiredVersion {
		slog.Warn("Database version is newer than the application supports, possible downgrade detected", slog.Uint64("db_version", uint64(currentVersion)), slog.Uint64("app_version", uint64(requiredVersion)))
		if !common.EnvConfig.AllowDowngrade {
			return fmt.Errorf("database version (%d) is newer than application version (%d), downgrades are not allowed (set ALLOW_DOWNGRADE=true to enable)", currentVersion, requiredVersion)
		}
		slog.Info("Fetching migrations from GitHub to handle possible downgrades")
		return migrateDatabaseFromGitHub(sqlDb, requiredVersion, currentVersion)
	}

	err = m.Migrate(requiredVersion)
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		if errors.As(err, &migrate.ErrDirty{}) {
			return fmt.Errorf("database migration failed. Please create an issue on GitHub and temporarely downgrade to the previous version: %w", err)
		}
		return fmt.Errorf("failed to apply embedded migrations: %w", err)
	}
	return nil
}

// GetEmbeddedMigrateInstance creates a migrate.Migrate instance using embedded migration files.
func GetEmbeddedMigrateInstance(sqlDb *sql.DB) (*migrate.Migrate, error) {
	path := "migrations/" + string(common.EnvConfig.DbProvider)
	source, err := iofs.New(resources.FS, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedded migration source: %w", err)
	}

	driver, err := newMigrationDriver(sqlDb, common.EnvConfig.DbProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}
	return m, nil
}

// newMigrationDriver creates a database.Driver instance based on the given database provider.
func newMigrationDriver(sqlDb *sql.DB, dbProvider common.DbProvider) (driver database.Driver, err error) {
	switch dbProvider {
	case common.DbProviderSqlite:
		driver, err = sqliteMigrate.WithInstance(sqlDb, &sqliteMigrate.Config{
			NoTxWrap: true,
		})
	case common.DbProviderPostgres:
		driver, err = postgresMigrate.WithInstance(sqlDb, &postgresMigrate.Config{})
	default:
		// Should never happen at this point
		return nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	return driver, nil
}

// migrateDatabaseFromGitHub applies database migrations fetched from GitHub to handle downgrades.
func migrateDatabaseFromGitHub(sqlDb *sql.DB, requiredVersion uint, currentVersion uint) error {
	srcURL := "github://pocket-id/pocket-id/backend/resources/migrations/" + string(common.EnvConfig.DbProvider)

	driver, err := newMigrationDriver(sqlDb, common.EnvConfig.DbProvider)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(srcURL, "pocket-id", driver)
	if err != nil {
		return fmt.Errorf("failed to create GitHub migration instance: %w", err)
	}

	// Reset the dirty state before forcing the version
	if err := m.Force(int(currentVersion)); err != nil {
		return fmt.Errorf("failed to force database version: %w", err)
	}

	if err := m.Migrate(requiredVersion); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to apply GitHub migrations: %w", err)
	}

	return nil
}

// getRequiredMigrationVersion reads the embedded migration files and returns the highest version number found.
func getRequiredMigrationVersion(path string) (uint, error) {
	entries, err := resources.FS.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var maxVersion uint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var version uint
		n, err := fmt.Sscanf(name, "%d_", &version)
		if err == nil && n == 1 {
			if version > maxVersion {
				maxVersion = version
			}
		}
	}

	return maxVersion, nil
}
