package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/italypaleale/francis/builtin/ratelimit"
	"github.com/italypaleale/francis/components/postgres"
	"github.com/italypaleale/francis/host/local"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

type NewActorsOpts struct {
	SQLite   *sql.DB
	Postgres *pgxpool.Pool

	EnvConfig   *common.EnvConfigSchema
	AppConfig   *service.AppConfigService
	HttpClient  *http.Client
	DB          *gorm.DB
	FileStorage storage.FileStorage
}

func NewActors(o NewActorsOpts) (*local.Host, map[string]*ratelimit.RateLimitService, error) {
	log := slog.Default()

	// Derive a PSK from the global encryption key
	// The runtime PSK derives the cluster CA used for host-to-host mTLS
	psk, err := crypto.DeriveKey(o.EnvConfig, "pocketid/actors-psk")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive PSK: %w", err)
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
		return nil, nil, err
	}
	opts = append(opts, cronjobs...)

	// Add the rate limiters
	rateLimiters, rateLimiterOpts, err := o.getRateLimiters()
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, rateLimiterOpts...)

	// Add the database connection
	providerOpt, err := o.getProvider()
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, providerOpt)

	// Create a new actor host
	h, err := local.NewHost(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create actor host: %w", err)
	}

	// Bind a service for each rate limiter so the middleware can invoke them
	rateLimitServices := make(map[string]*ratelimit.RateLimitService, len(rateLimiters))
	for name, rl := range rateLimiters {
		rateLimitServices[name] = rl.Service(h.Service())
	}

	return h, rateLimitServices, nil
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
	if common.EnvConfig.AppEnv == "test" {
		return opts, nil
	}

	// Register the analytics job
	analyticsJob, err := job.GetAnalyticsJob(o.AppConfig, o.HttpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics cron job: %w", err)
	}
	if analyticsJob != nil {
		// This could be nil if analytics are disabled
		opts = append(opts, local.WithBuiltInActor(analyticsJob))
	}

	// Register the file cleanup jobs
	fileCleanupJobs, err := job.GetFileCleanupJobs(o.DB, o.FileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to get file cleanup cron jobs: %w", err)
	}
	for _, j := range fileCleanupJobs {
		opts = append(opts, local.WithBuiltInActor(j))
	}

	return opts, nil
}

// getRateLimiters creates a built-in rate-limit actor for each middleware policy and returns both the created actors (keyed by policy name) and the host options to register them
// Unlike cron jobs, rate limiters keep no durable state, so they are registered in every environment
func (o *NewActorsOpts) getRateLimiters() (actors map[string]*ratelimit.RateLimit, opts []local.HostOption, err error) {
	policies := middleware.RateLimitPolicies()
	actors = make(map[string]*ratelimit.RateLimit, len(policies))
	for _, p := range policies {
		rl, err := ratelimit.New(
			p.Name,
			ratelimit.WithRate(p.Rate),
			ratelimit.WithPer(p.Per),
			ratelimit.WithBurst(p.Burst),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating rate limiter %q: %w", p.Name, err)
		}
		actors[p.Name] = rl
		opts = append(opts, local.WithBuiltInActor(rl))
	}

	return actors, opts, nil
}
