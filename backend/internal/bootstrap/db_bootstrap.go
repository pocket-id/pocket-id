package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/github"
	sqlitekit "github.com/italypaleale/go-sql-utils/sqlite"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/libtnb/sqlite"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	sqliteutil "github.com/pocket-id/pocket-id/backend/internal/utils/sqlite"
)

func NewDatabase(ctx context.Context) (db *gorm.DB, pg *pgxpool.Pool, err error) {
	db, pg, err = ConnectDatabase(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Run migrations
	err = utils.MigrateDatabase(ctx, sqlDb)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, pg, nil
}

//nolint:gocognit
func ConnectDatabase(ctx context.Context) (db *gorm.DB, pg *pgxpool.Pool, err error) {
	var dialector gorm.Dialector

	// Choose the correct database provider
	var onConnFn func(conn *sql.DB)
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for SQLite database")
		}

		sqliteutil.RegisterSqliteFunctions()

		connString, dbPath, isMemoryDB, err := sqlitekit.ParseConnectionString(common.EnvConfig.DbConnectionString, slog.Default())
		if err != nil {
			return nil, nil, err
		}

		if !isMemoryDB {
			err = sqlitekit.EnsureDatabaseDir(dbPath)
			if err != nil {
				return nil, nil, err
			}

			var sqliteNetworkFilesystem bool
			sqliteNetworkFilesystem, err = sqlitekit.IsNetworkedFileSystem(filepath.Dir(dbPath))
			if err != nil {
				// Log the error only
				slog.Warn("Failed to detect filesystem type for the SQLite database directory", slog.String("path", filepath.Dir(dbPath)), slog.Any("error", err))
			} else if sqliteNetworkFilesystem {
				slog.Warn("⚠️⚠️⚠️ SQLite databases should not be stored on a networked file system like NFS, SMB, or FUSE, as there's a risk of crashes and even database corruption", slog.String("path", filepath.Dir(dbPath)))
			}
		}

		// Before we connect, also make sure that there's a temporary folder for SQLite to write its data
		err = sqlitekit.EnsureTempDir(filepath.Dir(dbPath), slog.Default())
		if err != nil {
			return nil, nil, err
		}

		if isMemoryDB {
			// For in-memory SQLite databases, we must limit to 1 open connection at the same time, or they won't see the whole data
			// The other workaround, of using shared caches, doesn't work well with multiple write transactions trying to happen at once
			onConnFn = func(conn *sql.DB) {
				conn.SetMaxOpenConns(1)
			}
		}

		dialector = sqlite.Open(connString)
	case common.DbProviderPostgres:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}

		// We need a pgxpool object for francis, so we open this as a pgxpool...
		pg, err = pgxpool.New(ctx, common.EnvConfig.DbConnectionString)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Postgres pool: %w", err)
		}

		// ...test it with a ping...
		pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
		defer pingCancel()
		err = pg.Ping(pingCtx)
		if err != nil {
			pg.Close()
			return nil, nil, fmt.Errorf("failed to ping Postgres database: %w", err)
		}

		// ...then create the dialector by adapting it to *sql.DB
		dialector = postgres.New(postgres.Config{
			Conn: stdlib.OpenDBFromPool(pg),
		})
	default:
		return nil, nil, fmt.Errorf("unsupported database provider: %s", common.EnvConfig.DbProvider)
	}

	// Try connecting up to 3 times
	for i := 1; i <= 3; i++ {
		db, err = gorm.Open(dialector, &gorm.Config{
			TranslateError: true,
			Logger:         getGormLogger(),
		})
		if err == nil {
			slog.Info("Connected to database", slog.String("provider", string(common.EnvConfig.DbProvider)))

			// Invoke the onConnFn callback if any
			if onConnFn != nil {
				conn, err := db.DB()
				if err != nil {
					if pg != nil {
						pg.Close()
					}
					return nil, nil, fmt.Errorf("failed to get database connection for onConnFn callback: %w", err)
				}

				onConnFn(conn)
			}

			return db, pg, nil
		}

		// If we're here, the connection failed
		slog.Warn("Failed to connect to database, will retry in 3s", slog.Int("attempt", i), slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))
		time.Sleep(3 * time.Second)
	}

	slog.Error("Failed to connect to database after 3 attempts", slog.String("provider", string(common.EnvConfig.DbProvider)), slog.Any("error", err))

	if pg != nil {
		pg.Close()
	}

	return nil, nil, err
}

func getGormLogger() gormLogger.Interface {
	loggerOpts := make([]slogGorm.Option, 0, 5)
	loggerOpts = append(loggerOpts,
		slogGorm.WithSlowThreshold(200*time.Millisecond),
		slogGorm.WithErrorField("error"),
	)

	if common.EnvConfig.LogLevel == "debug" {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelDebug),
			slogGorm.WithRecordNotFoundError(),
			slogGorm.WithTraceAll(),
		)

	} else {
		loggerOpts = append(loggerOpts,
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelWarn),
			slogGorm.WithIgnoreTrace(),
		)
	}

	return slogGorm.New(loggerOpts...)
}
