package bootstrap

import (
	"log"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// Performs the migration of the database connection string
// See: https://github.com/pocket-id/pocket-id/pull/388
func migrateConfigDBConnstring() {
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		// Check if we're using the deprecated SqliteDBPath env var
		if common.EnvConfig.SqliteDBPath != "" {
			connString := "file:" + common.EnvConfig.SqliteDBPath + "?_journal_mode=WAL&_busy_timeout=2500&_txlock=immediate"
			common.EnvConfig.DbConnectionString = connString
			common.EnvConfig.SqliteDBPath = ""

			log.Printf("[WARN] Env var 'SQLITE_DB_PATH' is deprecated - use 'DB_CONNECTION_STRING' instead with the value: '%s'", connString)
		}
	case common.DbProviderPostgres:
		// Check if we're using the deprecated PostgresConnectionString alias
		if common.EnvConfig.PostgresConnectionString != "" {
			common.EnvConfig.DbConnectionString = common.EnvConfig.PostgresConnectionString
			common.EnvConfig.PostgresConnectionString = ""

			log.Print("[WARN] Env var 'POSTGRES_CONNECTION_STRING' is deprecated - use 'DB_CONNECTION_STRING' instead with the same value")
		}
	default:
		// We don't do anything here in the default case
		// This is an error, but will be handled later on
	}
}
