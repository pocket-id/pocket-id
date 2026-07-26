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
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

type NewActorsOpts struct {
	SQLite   *sql.DB
	Postgres *pgxpool.Pool

	EnvConfig   *common.EnvConfigSchema
	InstanceID  string
	HttpClient  *http.Client
	DB          *gorm.DB
	FileStorage storage.FileStorage
}

func NewActors(o NewActorsOpts) (*local.Host, map[string]*ratelimit.RateLimitService, error) {
	log := slog.Default()

	// Derive a PSK from the global encryption key
	// The runtime PSK derives the cluster CA used for host-to-host mTLS
	psk, err := o.getPSK()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive PSK: %w", err)
	}

	// Options for the host
	opts := []local.HostOption{
		local.WithAddress(net.JoinHostPort(o.EnvConfig.ActorsHost, o.EnvConfig.ActorsPort)),
		local.WithLogger(log.With("scope", "actor-host")),
		local.WithRuntimePSKs(psk),
		local.WithShutdownGracePeriod(10 * time.Second),
		// TODO: Tweak these values once Pocket ID fully supports horizontal scaling.
		// The relaxed intervals are appropriate for a single active host, but should be
		// tuned for lower latency and better distribution across a multi-host cluster.
		local.WithHostHealthCheckDeadline(90 * time.Second),
		local.WithAlarmsPollInterval(5 * time.Minute),
		local.WithAlarmsFetchAheadInterval(5 * time.Minute),
	}

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

	// Add all cron jobs
	err = o.registerCronJobs(h)
	if err != nil {
		return nil, nil, err
	}

	// Add the rate limiters
	rateLimiters, err := o.registerRateLimiters(h)
	if err != nil {
		return nil, nil, err
	}

	// Bind a service for each rate limiter so the middleware can invoke them
	rateLimitServices := make(map[string]*ratelimit.RateLimitService, len(rateLimiters))
	for name, rl := range rateLimiters {
		rateLimitServices[name] = rl.Service(h.Service())
	}

	return h, rateLimitServices, nil
}

// Derive a PSK from the global encryption key
func (o *NewActorsOpts) getPSK() ([]byte, error) {
	// This is tied to the instance ID of the Pocket ID deployment/cluster
	// Note: changing the key derivation or the seed is a breaking change
	return crypto.DeriveKey(o.EnvConfig.EncryptionKey, "pocketid/actors-psk/"+o.InstanceID)
}

// NewActorStateStore creates a minimal actor host that can read and write actor state directly, without joining the cluster or binding a network port.
// It's meant for short-lived contexts such as CLI commands that need to persist actor state (for example, one-time access tokens) without running the full actor host.
// The returned host must NOT be Run(): only direct state operations (Get/Set/Delete on state) are supported, and they require the actor state tables to already exist, which is the case whenever the server has run at least once against this database.
func NewActorStateStore(db *gorm.DB, pg *pgxpool.Pool) (*local.Host, error) {
	opts := &NewActorsOpts{DB: db, Postgres: pg}
	if pg == nil {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get *sql.DB connection from Gorm: %w", err)
		}
		opts.SQLite = sqlDB
	}

	providerOpt, err := opts.getProvider()
	if err != nil {
		return nil, err
	}

	return local.NewHost(
		// The address is required by the host but never bound, since the host is not Run
		local.WithAddress("127.0.0.1:1"),
		local.WithLogger(slog.Default().With("scope", "actor-state-store")),
		// The health-check deadline only needs to exceed the provider's query timeout to pass validation
		local.WithHostHealthCheckDeadline(90*time.Second),
		providerOpt,
	)
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

func (o *NewActorsOpts) registerCronJobs(host *local.Host) (err error) {
	// In test mode, we do not register anything
	if common.EnvConfig.AppEnv == "test" {
		return nil
	}

	// Register the analytics job
	analyticsJob, err := job.GetAnalyticsJob(o.HttpClient, o.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get analytics cron job: %w", err)
	}

	// This could be nil if analytics are disabled
	if analyticsJob != nil {
		err = host.RegisterBuiltInActor(analyticsJob)
		if err != nil {
			return fmt.Errorf("error registering built-in actor for analytics job: %w", err)
		}
	}

	// Register the file cleanup jobs
	fileCleanupJobs, err := job.GetFileCleanupJobs(o.DB, o.FileStorage)
	if err != nil {
		return fmt.Errorf("failed to get file cleanup cron jobs: %w", err)
	}
	for _, j := range fileCleanupJobs {
		err = host.RegisterBuiltInActor(j)
		if err != nil {
			return fmt.Errorf("error registering built-in actor for cleanup job: %w", err)
		}
	}

	return nil
}

// registerRateLimiters creates a built-in rate-limit actor for each middleware policy and returns both the created actors (keyed by policy name) and the host options to register them
// Unlike cron jobs, rate limiters keep no durable state, so they are registered in every environment
func (o *NewActorsOpts) registerRateLimiters(host *local.Host) (actors map[string]*ratelimit.RateLimit, err error) {
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
			return nil, fmt.Errorf("error creating rate limiter %q: %w", p.Name, err)
		}
		actors[p.Name] = rl

		err = host.RegisterBuiltInActor(rl)
		if err != nil {
			return nil, fmt.Errorf("error registering built-in actor for rate limiter '%s': %w", p.Name, err)
		}
	}

	return actors, nil
}
