package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"gorm.io/gorm"

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
	Provider   string                      `json:"provider"`
	Version    uint                        `json:"version"`
	Tables     map[string][]map[string]any `json:"tables"`
	TableOrder []string                    `json:"tableOrder"`
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

	err = s.ImportDatabase(ctx, dbData)
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
func (s *ImportService) ImportDatabase(ctx context.Context, dbData DatabaseExport) error {
	err := s.resetSchema(ctx, dbData.Version)
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
func (s *ImportService) resetSchema(ctx context.Context, targetVersion uint) error {
	sqlDb, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Drop the existing schema
	if s.db.Name() == "sqlite" {
		// On SQLite, the connection pool forces `foreign_keys(1)` on every connection, so DROP TABLE performs an implicit DELETE that fires foreign-key cascades/triggers and fails depending on the drop order
		// golang-migrate's Drop() runs across the pool, so disabling foreign keys on a single Gorm connection wouldn't reach the connection it uses
		// Instead, drop every table on a single dedicated connection with foreign keys disabled
		err = dropAllTablesSQLite(ctx, sqlDb)
		if err != nil {
			return fmt.Errorf("failed to drop existing schema: %w", err)
		}
	} else {
		m, cleanup, err := utils.GetEmbeddedMigrateInstance(ctx, sqlDb)
		if err != nil {
			return fmt.Errorf("failed to get migrate instance: %w", err)
		}
		err = m.Drop()
		cleanup()
		if err != nil {
			return fmt.Errorf("failed to drop existing schema: %w", err)
		}
	}

	// Re-create the schema by migrating to the target version
	// The migration files manage their own foreign-key state where needed
	m, cleanup, err := utils.GetEmbeddedMigrateInstance(ctx, sqlDb)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}
	defer cleanup()

	err = m.Migrate(targetVersion)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// dropAllTablesSQLite drops every user table using a single dedicated connection with foreign-key enforcement disabled, so DROP TABLE doesn't trip foreign-key cascades or triggers
// foreign_keys is a per-connection pragma, so pinning to one connection guarantees it stays disabled for every drop
func dropAllTablesSQLite(ctx context.Context, sqlDb *sql.DB) error {
	conn, err := sqlDb.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	rows, err := conn.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			// We cannot defer as that would block the DROP TABLE queries
			//nolint:sqlclosecheck
			_ = rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return err
	}

	for _, t := range tables {
		_, err = conn.ExecContext(ctx, `DROP TABLE IF EXISTS "`+t+`"`)
		if err != nil {
			return fmt.Errorf("failed to drop table %q: %w", t, err)
		}
	}

	return nil
}

// insertData populates the DB with the imported data
func (s *ImportService) insertData(dbData DatabaseExport) error {
	schema, err := utils.LoadDBSchemaTypes(s.db)
	if err != nil {
		return fmt.Errorf("failed to load schema types: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Iterate through all tables
		// Some tables need to be processed in order
		tables := make([]string, 0, len(dbData.Tables))
		tables = append(tables, dbData.TableOrder...)

		for t := range dbData.Tables {
			// Skip tables already present where the order matters
			// Also skip the schema_migrations table
			if slices.Contains(dbData.TableOrder, t) || t == "schema_migrations" {
				continue
			}
			tables = append(tables, t)
		}

		// Insert rows
		for _, table := range tables {
			for _, row := range dbData.Tables[table] {
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
func normalizeRowWithSchema(row map[string]any, table string, schema utils.DBSchemaTypes) error {
	if schema[table] == nil {
		return fmt.Errorf("schema not found for table '%s'", table)
	}

	for col, val := range row {
		if val == nil {
			// If the value is nil, skip the column
			continue
		}

		colType := schema[table][col]

		switch colType.Name {
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
				return fmt.Errorf("failed to decode value for column '%s/%s' from base64: %w", table, col, err)
			}

			// For jsonb, we additionally cast to json.RawMessage
			if colType.Name == "jsonb" {
				row[col] = json.RawMessage(b)
			} else {
				row[col] = b
			}
		}
	}

	return nil
}
