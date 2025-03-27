package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/resources"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDatabase() (db *gorm.DB) {
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}

	// Choose the correct driver for the database provider
	var driver database.Driver
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		driver, err = sqliteMigrate.WithInstance(sqlDb, &sqliteMigrate.Config{})
	case common.DbProviderPostgres:
		driver, err = postgresMigrate.WithInstance(sqlDb, &postgresMigrate.Config{})
	default:
		log.Fatalf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}
	if err != nil {
		log.Fatalf("failed to create migration driver: %v", err)
	}

	// Run migrations
	if err := migrateDatabase(driver); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func migrateDatabase(driver database.Driver) error {
	// Use the embedded migrations
	source, err := iofs.New(resources.FS, "migrations/"+string(common.EnvConfig.DbProvider))
	if err != nil {
		return fmt.Errorf("failed to create embedded migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pocket-id", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func connectDatabase() (db *gorm.DB, err error) {
	var dialector gorm.Dialector

	// Choose the correct database provider
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		dialector = sqlite.Open(common.EnvConfig.SqliteDBPath)
	case common.DbProviderPostgres:
		dialector = postgres.Open(common.EnvConfig.PostgresConnectionString)
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}

	for i := 1; i <= 3; i++ {
		db, err = gorm.Open(dialector, &gorm.Config{
			TranslateError: true,
			Logger:         getLogger(),
		})
		if err == nil {
			break
		} else {
			log.Printf("Attempt %d: Failed to initialize database. Retrying...", i)
			time.Sleep(3 * time.Second)
		}
	}

	return db, err
}

func getLogger() logger.Interface {
	isProduction := common.EnvConfig.AppEnv == "production"

	var logLevel logger.LogLevel
	if isProduction {
		logLevel = logger.Error
	} else {
		logLevel = logger.Info
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: isProduction,
			ParameterizedQueries:      isProduction,
			Colorful:                  !isProduction,
		},
	)
}
