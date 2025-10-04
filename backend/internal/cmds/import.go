package cmds

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

type importFlags struct {
	Path string
}

func init() {
	var flags importFlags

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Imports all data of Pocket ID from a ZIP file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(flags)
		},
	}

	importCmd.Flags().StringVarP(&flags.Path, "path", "p", "./pocket-id-export.zip", "Path to the ZIP file to import the data from")

	rootCmd.AddCommand(importCmd)
}

// runImport handles the high-level orchestration of the import process
func runImport(flags importFlags) error {
	if !askForConfirmation() {
		fmt.Println("Import aborted.")
		return nil
	}

	r, err := zip.OpenReader(flags.Path)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	dbData, err := processZipDatabaseJson(r.File)
	if err != nil {
		return err
	}

	if dbData.Provider != string(common.EnvConfig.DbProvider) {
		return fmt.Errorf(
			"import file is for %s, but current DB provider is %s",
			dbData.Provider,
			common.EnvConfig.DbProvider,
		)
	}

	// Connect DB and reset schema
	db, err := bootstrap.NewDatabase()
	if err != nil {
		return err
	}
	if err := resetSchema(db); err != nil {
		return err
	}

	if err := runMigrations(db, dbData.Version); err != nil {
		return err
	}

	if err := insertData(db, dbData); err != nil {
		return err
	}

	if err := processZipFiles(r.File); err != nil {
		return err
	}

	fmt.Println("Import completed successfully.")
	return nil
}

func askForConfirmation() bool {
	fmt.Println("WARNING: This feature is experimental and may not work correctly. Please create a backup before proceeding and report any issues you encounter.")
	fmt.Println()
	fmt.Println("WARNING: Import will erase all existing data at the following locations:")
	fmt.Printf("Database:      %s\n", absolutePathOrOriginal(common.EnvConfig.DbConnectionString))
	fmt.Printf("Uploads Path:  %s\n", absolutePathOrOriginal(common.EnvConfig.UploadPath))

	fmt.Printf("Keys Path:     %s\n", absolutePathOrOriginal(common.EnvConfig.KeysPath))

	ok, err := utils.PromptForConfirmation("Do you want to continue?")
	if err != nil {
		panic(err)
	}
	return ok

}

// processZipDatabaseJson extracts database.json from the ZIP archive
func processZipDatabaseJson(files []*zip.File) (databaseJson, error) {
	var dbData databaseJson

	for _, f := range files {
		if f.Name == "database.json" {
			if err := readDatabaseJSON(f, &dbData); err != nil {
				return dbData, err
			}
			return dbData, nil
		}
	}
	return dbData, errors.New("database.json not found in the ZIP file")
}

// processZipFiles extracts uploads/ and keys/ from the ZIP archive
func processZipFiles(files []*zip.File) error {
	for _, f := range files {
		switch {
		case strings.HasPrefix(f.Name, "uploads/"):
			if err := extractIntoBase(f, common.EnvConfig.UploadPath, "uploads/"); err != nil {
				return fmt.Errorf("failed to extract uploads: %w", err)
			}

		case strings.HasPrefix(f.Name, "keys/"):
			if err := extractIntoBase(f, common.EnvConfig.KeysPath, "keys/"); err != nil {
				return fmt.Errorf("failed to extract keys: %w", err)
			}
		}
	}

	return nil
}

// readDatabaseJSON parses database.json from the ZIP file
func readDatabaseJSON(f *zip.File, dbData *databaseJson) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open database.json: %w", err)
	}
	defer rc.Close()

	if err := json.NewDecoder(rc).Decode(dbData); err != nil {
		return fmt.Errorf("failed to decode database.json: %w", err)
	}
	return nil
}

// resetSchema clears the DB schema depending on the provider
func resetSchema(db *gorm.DB) error {
	switch common.EnvConfig.DbProvider {
	case common.DbProviderPostgres:
		var currentSchema string
		if err := db.Raw(`SELECT current_schema();`).Scan(&currentSchema).Error; err != nil {
			return fmt.Errorf("failed to get current schema: %w", err)
		}

		if err := db.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE; CREATE SCHEMA "%s";`, currentSchema, currentSchema)).Error; err != nil {
			return fmt.Errorf("failed to reset schema %s: %w", currentSchema, err)
		}

	case common.DbProviderSqlite:
		var tables []string
		if err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';`).
			Scan(&tables).Error; err != nil {
			return fmt.Errorf("failed to list SQLite tables: %w", err)
		}

		for _, t := range tables {
			if err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, t)).Error; err != nil {
				return fmt.Errorf("failed to drop SQLite table %s: %w", t, err)
			}
		}
	}
	return nil
}

// runMigrations migrates the DB schema to the appropriate version
func runMigrations(db *gorm.DB, targetVersion uint) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	driver, err := bootstrap.NewMigrationDriver(sqlDB)
	if err != nil {
		return err
	}

	path := filepath.Join("migrations", string(common.EnvConfig.DbProvider))
	source, err := iofs.New(resources.FS, path)
	if err != nil {
		return fmt.Errorf("failed to create embedded migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Migrate(targetVersion); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if strings.HasPrefix(err.Error(), "no migration found") {
			return fmt.Errorf("database version is newer than the latest supported version (%d) by the current Pocket ID version", targetVersion)
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// insertData populates the DB with the imported data
func insertData(db *gorm.DB, dbData databaseJson) error {
	// Disable foreign key checks for SQLite
	if common.EnvConfig.DbProvider == common.DbProviderSqlite {
		if err := toggleSqliteForeignKeyChecks(db, false); err != nil {
			return fmt.Errorf("failed to disable foreign keys: %w", err)
		}
		defer func() {
			if err := toggleSqliteForeignKeyChecks(db, true); err != nil {
				fmt.Printf("Warning: failed to re-enable foreign keys: %v\n", err)
			}
		}()
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Disable foreign key checks for Postgres
		if common.EnvConfig.DbProvider == common.DbProviderPostgres {
			if err := togglePostgresForeignKeyChecks(tx, false); err != nil {
				return fmt.Errorf("failed to disable foreign keys: %w", err)
			}
			defer func() {
				if err := togglePostgresForeignKeyChecks(tx, true); err != nil {
					fmt.Printf("Warning: failed to re-enable foreign keys: %v\n", err)
				}
			}()
		}

		// Insert rows
		for table, rows := range dbData.Tables {
			if table == "schema_migrations" {
				continue
			}

			for _, row := range rows {
				normalizeRow(row)
				if err := tx.Table(table).Create(row).Error; err != nil {
					return fmt.Errorf("failed inserting into %s: %w", table, err)
				}
			}
		}

		return nil
	})
}

// normalizeRow mutates a row so it round-trips correctly for SQLite
func normalizeRow(row map[string]any) {
	for k, v := range row {
		if m, ok := v.(map[string]any); ok {
			if b64, ok := m["__binary__"].(string); ok {
				if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
					row[k] = data
				}
			}
		}
	}
}

// extractIntoBase writes a file entry from the ZIP under baseDir, removing the given prefix
func extractIntoBase(f *zip.File, baseDir, stripPrefix string) error {
	const maxFileSize = 50 << 20 // 50 MiB is a very generous limit

	if f.UncompressedSize64 > maxFileSize {
		return fmt.Errorf("file %s too large (%d bytes)", f.Name, f.UncompressedSize64)
	}

	name := strings.TrimPrefix(f.Name, stripPrefix)
	if strings.HasSuffix(f.Name, "/") || name == "" {
		return nil // skip directories
	}

	// Validate path to prevent Zip Slip
	targetPath := filepath.Join(baseDir, filepath.FromSlash(name))
	if !strings.HasPrefix(targetPath, filepath.Clean(baseDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", f.Name)
	}

	// Clean up any existing file or directory at the target path
	_ = os.RemoveAll(targetPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", targetPath, err)
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	//nolint:gosec // f.UncompressedSize64 is capped above
	if _, err := io.CopyN(out, rc, int64(f.UncompressedSize64)); err != nil {
		return fmt.Errorf("copy failed for %s: %w", f.Name, err)
	}

	return nil
}

// absolutePathOrOriginal returns the absolute path of the given path,
// or the original path if it cannot be resolved
func absolutePathOrOriginal(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// togglePostgresForeignKeyChecks enables or disables foreign key checks in Postgres
// Must be called within a transaction
func togglePostgresForeignKeyChecks(tx *gorm.DB, enable bool) error {
	var tables []string
	if err := tx.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).
		Scan(&tables).Error; err != nil {
		return fmt.Errorf("failed to fetch postgres tables: %w", err)
	}

	action := "DISABLE"
	if enable {
		action = "ENABLE"
	}

	for _, t := range tables {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE "%s" %s TRIGGER ALL;`, t, action)).Error; err != nil {
			return fmt.Errorf("failed to %s triggers on %s: %w", strings.ToLower(action), t, err)
		}
	}

	return nil
}

// toggleSqliteForeignKeyChecks enables or disables foreign key checks in SQLite
// Must be called outside a transaction
func toggleSqliteForeignKeyChecks(db *gorm.DB, enable bool) error {
	value := "OFF"
	if enable {
		value = "ON"
	}

	return db.Exec(fmt.Sprintf("PRAGMA foreign_keys = %s;", value)).Error
}
