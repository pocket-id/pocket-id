package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/italypaleale/francis/components/postgres"
	"github.com/italypaleale/francis/host/local"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

type NewActorsOpts struct {
	EnvConfig  *common.EnvConfigSchema
	SQLite     *sql.DB
	Postgres   *pgxpool.Pool
	AppConfig  *service.AppConfigService
	HttpClient *http.Client
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

	// Add all cron jobs
	cronjobs, err := o.getCronJobs()
	if err != nil {
		return nil, err
	}
	opts = append(opts, cronjobs...)

	// Add the database connection
	providerOpt, err := o.getProvider()
	if err != nil {
		return nil, err
	}
	opts = append(opts, providerOpt)

	// Create a new actor host
	h, err := local.NewHost(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor host: %w", err)
	}

	return h, nil
}

func (o *NewActorsOpts) getProvider() (local.HostOption, error) {
	switch {
	case o.Postgres != nil && o.SQLite != nil:
		return nil, errors.New("cannot have both Postgres and SQLite connections")
	case o.Postgres != nil:
		return local.WithPostgresProvider(postgres.PostgresProviderOptions{
			DB: o.Postgres,
		}), nil
	case o.SQLite != nil:
		return local.WithSQLiteProvider(local.SQLiteProviderOptions{
			DB: o.SQLite,
		}), nil
	default:
		return nil, errors.New("one of Postgres and SQLite must be set")
	}
}

func (o *NewActorsOpts) getCronJobs() (opts []local.HostOption, err error) {
	// In test mode, we do not register anything
	if common.EnvConfig.AppEnv != "test" {
		return opts, nil
	}

	// Register the analytics cron job
	analyticsJob, err := job.GetAnalyticsJob(o.AppConfig, o.HttpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics cron job: %w", err)
	}
	if analyticsJob != nil {
		// This could be nil if analytics are disabled
		opts = append(opts, local.WithBuiltInActor(analyticsJob))
	}

	return opts, nil
}
