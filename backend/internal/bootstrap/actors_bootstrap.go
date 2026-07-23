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
	"github.com/italypaleale/francis/components"
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

	// Derive the cluster host limit from the HA setting
	// With HA disabled the cluster is capped at a single replica; enabling HA lifts the cap
	maxHosts := 1
	if o.EnvConfig.HAEnabled {
		maxHosts = 0
	}

	// Options for the host
	opts := []local.HostOption{
		local.WithAddress(net.JoinHostPort(o.EnvConfig.ActorsHost, o.EnvConfig.ActorsPort)),
		local.WithLogger(log.With("scope", "actor-host")),
		local.WithRuntimePSKs(psk),
		local.WithShutdownGracePeriod(10 * time.Second),
		local.WithMaxHosts(maxHosts),
		local.WithHostHealthCheckDeadline(ActorsHostHealthCheckDeadline(o.EnvConfig.HAEnabled)),
	}

	// With a single active host the relaxed alarm intervals reduce database load
	// When HA is enabled they are dropped so Francis uses its tighter defaults, which distribute alarm work and fail over faster across multiple hosts
	if !o.EnvConfig.HAEnabled {
		opts = append(opts,
			local.WithAlarmsPollInterval(5*time.Minute),
			local.WithAlarmsFetchAheadInterval(5*time.Minute),
		)
	}

	// Add the database connection
	providerOpt, err := o.getProviderOption()
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

// nonHAHostHealthCheckDeadline is the relaxed host health-check deadline used when HA is disabled
// A single active host does not need aggressive health checks, so a longer deadline reduces database load
const nonHAHostHealthCheckDeadline = 90 * time.Second

// ActorsHostHealthCheckDeadline returns the health-check deadline the actor host uses for the given HA setting
// The cluster admin used during import must pass the same value so it waits the right amount of time for hosts to drain
func ActorsHostHealthCheckDeadline(haEnabled bool) time.Duration {
	if haEnabled {
		return components.DefaultHostHealthCheckDeadline
	}
	return nonHAHostHealthCheckDeadline
}

// ActorsProviderOptions builds the Francis provider options for the given database handles
// The actor host and the cluster admin must use the same options so they address the same cluster
func ActorsProviderOptions(pg *pgxpool.Pool, sqliteDB *sql.DB) (components.ProviderOptions, error) {
	switch {
	case pg != nil && sqliteDB != nil:
		return nil, errors.New("cannot have both Postgres and SQLite connections")
	case pg != nil:
		return postgres.PostgresProviderOptions{DB: pg}, nil
	case sqliteDB != nil:
		return local.SQLiteProviderOptions{DB: sqliteDB}, nil
	default:
		return nil, errors.New("one of Postgres and SQLite must be set")
	}
}

// getProviderOption wraps the shared provider options in the host option the local host expects
func (o *NewActorsOpts) getProviderOption() (local.HostOption, error) {
	providerOpts, err := ActorsProviderOptions(o.Postgres, o.SQLite)
	if err != nil {
		return nil, err
	}
	switch v := providerOpts.(type) {
	case postgres.PostgresProviderOptions:
		return local.WithPostgresProvider(v), nil
	case local.SQLiteProviderOptions:
		return local.WithSQLiteProvider(v), nil
	default:
		return nil, fmt.Errorf("unsupported provider options type: %T", providerOpts)
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
