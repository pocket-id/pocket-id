package cmds

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

type exportFlags struct {
	Path string
}

type databaseJson struct {
	Provider string                      `json:"provider"`
	Version  uint                        `json:"version"`
	Tables   map[string][]map[string]any `json:"tables"`
}

func init() {
	var flags exportFlags

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exports all data of Pocket ID into a ZIP file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(flags)
		},
	}

	exportCmd.Flags().StringVarP(&flags.Path, "path", "p", "./pocket-id-export.zip", "Path to the ZIP file to export the data to")

	rootCmd.AddCommand(exportCmd)
}

// runExport orchestrates the export flow
func runExport(flags exportFlags) error {
	db, err := bootstrap.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	dbData, err := extractDatabase(db)
	if err != nil {
		return err
	}

	if err := createExportZip(flags.Path, dbData); err != nil {
		return err
	}

	fmt.Printf("Exported data to %s\n", flags.Path)
	return nil
}

// extractDatabase reads all tables into a databaseJson struct
func extractDatabase(db *gorm.DB) (databaseJson, error) {
	var tables []string
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations';`).Scan(&tables).Error; err != nil {
			return databaseJson{}, fmt.Errorf("failed to query sqlite tables: %w", err)
		}

	case common.DbProviderPostgres:
		if err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename != 'schema_migrations';`).Scan(&tables).Error; err != nil {
			return databaseJson{}, fmt.Errorf("failed to query postgres tables: %w", err)
		}
	}

	var version uint
	if err := db.Raw("SELECT version FROM schema_migrations").Row().Scan(&version); err != nil {
		return databaseJson{}, fmt.Errorf("failed to query schema version: %w", err)
	}

	out := databaseJson{
		Provider: db.Name(),
		Version:  version,
		Tables:   map[string][]map[string]any{},
	}

	for _, table := range tables {
		if err := dumpTable(db, table, &out); err != nil {
			return databaseJson{}, err
		}
	}

	return out, nil
}

// dumpTable selects all rows from a table and appends them to out.Tables
func dumpTable(db *gorm.DB, table string, out *databaseJson) error {
	rows, err := db.Raw(fmt.Sprintf("SELECT * FROM %s", table)).Rows()
	if err != nil {
		return fmt.Errorf("failed to read table %s: %w", table, err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
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
			rowMap[col] = normalizeForJSON(vals[i])
		}
		out.Tables[table] = append(out.Tables[table], rowMap)
	}

	return rows.Err()
}

// createExportZip writes the database JSON and file assets into a ZIP archive
func createExportZip(path string, dbData databaseJson) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Write DB JSON
	if err := writeDatabaseJSON(zipWriter, dbData); err != nil {
		return err
	}

	// Add filesystem assets (uploads + keys)
	for _, basePath := range []string{common.EnvConfig.UploadPath, common.EnvConfig.KeysPath} {
		if err := addDirectoryToZip(zipWriter, basePath); err != nil {
			return err
		}
	}

	return nil
}

// writeDatabaseJSON adds database.json to the ZIP archive
func writeDatabaseJSON(zipWriter *zip.Writer, dbData databaseJson) error {
	jsonWriter, err := zipWriter.Create("database.json")
	if err != nil {
		return fmt.Errorf("failed to create database.json in zip: %w", err)
	}
	if err := json.NewEncoder(jsonWriter).Encode(dbData); err != nil {
		return fmt.Errorf("failed to encode database.json: %w", err)
	}
	return nil
}

// addDirectoryToZip recursively adds all files from basePath into the zip under the correct prefix
func addDirectoryToZip(zipWriter *zip.Writer, basePath string) error {
	if basePath == "" {
		return nil
	}
	prefix := "uploads"
	if basePath == common.EnvConfig.KeysPath {
		prefix = "keys"
	}

	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		zipPath := filepath.Join(prefix, relPath)

		w, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", zipPath, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to copy file %s into zip: %w", path, err)
		}
		return nil
	})
}

// normalizeForJSON ensures DB values round-trip through JSON safely
func normalizeForJSON(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		s := string(t)
		// Try JSON
		if isJSONObjectOrArray(s) {
			var j any
			if err := json.Unmarshal(t, &j); err == nil {
				return j
			}
		}
		// Try UTF-8 text
		if utf8.Valid(t) {
			return s
		}
		// Fallback: base64-encode as binary
		return map[string]any{"__binary__": base64.StdEncoding.EncodeToString(t)}

	case time.Time:
		return t.UTC().Format(time.RFC3339Nano)

	default:
		return t
	}
}

// isJSONObjectOrArray checks if a string looks like a JSON object or array
func isJSONObjectOrArray(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	return (s[0] == '{' && s[len(s)-1] == '}') ||
		(s[0] == '[' && s[len(s)-1] == ']')
}
