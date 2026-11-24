package utils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	goosedb "github.com/pressly/goose/v3/database"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/resources"
)

const (
	// baselineMigrationVersion is the version of the migration that creates the whole schema Pocket ID v3 started with
	// The migrations of earlier versions are not part of this version, so a schema older than the baseline cannot be migrated
	baselineMigrationVersion = 20261114130000

	// migrationFilesTable keeps a copy of the migration files of every Pocket ID version that ran against the database
	// An older version needs them to roll back the migrations it doesn't know
	migrationFilesTable = "goose_migration_files"
)

// IsMigrationTable reports whether a table keeps track of the migrations instead of holding Pocket ID data
func IsMigrationTable(table string) bool {
	return strings.HasPrefix(table, "goose_")
}

// CheckMigrationVersionSupported returns an error if data at the given schema version is too old to be migrated to the current schema
func CheckMigrationVersionSupported(version int64) error {
	if version < baselineMigrationVersion {
		return fmt.Errorf("schema version %d is older than the one of Pocket ID v3 (%d)", version, baselineMigrationVersion)
	}
	return nil
}

// MigrateDatabase applies the embedded migrations to the database
// If the database has migrations of a newer Pocket ID version, they are rolled back first when downgrades are allowed, and kept otherwise
func MigrateDatabase(ctx context.Context, sqlDb *sql.DB, dbProvider common.DbProvider) error {
	dialect := migrationDialect(dbProvider)
	store, err := goosedb.NewStore(dialect, goose.DefaultTablename)
	if err != nil {
		return fmt.Errorf("failed to create the migration store: %w", err)
	}
	files, err := fs.Sub(resources.FS, "migrations/"+string(dbProvider))
	if err != nil {
		return fmt.Errorf("failed to open the embedded migrations: %w", err)
	}
	provider, err := newMigrationProvider(sqlDb, dialect, files)
	if err != nil {
		return err
	}

	// Keep a copy of the migration files, which an older Pocket ID version needs to roll them back
	err = storeMigrationFiles(ctx, sqlDb, files, provider.ListSources())
	if err != nil {
		return err
	}

	// Roll back the migrations of a newer Pocket ID version, then apply the ones of this version
	err = rollBackUnknownMigrations(ctx, sqlDb, dbProvider, store, provider.ListSources())
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply the migrations: %w", err)
	}
	for _, result := range results {
		slog.InfoContext(ctx, "Applied database migration", slog.String("migration", result.Source.Path), slog.Duration("duration", result.Duration))
	}

	return nil
}

// MigrateDatabaseTo applies the embedded migrations up to and including the given version to an empty database
func MigrateDatabaseTo(ctx context.Context, sqlDb *sql.DB, dbProvider common.DbProvider, version int64) error {
	files, err := fs.Sub(resources.FS, "migrations/"+string(dbProvider))
	if err != nil {
		return fmt.Errorf("failed to open the embedded migrations: %w", err)
	}
	provider, err := newMigrationProvider(sqlDb, migrationDialect(dbProvider), files)
	if err != nil {
		return err
	}

	_, err = provider.UpTo(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to apply the migrations: %w", err)
	}
	return nil
}

// TableExists reports whether the database has a table with the given name
func TableExists(ctx context.Context, sqlDb *sql.DB, dbProvider common.DbProvider, table string) (exists bool, err error) {
	query := `SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = $1)`
	if dbProvider == common.DbProviderPostgres {
		query = `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = current_schema() AND tablename = $1)`
	}
	err = sqlDb.QueryRowContext(ctx, query, table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check for the %s table: %w", table, err)
	}
	return exists, nil
}

func migrationDialect(dbProvider common.DbProvider) goose.Dialect {
	if dbProvider == common.DbProviderPostgres {
		return goose.DialectPostgres
	}
	return goose.DialectSQLite3
}

func newMigrationProvider(sqlDb *sql.DB, dialect goose.Dialect, files fs.FS) (*goose.Provider, error) {
	provider, err := goose.NewProvider(dialect, sqlDb, files,
		// Migrations of parallel branches can be merged in a different order than their versions, and they must still be applied
		goose.WithAllowOutofOrder(true),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the migration provider: %w", err)
	}
	return provider, nil
}

// storeMigrationFiles saves the content of every embedded migration file in the database
func storeMigrationFiles(ctx context.Context, sqlDb *sql.DB, files fs.FS, sources []*goose.Source) error {
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+migrationFilesTable+` (
		version_id BIGINT NOT NULL PRIMARY KEY,
		file_name TEXT NOT NULL,
		content TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("failed to create the migration files table: %w", err)
	}

	for _, source := range sources {
		content, err := fs.ReadFile(files, source.Path)
		if err != nil {
			return fmt.Errorf("failed to read migration %d: %w", source.Version, err)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO `+migrationFilesTable+` (version_id, file_name, content) VALUES ($1, $2, $3)
			ON CONFLICT (version_id) DO UPDATE SET file_name = excluded.file_name, content = excluded.content`,
			source.Version, source.Path, string(content))
		if err != nil {
			return fmt.Errorf("failed to store migration %d: %w", source.Version, err)
		}
	}

	return tx.Commit()
}

// rollBackUnknownMigrations rolls back the applied migrations that aren't part of this version, newest first, using the files the newer version stored
// Without ALLOW_DOWNGRADE it keeps them and this version runs on the newer schema, so an older replica still starts during a rolling update
func rollBackUnknownMigrations(ctx context.Context, sqlDb *sql.DB, dbProvider common.DbProvider, store goosedb.Store, sources []*goose.Source) error {
	// A new database has no migrations to roll back
	exists, err := TableExists(ctx, sqlDb, dbProvider, goose.DefaultTablename)
	if err != nil || !exists {
		return err
	}

	// Find the migrations this version doesn't know, in the reverse order they were applied
	known := make(map[int64]bool, len(sources))
	for _, source := range sources {
		known[source.Version] = true
	}
	applied, err := store.ListMigrations(ctx, sqlDb)
	if err != nil {
		return fmt.Errorf("failed to list the applied migrations: %w", err)
	}
	var unknown []int64
	for _, migration := range applied {
		if migration.Version != 0 && migration.IsApplied && !known[migration.Version] {
			unknown = append(unknown, migration.Version)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	if !common.EnvConfig.AllowDowngrade {
		slog.WarnContext(ctx, "The database has migrations of a newer Pocket ID version, which are kept because downgrades are not allowed (set ALLOW_DOWNGRADE=true to roll them back)", slog.Any("versions", unknown))
		return nil
	}
	slog.WarnContext(ctx, "Rolling back the migrations of a newer Pocket ID version", slog.Any("versions", unknown))

	// Load the files of those migrations, which the newer version stored when it ran
	stored := fstest.MapFS{}
	for _, version := range unknown {
		var name, content string
		err = sqlDb.QueryRowContext(ctx, `SELECT file_name, content FROM `+migrationFilesTable+` WHERE version_id = $1`, version).Scan(&name, &content)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("migration %d cannot be rolled back, because the Pocket ID version that applied it didn't store it", version)
		} else if err != nil {
			return fmt.Errorf("failed to load migration %d: %w", version, err)
		}
		stored[name] = &fstest.MapFile{Data: []byte(content)}
	}

	provider, err := newMigrationProvider(sqlDb, migrationDialect(dbProvider), stored)
	if err != nil {
		return err
	}
	for _, version := range unknown {
		result, err := provider.ApplyVersion(ctx, version, false)
		if err != nil {
			return fmt.Errorf("failed to roll back migration %d: %w", version, err)
		}
		slog.InfoContext(ctx, "Rolled back database migration", slog.String("migration", result.Source.Path), slog.Duration("duration", result.Duration))
	}

	return nil
}
