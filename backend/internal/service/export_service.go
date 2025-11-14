package service

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"gorm.io/gorm"
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
	tables, err := s.listTables()
	if err != nil {
		return DatabaseExport{}, err
	}

	version, err := s.schemaVersion()
	if err != nil {
		return DatabaseExport{}, err
	}

	out := DatabaseExport{
		Provider: s.db.Name(),
		Version:  version,
		Tables:   map[string][]map[string]any{},
	}

	for _, table := range tables {
		if err := s.dumpTable(table, &out); err != nil {
			return DatabaseExport{}, err
		}
	}

	return out, nil
}

func (s *ExportService) listTables() ([]string, error) {
	var tables []string

	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if err := s.db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations';`).Scan(&tables).Error; err != nil {
			return nil, fmt.Errorf("failed to query sqlite tables: %w", err)
		}

	case common.DbProviderPostgres:
		if err := s.db.Raw(`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname='public' AND tablename != 'schema_migrations';`).Scan(&tables).Error; err != nil {
			return nil, fmt.Errorf("failed to query postgres tables: %w", err)
		}
	}

	return tables, nil
}

func (s *ExportService) schemaVersion() (uint, error) {
	var version uint
	if err := s.db.Raw("SELECT version FROM schema_migrations").Row().Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}
	return version, nil
}

// dumpTable selects all rows from a table and appends them to out.Tables
func (s *ExportService) dumpTable(table string, out *DatabaseExport) error {
	rows, err := s.db.Raw("SELECT * FROM " + table).Rows()
	if err != nil {
		return fmt.Errorf("failed to read table %s: %w", table, err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()

	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("failed to scan row in table %s: %w", table, err)
		}

		rowMap := make(map[string]any, len(cols))
		for i, col := range cols {
			sqlType := colTypes[i].DatabaseTypeName()
			rowMap[col] = normalizeForJSON(vals[i], sqlType)
		}

		// Skip the app lock row in the kv table
		if table == "kv" {
			if keyVal, ok := rowMap["key"]; ok && keyVal == lockKey {
				continue
			}
		}

		out.Tables[table] = append(out.Tables[table], rowMap)
	}

	return rows.Err()
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

// normalizeForJSON ensures DB values round-trip through JSON safely
func normalizeForJSON(value any, columnType string) any {
	switch t := value.(type) {
	case nil:
		return nil
	case []byte:
		s := string(t)
		// Try UTF-8 text
		if utf8.Valid(t) {
			return s
		}
		// Fallback: base64-encode as binary
		return map[string]any{"__binary__": base64.StdEncoding.EncodeToString(t)}

	case time.Time:
		return t.Unix()

	case int64:
		if columnType == "BOOLEAN" {
			return t != 0
		}
		return t

	default:
		return t
	}
}
