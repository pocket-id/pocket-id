package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/italypaleale/francis/components/postgres"
	"github.com/italypaleale/francis/host/local"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

type NewActorsOpts struct {
	EnvConfig *common.EnvConfigSchema
	SQLite    *sql.DB
	Postgres  *pgxpool.Pool
}

func NewActors(o NewActorsOpts) (*local.Host, error) {
	log := slog.Default()

	// Derive a PSK from the global encryption key
	// The runtime PSK derives the cluster CA used for host-to-host mTLS
	psk, err := crypto.DeriveKey(o.EnvConfig, "pocketid/actors-psk")
	if err != nil {
		return nil, fmt.Errorf("failed to derive PSK: %w", err)
	}

	// Options for the host
	opts := []local.HostOption{
		local.WithAddress(net.JoinHostPort(o.EnvConfig.ActorsHost, o.EnvConfig.ActorsPort)),
		local.WithLogger(log.With("scope", "actor-host")),
		local.WithRuntimePSKs(psk),
		local.WithShutdownGracePeriod(10 * time.Second),
	}

	// Add the database connection
	switch {
	case o.Postgres != nil && o.SQLite != nil:
		return nil, errors.New("cannot have both Postgres and SQLite connections")
	case o.Postgres != nil:
		opts = append(opts, local.WithPostgresProvider(postgres.PostgresProviderOptions{
			DB: o.Postgres,
		}))
	case o.SQLite != nil:
		opts = append(opts, local.WithSQLiteProvider(local.SQLiteProviderOptions{
			DB: o.SQLite,
		}))
	default:
		return nil, errors.New("one of Postgres and SQLite must be set")
	}

	// Create a new actor host
	h, err := local.NewHost(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor host: %w", err)
	}

	return h, nil
}
