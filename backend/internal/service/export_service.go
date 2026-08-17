package service

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// ExportService handles exporting Pocket ID data into a ZIP archive.
type ExportService struct {
	db      *gorm.DB
	storage storage.FileStorage
	actors  ActorsBackupProvider
}

// NewExportService creates a new ExportService.
//
// actors is used to back up the actor host's data, which is stored outside of the Pocket ID schema; when nil, the export does not include it.
func NewExportService(db *gorm.DB, storage storage.FileStorage, actors ActorsBackupProvider) *ExportService {
	return &ExportService{
		db:      db,
		storage: storage,
		actors:  actors,
	}
}

// ExportToZip performs the full export process and writes the ZIP data to the given writer.
func (s *ExportService) ExportToZip(ctx context.Context, w io.Writer) error {
	dbData, err := s.extractDatabase(ctx)
	if err != nil {
		return err
	}

	return s.writeExportZipStream(ctx, w, dbData)
}

// extractDatabase reads all tables into a DatabaseExport struct
//
// Every table is read inside a single read transaction, so the dump is a consistent snapshot of one point in time
func (s *ExportService) extractDatabase(ctx context.Context) (out DatabaseExport, err error) {
	err = s.db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			out, err = extractDatabaseTx(tx)
			return err
		}, snapshotTxOptions(s.db.Name()))
	if err != nil {
		return DatabaseExport{}, err
	}

	return out, nil
}

// snapshotTxOptions returns the transaction options that give a read a consistent snapshot on the given database provider, without blocking concurrent writers
func snapshotTxOptions(provider string) *sql.TxOptions {
	opts := &sql.TxOptions{
		ReadOnly: true,
	}

	// Postgres defaults to read committed, which takes a new snapshot for every statement, so tables read later in the export would include writes that landed after it started
	// Repeatable read instead pins a single snapshot for the whole transaction
	// SQLite is left at the driver's default: in WAL mode (the one used by Pocket ID) a read transaction already sees one consistent snapshot, and the driver rejects nothing but also honors no other isolation level
	if provider == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}

	return opts
}

// extractDatabaseTx dumps every exported table using the given transaction
func extractDatabaseTx(tx *gorm.DB) (DatabaseExport, error) {
	schema, err := utils.LoadDBSchemaTypes(tx)
	if err != nil {
		return DatabaseExport{}, fmt.Errorf("failed to load schema types: %w", err)
	}

	version, err := schemaVersion(tx)
	if err != nil {
		return DatabaseExport{}, err
	}

	out := DatabaseExport{
		Provider: tx.Name(),
		Version:  version,
		Tables:   map[string][]map[string]any{},
		// These tables need to be inserted in a specific order because of foreign key constraints
		// Not all tables are listed here, because not all tables are order-dependent
		TableOrder: []string{"users", "user_groups", "oidc_clients", "oauth2_sessions", "signup_tokens", "apis", "api_permissions", "oidc_clients_allowed_api_permissions"},
	}

	for table := range schema {
		// Skip internal tables and the actor host's own "francis_" tables
		if table == "storage" || table == "schema_migrations" || strings.HasPrefix(table, "francis_") {
			continue
		}
		err = dumpTable(tx, table, schema[table], &out)
		if err != nil {
			return DatabaseExport{}, err
		}
	}

	return out, nil
}

func schemaVersion(tx *gorm.DB) (uint, error) {
	var version uint
	err := tx.Raw("SELECT version FROM schema_migrations").Row().Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}
	return version, nil
}

// dumpTable selects all rows from a table and appends them to out.Tables
func dumpTable(tx *gorm.DB, table string, types utils.DBSchemaTableTypes, out *DatabaseExport) error {
	rows, err := tx.Raw("SELECT * FROM " + table).Rows()
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
		vals := getScanValuesForTable(cols, types)
		err = rows.Scan(vals...)
		if err != nil {
			return fmt.Errorf("failed to scan row in table %s: %w", table, err)
		}

		rowMap := make(map[string]any, len(cols))
		for i, col := range cols {
			rowMap[col] = vals[i]
		}

		out.Tables[table] = append(out.Tables[table], rowMap)
	}

	return rows.Err()
}

func getScanValuesForTable(cols []string, types utils.DBSchemaTableTypes) []any {
	res := make([]any, len(cols))
	for i, col := range cols {
		// Store a pointer
		// Note: don't create a helper function for this switch, because it would return type "any" and mess everything up
		// If the column is nullable, we need a pointer to a pointer!
		switch types[col].Name {
		case "boolean", "bool":
			var x bool
			if types[col].Nullable {
				res[i] = new(new(x))
			} else {
				res[i] = new(x)
			}
		case "blob", "bytea", "jsonb":
			// Treat jsonb columns as binary too
			var x []byte
			if types[col].Nullable {
				res[i] = new(new(x))
			} else {
				res[i] = new(x)
			}
		case "timestamp", "timestamptz", "timestamp with time zone", "datetime":
			var x datatype.DateTime
			if types[col].Nullable {
				res[i] = new(new(x))
			} else {
				res[i] = new(x)
			}
		case "integer", "int", "bigint":
			var x int64
			if types[col].Nullable {
				res[i] = new(new(x))
			} else {
				res[i] = new(x)
			}
		default:
			// Treat everything else as a string (including the "numeric" type)
			var x string
			if types[col].Nullable {
				res[i] = new(new(x))
			} else {
				res[i] = new(x)
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

	err = jsonEncoder.Encode(dbData)
	if err != nil {
		return fmt.Errorf("failed to encode database.json: %w", err)
	}

	// Add the actor host's data
	err = s.addActorsBackupToZip(ctx, zipWriter)
	if err != nil {
		return fmt.Errorf("error adding the actor host's data to the export zip: %w", err)
	}

	// Add uploaded files
	err = s.addUploadsToZip(ctx, zipWriter)
	if err != nil {
		return fmt.Errorf("error adding uploads to the export zip: %w", err)
	}

	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("error closing the zip writer: %w", err)
	}

	return nil
}

// addActorsBackupToZip adds the actor host's data (actor state, alarms, and dead-lettered jobs) to the ZIP archive as a Francis backup stream
// That data lives in the actor host's own tables, so it's exported through Francis' own portable backup format instead
func (s *ExportService) addActorsBackupToZip(ctx context.Context, zipWriter *zip.Writer) error {
	if s.actors == nil {
		return nil
	}

	w, err := zipWriter.Create(ActorsBackupFileName)
	if err != nil {
		return fmt.Errorf("failed to create %s in zip: %w", ActorsBackupFileName, err)
	}

	err = s.actors.Backup(ctx, w)
	if err != nil {
		return fmt.Errorf("failed to back up the actor host's data: %w", err)
	}

	return nil
}

// addUploadsToZip adds all files from the storage to the ZIP archive under the "uploads/" directory
func (s *ExportService) addUploadsToZip(ctx context.Context, zipWriter *zip.Writer) error {
	return s.storage.Walk(ctx, "/", func(p storage.ObjectInfo) error {
		zipPath := filepath.Join("uploads", p.Path)

		w, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for '%s': %w", zipPath, err)
		}

		f, _, err := s.storage.Open(ctx, p.Path)
		if err != nil {
			return fmt.Errorf("failed to open file '%s': %w", zipPath, err)
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		if err != nil {
			return fmt.Errorf("failed to copy file '%s' into zip: %w", zipPath, err)
		}
		return nil
	})
}
