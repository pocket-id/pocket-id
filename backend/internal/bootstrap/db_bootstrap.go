package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/github"
	sqlinstrument "github.com/italypaleale/go-sql-utils/instrument"
	postgresinstrument "github.com/italypaleale/go-sql-utils/instrument/postgres"
	sqliteinstrument "github.com/italypaleale/go-sql-utils/instrument/sqlite"
	sqlitekit "github.com/italypaleale/go-sql-utils/sqlite"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/libtnb/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	gormMetrics "gorm.io/plugin/opentelemetry/metrics"

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

func ConnectDatabase(ctx context.Context) (db *gorm.DB, pg *pgxpool.Pool, err error) {
	var dialector gorm.Dialector

	// Choose the correct database provider
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for SQLite database")
		}

		sqliteutil.RegisterSqliteFunctions()

		// The connector validates the connection string and performs the filesystem setup SQLite needs: it creates the database and temporary directories
		// It also warns when the database lives on a networked filesystem, which is unsupported
		connector, err := sqlitekit.NewConnector(sqlitekit.ConnectOpts{
			ConnString: addSqliteDatetimeParams(common.EnvConfig.DbConnectionString),
			Logger:     slog.Default(),
		})
		if err != nil {
			return nil, nil, err
		}

		// We open the connection ourselves, rather than letting Gorm do it, so it goes through the instrumented driver
		// It also caps in-memory databases to a single connection, which they need to see the whole data
		sqliteDB, err := sqliteinstrument.Open(connector, sqlInstrumentOptions())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}

		dialector = sqlite.New(sqlite.Config{Conn: sqliteDB})
	case common.DbProviderPostgres:
		if common.EnvConfig.DbConnectionString == "" {
			return nil, nil, errors.New("missing required env var 'DB_CONNECTION_STRING' for Postgres database")
		}

		// We need a pgxpool object for francis, so we open this as a pgxpool...
		poolCfg, err := pgxpool.ParseConfig(common.EnvConfig.DbConnectionString)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse Postgres connection string: %w", err)
		}

		// ...with the instrumented tracer attached, chaining any tracer the connection string may have configured
		poolCfg.ConnConfig.Tracer = postgresinstrument.NewTracer(sqlInstrumentOptions(), poolCfg.ConnConfig.Tracer)

		pg, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Postgres pool: %w", err)
		}

		// Test it with a ping
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
			// Disable logging in Gorm because the driver itself is instrumented
			Logger: gormLogger.Discard,
		})
		if err == nil {
			slog.Info("Connected to database", slog.String("provider", string(common.EnvConfig.DbProvider)))

			conn, err := db.DB()
			if err != nil {
				if pg != nil {
					pg.Close()
				}
				return nil, nil, fmt.Errorf("failed to get *sql.DB connection from Gorm: %w", err)
			}

			// Report the metrics for the connection pool
			// Gorm's OpenTelemetry plugin is not used for this: it would wrap every statement span the instrumentation already emits in a second span
			gormMetrics.ReportDBStatsMetrics(conn)

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

// sqlInstrumentOptions returns the instrumentation applied to the database connection, shared by both providers
// This enables tracing in addition to logs
func sqlInstrumentOptions() *sqlinstrument.Options {
	return &sqlinstrument.Options{
		Log: slog.Default().With("scope", "sql"),
		// Logging every statement is only useful while debugging, and the instrumentation drops the records anyway unless the logger is at debug level
		QueryLog: common.EnvConfig.LogLevel == "debug",
		// Slow statements are worth a warning at any log level
		SlowThreshold: 250 * time.Millisecond,
	}
}

// addSqliteDatetimeParams adds the datetime parameters to a SQLite connection string, leaving any the user set explicitly alone.
func addSqliteDatetimeParams(connString string) string {
	// sqliteDatetimeParams are the DSN parameters the Gorm SQLite driver injects when it opens the connection itself.
	// Pocket ID opens the connection instead, to instrument it, so they have to be set here or modernc.org/sqlite stops returning time.Time for datetime columns.
	// See injectDSNParams in github.com/libtnb/sqlite.
	var sqliteDatetimeParams = map[string]string{
		"_texttotime":  "1",
		"_inttotime":   "1",
		"_time_format": "sqlite",
	}

	path, rawQuery, _ := strings.Cut(connString, "?")
	if path == "" {
		// Return the connection string so the driver reports an error
		return connString
	}

	qs, err := url.ParseQuery(rawQuery)
	if err != nil {
		// Return the connection string so the driver reports an error
		return connString
	}

	for k, v := range sqliteDatetimeParams {
		if len(qs[k]) == 0 {
			qs.Set(k, v)
		}
	}

	return path + "?" + qs.Encode()
}
