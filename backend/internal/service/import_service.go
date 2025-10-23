package service

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
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

// ImportService handles importing Pocket ID data from an exported ZIP archive.
type ImportService struct {
	db *gorm.DB
}

type DatabaseExport struct {
	Provider string                      `json:"provider"`
	Version  uint                        `json:"version"`
	Tables   map[string][]map[string]any `json:"tables"`
}

func NewImportService(db *gorm.DB) *ImportService {
	return &ImportService{db: db}
}

// ImportFromZip performs the full import process from the given ZIP reader.
func (s *ImportService) ImportFromZip(r *zip.Reader) error {
	dbData, err := processZipDatabaseJson(r.File)
	if err != nil {
		return err
	}

	if err := s.ImportDatabase(dbData); err != nil {
		return err
	}

	if err := processZipFiles(r.File); err != nil {
		return err
	}

	return nil
}

// ImportDatabase only imports the database data from the given DatabaseExport struct.
func (s *ImportService) ImportDatabase(dbData DatabaseExport) error {
	if err := s.resetSchema(dbData.Version, dbData.Provider); err != nil {
		return err
	}

	if err := s.insertData(dbData); err != nil {
		return err
	}

	return nil
}

// processZipDatabaseJson extracts database.json from the ZIP archive
func processZipDatabaseJson(files []*zip.File) (dbData DatabaseExport, err error) {
	for _, f := range files {
		if f.Name == "database.json" {
			rc, err := f.Open()
			if err != nil {
				return dbData, fmt.Errorf("failed to open database.json: %w", err)
			}

			if err := json.NewDecoder(rc).Decode(&dbData); err != nil {
				_ = rc.Close()
				return dbData, fmt.Errorf("failed to decode database.json: %w", err)
			}

			_ = rc.Close()
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

// resetSchema drops the existing schema and migrates to the target version
func (s *ImportService) resetSchema(targetVersion uint, exportDbProvider string) error {
	sqlDb, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	m, err := utils.GetEmbeddedMigrateInstance(sqlDb)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	if err = m.Drop(); err != nil {
		return fmt.Errorf("failed to drop existing schema: %w", err)
	}

	// Needs to be called again to re-create the schema_migrations table
	m, err = utils.GetEmbeddedMigrateInstance(sqlDb)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	if err := m.Migrate(targetVersion); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// Special case: If no migrations are found, it may be due to a different DB provider.
		// In that case we just apply all migrations from the current provider.
		if strings.HasPrefix(err.Error(), "no migration found") && exportDbProvider != string(common.EnvConfig.DbProvider) {
			if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("migration failed: %w", err)
			}
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// loadSchemaTypes retrieves the column types for all tables in the DB
func loadSchemaTypes(db *gorm.DB) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)

	switch common.EnvConfig.DbProvider {
	case common.DbProviderPostgres:
		var rows []struct {
			TableName  string
			ColumnName string
			DataType   string
		}
		if err := db.Raw(`
			SELECT table_name, column_name, data_type
			FROM information_schema.columns
			WHERE table_schema = 'public';
		`).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			t := strings.ToLower(r.DataType)
			if result[r.TableName] == nil {
				result[r.TableName] = make(map[string]string)
			}
			result[r.TableName][r.ColumnName] = t
		}

	case common.DbProviderSqlite:
		var tables []string
		if err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';`).Scan(&tables).Error; err != nil {
			return nil, err
		}
		for _, table := range tables {
			var cols []struct {
				Name string
				Type string
			}
			if err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s);", table)).Scan(&cols).Error; err != nil {
				return nil, err
			}
			for _, c := range cols {
				if result[table] == nil {
					result[table] = make(map[string]string)
				}
				result[table][c.Name] = strings.ToLower(c.Type)
			}
		}
	}
	return result, nil
}

// insertData populates the DB with the imported data
func (s *ImportService) insertData(dbData DatabaseExport) error {
	schema, err := loadSchemaTypes(s.db)
	if err != nil {
		return fmt.Errorf("failed to load schema types: %w", err)
	}

	// Disable foreign key checks for SQLite
	if common.EnvConfig.DbProvider == common.DbProviderSqlite {
		if err := toggleSqliteForeignKeyChecks(s.db, false); err != nil {
			return fmt.Errorf("failed to disable foreign keys: %w", err)
		}
		defer func() {
			_ = toggleSqliteForeignKeyChecks(s.db, true)
		}()
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Disable foreign key checks for Postgres
		if common.EnvConfig.DbProvider == common.DbProviderPostgres {
			if err := togglePostgresForeignKeyChecks(tx, false); err != nil {
				return fmt.Errorf("failed to disable foreign keys: %w", err)
			}
			defer func() {
				_ = togglePostgresForeignKeyChecks(tx, true)
			}()
		}

		// Insert rows
		for table, rows := range dbData.Tables {
			if table == "schema_migrations" {
				continue
			}

			for _, row := range rows {
				normalizeRowWithSchema(row, table, schema)
				if err := tx.Table(table).Create(row).Error; err != nil {
					return fmt.Errorf("failed inserting into %s: %w", table, err)
				}
			}
		}
		return nil
	})
}

// normalizeRowWithSchema converts row values based on the DB schema
func normalizeRowWithSchema(row map[string]any, table string, schema map[string]map[string]string) {
	for col, val := range row {
		// Decode binary blobs
		if m, ok := val.(map[string]any); ok {
			if b64, ok := m["__binary__"].(string); ok {
				if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
					row[col] = data
					continue
				}
			}
		}

		colType := ""
		if schema[table] != nil {
			colType = schema[table][col]
		}

		if strings.Contains(colType, "timestamp") || strings.Contains(colType, "datetime") {
			switch v := val.(type) {
			case float64:
				if v > 1e9 && v < 1e12 {
					row[col] = time.Unix(int64(v), 0).UTC()
				} else if v >= 1e12 {
					row[col] = time.UnixMilli(int64(v)).UTC()
				}
			case string:
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					row[col] = t.UTC()
				}
			}
		}
	}
}

// extractIntoBase writes a file entry from the ZIP under baseDir, removing the given prefix
func extractIntoBase(f *zip.File, baseDir, stripPrefix string) error {
	const maxFileSize = 50 << 20 // 50 MiB
	if f.UncompressedSize64 > maxFileSize {
		return fmt.Errorf("file %s too large (%d bytes)", f.Name, f.UncompressedSize64)
	}

	name := strings.TrimPrefix(f.Name, stripPrefix)
	if strings.HasSuffix(f.Name, "/") || name == "" {
		return nil // skip directories
	}

	targetPath := filepath.Join(baseDir, filepath.FromSlash(name))
	if !strings.HasPrefix(targetPath, filepath.Clean(baseDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", f.Name)
	}

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

// togglePostgresForeignKeyChecks enables/disables FK checks in Postgres
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

// toggleSqliteForeignKeyChecks enables/disables FK checks in SQLite
func toggleSqliteForeignKeyChecks(db *gorm.DB, enable bool) error {
	value := "OFF"
	if enable {
		value = "ON"
	}
	return db.Exec(fmt.Sprintf("PRAGMA foreign_keys = %s;", value)).Error
}
