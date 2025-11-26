package utils

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// DBTableExists checks if a table exists in the database
func DBTableExists(db *gorm.DB, tableName string) (exists bool, err error) {
	switch db.Name() {
	case "postgres":
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = ?
		)`
		err = db.Raw(query, tableName).Scan(&exists).Error
		if err != nil {
			return false, err
		}
	case "sqlite":
		query := `SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name=?`
		err = db.Raw(query, tableName).Scan(&exists).Error
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported database dialect: %s", db.Name())
	}

	return exists, nil
}

type DBSchemaColumn struct {
	Name     string
	Nullable bool
}
type DBSchemaTableTypes = map[string]DBSchemaColumn
type DBSchemaTypes = map[string]DBSchemaTableTypes

// LoadDBSchemaTypes retrieves the column types for all tables in the DB
// Result is a map of "table --> column --> {name: column type name, nullable: boolean}"
func LoadDBSchemaTypes(db *gorm.DB) (result DBSchemaTypes, err error) {
	result = make(DBSchemaTypes)

	switch db.Name() {
	case "postgres":
		var rows []struct {
			TableName  string
			ColumnName string
			DataType   string
			Nullable   bool
		}
		err := db.
			Raw(`
				SELECT table_name, column_name, data_type, is_nullable = 'YES' AS nullable
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
				result[r.TableName] = make(map[string]DBSchemaColumn)
			}
			result[r.TableName][r.ColumnName] = DBSchemaColumn{
				Name:     strings.ToLower(t),
				Nullable: r.Nullable,
			}
		}

	case "sqlite":
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
				Name    string
				Type    string
				Notnull bool
			}
			err := db.
				Raw(`PRAGMA table_info("` + table + `");`).
				Scan(&cols).
				Error
			if err != nil {
				return nil, err
			}
			for _, c := range cols {
				if result[table] == nil {
					result[table] = make(map[string]DBSchemaColumn)
				}
				result[table][c.Name] = DBSchemaColumn{
					Name:     strings.ToLower(c.Type),
					Nullable: !c.Notnull,
				}
			}
		}

	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", db.Name())
	}

	return result, nil
}
