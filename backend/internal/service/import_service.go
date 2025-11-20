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

	"github.com/golang-migrate/migrate/v4"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
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

	err = s.ImportDatabase(dbData)
	if err != nil {
		return err
	}

	err = s.importUploads(ctx, r.File)
	if err != nil {
		return err
	}

	return nil
}

// ImportDatabase only imports the database data from the given DatabaseExport struct.
func (s *ImportService) ImportDatabase(dbData DatabaseExport) error {
	err := s.resetSchema(dbData.Version, dbData.Provider)
	if err != nil {
		return err
	}

	err = s.insertData(dbData)
	if err != nil {
		return err
	}

	return nil
}

// processZipDatabaseJson extracts database.json from the ZIP archive
func processZipDatabaseJson(files []*zip.File) (dbData DatabaseExport, err error) {
	for _, f := range files {
		if f.Name == "database.json" {
			return parseDatabaseJsonStream(f)
		}
	}
	return dbData, errors.New("database.json not found in the ZIP file")
}

func parseDatabaseJsonStream(f *zip.File) (dbData DatabaseExport, err error) {
	rc, err := f.Open()
	if err != nil {
		return dbData, fmt.Errorf("failed to open database.json: %w", err)
	}
	defer rc.Close()

	err = json.NewDecoder(rc).Decode(&dbData)
	if err != nil {
		return dbData, fmt.Errorf("failed to decode database.json: %w", err)
	}

	return dbData, nil
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

		err := s.storage.DeleteAll(ctx, targetPath)
		if err != nil {
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

		err = s.storage.Save(ctx, targetPath, bytes.NewReader(buf))
		if err != nil {
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

	err = m.Drop()
	if err != nil {
		return fmt.Errorf("failed to drop existing schema: %w", err)
	}

	// Needs to be called again to re-create the schema_migrations table
	m, err = utils.GetEmbeddedMigrateInstance(sqlDb)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	err = m.Migrate(targetVersion)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// Special case: If no migrations are found, it may be due to a different DB provider.
		// In that case we just apply all migrations from the current provider.
		if strings.HasPrefix(err.Error(), "no migration found") && exportDbProvider != string(common.EnvConfig.DbProvider) {
			err = m.Up()
			if err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("migration failed: %w", err)
			}
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// loadSchemaTypes retrieves the column types for all tables in the DB
func loadSchemaTypes(db *gorm.DB) (result map[string]map[string]string, err error) {
	result = make(map[string]map[string]string)

	switch common.EnvConfig.DbProvider {
	case common.DbProviderPostgres:
		var rows []struct {
			TableName  string
			ColumnName string
			DataType   string
		}
		err := db.
			Raw(`
				SELECT table_name, column_name, data_type
				FROM information_schema.columns
				WHERE table_schema = 'public';
			`).
			Scan(&rows).
			Error
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			t := strings.ToLower(r.DataType)
			if result[r.TableName] == nil {
				result[r.TableName] = make(map[string]string)
			}
			result[r.TableName][r.ColumnName] = strings.ToLower(t)
		}

	case common.DbProviderSqlite:
		var tables []string
		err = db.
			Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';`).
			Scan(&tables).
			Error
		if err != nil {
			return nil, err
		}
		for _, table := range tables {
			var cols []struct {
				Name string
				Type string
			}
			err := db.
				Raw(fmt.Sprintf("PRAGMA table_info(%s);", table)).
				Scan(&cols).
				Error
			if err != nil {
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

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Disable foreign key checks
		err := utils.ToggleDBForeignKeyChecks(tx, false)
		if err != nil {
			return fmt.Errorf("failed to disable foreign keys: %w", err)
		}
		defer func() {
			_ = utils.ToggleDBForeignKeyChecks(tx, true)
		}()

		// Insert rows
		for table, rows := range dbData.Tables {
			if table == "schema_migrations" {
				continue
			}

			for _, row := range rows {
				err = normalizeRowWithSchema(row, table, schema)
				if err != nil {
					return fmt.Errorf("failed to normalize row for table '%s': %w", table, err)
				}
				err = tx.Table(table).Create(row).Error
				if err != nil {
					return fmt.Errorf("failed inserting into table '%s': %w", table, err)
				}
			}
		}
		return nil
	})
}

// normalizeRowWithSchema converts row values based on the DB schema
func normalizeRowWithSchema(row map[string]any, table string, schema map[string]map[string]string) error {
	if schema[table] == nil {
		return fmt.Errorf("schema not found for table '%s'", table)
	}

	fmt.Println("TABLE:", table)
	for col, val := range row {
		colType := schema[table][col]
		fmt.Printf("BEFORE %s (%s) - %T %v\n", col, colType, val, val)

		switch colType {
		case "timestamp", "timestamptz", "timestamp with time zone", "datetime":
			// Dates are stored as strings
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("value for column '%s/%s' was expected to be a string, but was '%T'", table, col, val)
			}
			d, err := datatype.DateTimeFromString(str)
			if err != nil {
				return fmt.Errorf("failed to decode value for column '%s/%s' as timestamp: %w", table, col, err)
			}
			row[col] = d

		case "blob", "bytea", "jsonb":
			// Binary data and jsonb data is stored in the file as base64-encoded string
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("value for column '%s/%s' was expected to be a string, but was '%T'", table, col, val)
			}
			b, err := base64.StdEncoding.DecodeString(str)
			if err != nil {
				return fmt.Errorf("failed to decode value for column '%s/%s' from hex: %w", table, col, err)
			}

			// For jsonb, we additionally cast to json.RawMessage
			if colType == "jsonb" {
				row[col] = json.RawMessage(b)
			} else {
				row[col] = b
			}
		}

		fmt.Printf("AFTER %s (%s) - %T %v\n", col, colType, row[col], row[col])
	}

	return nil
}
