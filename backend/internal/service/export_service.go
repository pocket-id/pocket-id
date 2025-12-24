package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// ExportService handles exporting Pocket ID data into a ZIP archive.
type ExportService struct {
	db      *gorm.DB
	storage storage.FileStorage
}

func NewExportService(db *gorm.DB, storage storage.FileStorage) *ExportService {
	return &ExportService{
		db:      db,
		storage: storage,
	}
}

// ExportToZip performs the full export process and writes the ZIP data to the given writer.
func (s *ExportService) ExportToZip(ctx context.Context, w io.Writer) error {
	dbData, err := s.extractDatabase()
	if err != nil {
		return err
	}

	return s.writeExportZipStream(ctx, w, dbData)
}

// extractDatabase reads all tables into a DatabaseExport struct
func (s *ExportService) extractDatabase() (DatabaseExport, error) {
	schema, err := utils.LoadDBSchemaTypes(s.db)
	if err != nil {
		return DatabaseExport{}, fmt.Errorf("failed to load schema types: %w", err)
	}

	version, err := s.schemaVersion()
	if err != nil {
		return DatabaseExport{}, err
	}

	out := DatabaseExport{
		Provider: s.db.Name(),
		Version:  version,
		Tables:   map[string][]map[string]any{},
		// These tables need to be inserted in a specific order because of foreign key constraints
		// Not all tables are listed here, because not all tables are order-dependent
		TableOrder: []string{"users", "user_groups", "oidc_clients", "signup_tokens"},
	}

	for table := range schema {
		if table == "storage" || table == "schema_migrations" {
			continue
		}
		err = s.dumpTable(table, schema[table], &out)
		if err != nil {
			return DatabaseExport{}, err
		}
	}

	return out, nil
}

func (s *ExportService) schemaVersion() (uint, error) {
	var version uint
	if err := s.db.Raw("SELECT version FROM schema_migrations").Row().Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}
	return version, nil
}

// dumpTable selects all rows from a table and appends them to out.Tables
func (s *ExportService) dumpTable(table string, types utils.DBSchemaTableTypes, out *DatabaseExport) error {
	rows, err := s.db.Raw("SELECT * FROM " + table).Rows()
	if err != nil {
		return fmt.Errorf("failed to read table %s: %w", table, err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	if len(cols) != len(types) {
		// Should never happen...
		return fmt.Errorf("mismatched columns in table (%d) and schema (%d)", len(cols), len(types))
	}

	for rows.Next() {
		vals := s.getScanValuesForTable(cols, types)
		err = rows.Scan(vals...)
		if err != nil {
			return fmt.Errorf("failed to scan row in table %s: %w", table, err)
		}

		rowMap := make(map[string]any, len(cols))
		for i, col := range cols {
			rowMap[col] = vals[i]
		}

		// Skip the app lock row in the kv table
		if table == "kv" {
			if keyPtr, ok := rowMap["key"].(*string); ok && keyPtr != nil && *keyPtr == lockKey {
				continue
			}
		}

		out.Tables[table] = append(out.Tables[table], rowMap)
	}

	return rows.Err()
}

func (s *ExportService) getScanValuesForTable(cols []string, types utils.DBSchemaTableTypes) []any {
	res := make([]any, len(cols))
	for i, col := range cols {
		// Store a pointer
		// Note: don't create a helper function for this switch, because it would return type "any" and mess everything up
		// If the column is nullable, we need a pointer to a pointer!
		switch types[col].Name {
		case "boolean", "bool":
			var x bool
			if types[col].Nullable {
				res[i] = utils.Ptr(utils.Ptr(x))
			} else {
				res[i] = utils.Ptr(x)
			}
		case "blob", "bytea", "jsonb":
			// Treat jsonb columns as binary too
			var x []byte
			if types[col].Nullable {
				res[i] = utils.Ptr(utils.Ptr(x))
			} else {
				res[i] = utils.Ptr(x)
			}
		case "timestamp", "timestamptz", "timestamp with time zone", "datetime":
			var x datatype.DateTime
			if types[col].Nullable {
				res[i] = utils.Ptr(utils.Ptr(x))
			} else {
				res[i] = utils.Ptr(x)
			}
		case "integer", "int", "bigint":
			var x int64
			if types[col].Nullable {
				res[i] = utils.Ptr(utils.Ptr(x))
			} else {
				res[i] = utils.Ptr(x)
			}
		default:
			// Treat everything else as a string (including the "numeric" type)
			var x string
			if types[col].Nullable {
				res[i] = utils.Ptr(utils.Ptr(x))
			} else {
				res[i] = utils.Ptr(x)
			}
		}
	}

	return res
}

func (s *ExportService) writeExportZipStream(ctx context.Context, w io.Writer, dbData DatabaseExport) error {
	zipWriter := zip.NewWriter(w)

	// Add database.json
	jsonWriter, err := zipWriter.Create("database.json")
	if err != nil {
		return fmt.Errorf("failed to create database.json in zip: %w", err)
	}

	jsonEncoder := json.NewEncoder(jsonWriter)
	jsonEncoder.SetEscapeHTML(false)

	if err := jsonEncoder.Encode(dbData); err != nil {
		return fmt.Errorf("failed to encode database.json: %w", err)
	}

	// Add uploaded files
	if err := s.addUploadsToZip(ctx, zipWriter); err != nil {
		return err
	}

	return zipWriter.Close()
}

// addUploadsToZip adds all files from the storage to the ZIP archive under the "uploads/" directory
func (s *ExportService) addUploadsToZip(ctx context.Context, zipWriter *zip.Writer) error {
	return s.storage.Walk(ctx, "/", func(p storage.ObjectInfo) error {
		zipPath := filepath.Join("uploads", p.Path)

		w, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", zipPath, err)
		}

		f, _, err := s.storage.Open(ctx, p.Path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", zipPath, err)
		}
		defer f.Close()

		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to copy file %s into zip: %w", zipPath, err)
		}
		return nil
	})
}
