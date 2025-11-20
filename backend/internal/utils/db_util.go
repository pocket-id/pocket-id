package utils

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ToggleDBForeignKeyChecks enables/disables Foreign Key checks in the database
// db should hold an active transaction
func ToggleDBForeignKeyChecks(db *gorm.DB, enable bool) error {
	switch db.Name() {
	case "postgres":
		return togglePostgresForeignKeyChecks(db, enable)
	case "sqlite":
		return toggleSqliteForeignKeyChecks(db, enable)
	default:
		// Indicates a development-time error
		return fmt.Errorf("unsupported database dialect: %s", db.Name())
	}
}

func togglePostgresForeignKeyChecks(db *gorm.DB, enable bool) error {
	var tables []string
	err := db.
		Raw(`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public'`).
		Scan(&tables).Error
	if err != nil {
		return fmt.Errorf("failed to fetch postgres tables: %w", err)
	}
	action := "DISABLE"
	if enable {
		action = "ENABLE"
	}
	for _, t := range tables {
		err = db.
			Exec(fmt.Sprintf(`ALTER TABLE "%s" %s TRIGGER ALL;`, t, action)).
			Error
		if err != nil {
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
	return db.Exec("PRAGMA foreign_keys = " + value + ";").Error
}
