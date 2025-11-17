package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

// ImportService handles importing Pocket ID data from an exported ZIP archive.
type ImportService struct {
	db      *gorm.DB
	storage storage.FileStorage
}

type DatabaseExport struct {
	Provider string                      `json:"provider"`
	Version  uint                        `json:"version"`
	Tables   map[string][]map[string]any `json:"tables"`
}

func NewImportService(db *gorm.DB, storage storage.FileStorage) *ImportService {
	return &ImportService{
		db:      db,
		storage: storage,
	}
}

// ImportFromZip performs the full import process from the given ZIP reader.
func (s *ImportService) ImportFromZip(ctx context.Context, r *zip.Reader) error {
	dbData, err := processZipDatabaseJson(r.File)
	if err != nil {
		return err
	}

	if err := s.ImportDatabase(dbData); err != nil {
		return err
	}

	if err := s.importUploads(ctx, r.File); err != nil {
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

// importUploads imports files from the uploads/ directory in the ZIP archive
func (s *ImportService) importUploads(ctx context.Context, files []*zip.File) error {
	const maxFileSize = 50 << 20 // 50 MiB
	const uploadsPrefix = "uploads/"

	for _, f := range files {
		if !strings.HasPrefix(f.Name, uploadsPrefix) {
			continue
		}

		if f.UncompressedSize64 > maxFileSize {
			return fmt.Errorf("file %s too large (%d bytes)", f.Name, f.UncompressedSize64)
		}

		targetPath := strings.TrimPrefix(f.Name, uploadsPrefix)
		if strings.HasSuffix(f.Name, "/") || targetPath == "" {
			continue // Skip directories
		}

		if err := s.storage.DeleteAll(ctx, targetPath); err != nil {
			return fmt.Errorf("failed to delete existing file %s: %w", targetPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		buf, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read file %s: %w", f.Name, err)
		}

		if err := s.storage.Save(ctx, targetPath, bytes.NewReader(buf)); err != nil {
			return fmt.Errorf("failed to save file %s: %w", targetPath, err)
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
		if err := setSqliteForeignKeyChecks(s.db, false); err != nil {
			return fmt.Errorf("failed to disable foreign keys: %w", err)
		}
		defer func() {
			_ = setSqliteForeignKeyChecks(s.db, true)
		}()
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Disable foreign key checks for Postgres
		if common.EnvConfig.DbProvider == common.DbProviderPostgres {
			if err := setPostgresForeignKeyChecks(tx, false); err != nil {
				return fmt.Errorf("failed to disable foreign keys: %w", err)
			}
			defer func() {
				_ = setPostgresForeignKeyChecks(tx, true)
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
			if v, ok := val.(float64); ok {
				row[col] = time.Unix(int64(v), 0).UTC()
			}
		}
	}
}

// setPostgresForeignKeyChecks enables/disables FK checks in Postgres
func setPostgresForeignKeyChecks(tx *gorm.DB, enable bool) error {
	var tables []string
	if err := tx.Raw(`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public'`).
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

// setSqliteForeignKeyChecks enables/disables FK checks in SQLite
func setSqliteForeignKeyChecks(db *gorm.DB, enable bool) error {
	value := "OFF"
	if enable {
		value = "ON"
	}
	return db.Exec("PRAGMA foreign_keys = " + value + ";").Error
}
