package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/italypaleale/francis/builtin/ratelimit"
	"github.com/italypaleale/francis/components"
	"github.com/italypaleale/francis/components/postgres"
	"github.com/italypaleale/francis/components/sqlite"
	francishost "github.com/italypaleale/francis/host"
	"github.com/italypaleale/francis/host/local"
	"github.com/italypaleale/francis/host/remote"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

// ErrRemoteFrancisRuntime is returned by the helpers that reach the actor data through Pocket ID's own database when FRANCIS_HOST points to a standalone Francis runtime
// That runtime owns the actor data instead, so it can only be reached through the runtime itself
var ErrRemoteFrancisRuntime = errors.New("the actor data is owned by the standalone Francis runtime configured in FRANCIS_HOST, and is not stored in Pocket ID's database")

type NewActorsOpts struct {
	Postgres *pgxpool.Pool

	EnvConfig   *common.EnvConfigSchema
	InstanceID  string
	HttpClient  *http.Client
	DB          *gorm.DB
	FileStorage storage.FileStorage
}

func NewActors(o NewActorsOpts) (francishost.Host, map[string]*ratelimit.RateLimitService, error) {
	log := slog.Default().With("scope", "actor-host")

	// Create the actor host for the configured topology
	// The embedded runtime keeps the actor data in Pocket ID's own database, while a standalone Francis runtime owns it instead and coordinates every host that connects to it
	var (
		h   francishost.Host
		err error
	)
	if o.EnvConfig.HasEmbeddedFrancisRuntime() {
		log.Debug("Starting the embedded Francis runtime")
		h, err = o.newEmbeddedHost(log)
	} else {
		log.Info("Connecting to a standalone Francis runtime", slog.Any("addresses", o.EnvConfig.FrancisAddresses))
		h, err = o.newRemoteHost(log)
	}
	if err != nil {
		return nil, nil, err
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

// newEmbeddedHost creates the actor host that runs the Francis runtime inside the Pocket ID process, backed by Pocket ID's own database
func (o *NewActorsOpts) newEmbeddedHost(log *slog.Logger) (*local.Host, error) {
	// Derive a PSK from the global encryption key
	// The runtime PSK derives the cluster CA used for host-to-host mTLS
	psk, err := o.getPSK()
	if err != nil {
		return nil, fmt.Errorf("failed to derive PSK: %w", err)
	}

	// Derive the cluster host limit from the HA setting
	// With HA disabled the cluster is capped at a single replica
	maxHosts := 1
	if o.EnvConfig.HAEnabled {
		// 0 = no cap
		maxHosts = 0
	}

	// Options for the host
	opts := []local.HostOption{
		local.WithAddress(net.JoinHostPort(o.EnvConfig.ActorsHost, o.EnvConfig.ActorsPort)),
		local.WithLogger(log),
		local.WithRuntimePSKs(psk),
		local.WithShutdownGracePeriod(10 * time.Second),
		local.WithMaxHosts(maxHosts),
		local.WithHostHealthCheckDeadline(ActorsHostHealthCheckDeadline(o.EnvConfig.HAEnabled)),
	}

	// With a single active host the relaxed alarm intervals reduce database load
	// The longer lease duration also means fewer lease renewals, since Francis renews a lease 10s before it expires (no other host can claim the alarm anyways)
	// When HA is enabled these are dropped so Francis uses its tighter defaults, which distribute alarm work and fail over faster across multiple hosts
	if !o.EnvConfig.HAEnabled {
		opts = append(opts,
			local.WithAlarmsPollInterval(5*time.Minute),
			local.WithAlarmsFetchAheadInterval(5*time.Minute),
			local.WithAlarmsLeaseDuration(180*time.Second),
		)
	}

	// Add the database connection
	providerOpt, err := o.getProviderOption()
	if err != nil {
		return nil, err
	}
	opts = append(opts, providerOpt)

	h, err := local.NewHost(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor host: %w", err)
	}

	return h, nil
}

// newRemoteHost creates the actor host that connects to a standalone Francis runtime
// The runtime owns the actor state, placement, and alarms, so none of the embedded runtime's database and clustering options apply here
// That includes the cap on the number of hosts in the cluster, which the runtime enforces through its own "maxHosts" setting: Pocket ID cannot limit itself to a single replica from this side
func (o *NewActorsOpts) newRemoteHost(log *slog.Logger) (*remote.Host, error) {
	opts := []remote.HostOption{
		// Actors placed on this host are invoked by its peers at this address, which is also the one it advertises to the runtime
		remote.WithAddress(net.JoinHostPort(o.EnvConfig.ActorsHost, o.EnvConfig.ActorsPort)),
		remote.WithLogger(log),
		remote.WithRuntimeAddresses(o.EnvConfig.FrancisAddresses...),
		remote.WithHostBootstrapPSK(o.EnvConfig.FrancisHostPSK),
		remote.WithShutdownGracePeriod(10 * time.Second),
	}

	// Pinning the cluster CA lets Pocket ID verify the runtime on its very first connection
	// Francis requires the trust decision to be explicit, so without a pinned CA we have to opt into trusting the certificate served on first use, which it warns about
	if len(o.EnvConfig.FrancisCA) > 0 {
		opts = append(opts, remote.WithPinnedCA(o.EnvConfig.FrancisCA))
	} else {
		opts = append(opts, remote.WithUnsafeNoPinnedCA())
	}

	h, err := remote.NewHost(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote actor host: %w", err)
	}

	return h, nil
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
// It only works with the embedded runtime, since the actor state then lives in Pocket ID's own database: with a standalone Francis runtime it returns ErrRemoteFrancisRuntime.
func NewActorStateStore(o NewActorsOpts) (*local.Host, error) {
	if !o.EnvConfig.HasEmbeddedFrancisRuntime() {
		return nil, ErrRemoteFrancisRuntime
	}

	providerOpt, err := o.getProviderOption()
	if err != nil {
		return nil, err
	}

	psk, err := o.getPSK()
	if err != nil {
		return nil, fmt.Errorf("failed to derive PSK: %w", err)
	}

	return local.NewHost(
		// The address is required by the host but never bound, since the host is not Run
		local.WithAddress("127.0.0.1:1"),
		local.WithLogger(slog.Default().With("scope", "actor-state-store")),
		// The health-check deadline only needs to exceed the provider's query timeout to pass validation
		local.WithHostHealthCheckDeadline(90*time.Second),
		local.WithRuntimePSKs(psk),
		providerOpt,
	)
}

// ActorsHostHealthCheckDeadline returns the health-check deadline the actor host uses for the given HA setting
// This is exported because the import method needs it too
func ActorsHostHealthCheckDeadline(haEnabled bool) time.Duration {
	if haEnabled {
		return components.DefaultHostHealthCheckDeadline
	}

	// A single active host does not need aggressive health checks, so a longer deadline reduces database load
	return 90 * time.Second
}

// ActorsProviderOptions builds the Francis provider options for the given database
// The actor host, the cluster admin, and the backup provider must all use these so they address the same cluster
// A Postgres deployment passes both handles, since the Gorm one wraps the same pool, and the pool is what the provider takes
func ActorsProviderOptions(db *gorm.DB, pg *pgxpool.Pool) (components.ProviderOptions, error) {
	// Log each provider operation, such as a lease renewal or an actor lookup, while debugging
	// The statements those operations run are logged separately, by the instrumentation attached to the connection in ConnectDatabase
	operationLog := components.OperationLogConfig{
		Enabled: common.EnvConfig.LogLevel == "debug",
	}

	switch {
	case pg != nil:
		return postgres.PostgresProviderOptions{
			DB:           pg,
			OperationLog: operationLog,
		}, nil
	case db != nil:
		// The SQLite provider takes the raw connection, which only Gorm holds
		sqliteDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get *sql.DB connection from Gorm: %w", err)
		}
		return local.SQLiteProviderOptions{
			DB:           sqliteDB,
			OperationLog: operationLog,
		}, nil
	default:
		return nil, errors.New("one of the Postgres pool and the database connection must be set")
	}
}

// NewActorsBackupProvider creates a Francis provider that talks to the same cluster as the actor host, without registering a host or joining the cluster
// It's meant for backing up and restoring the actor host's own data
// The caller owns the returned provider and must Close it: the database connection stays owned by the caller and is not closed.
func NewActorsBackupProvider(ctx context.Context, providerOpts components.ProviderOptions) (components.ActorProvider, error) {
	// The health check deadline must match the actor host's, since it decides when a host that stopped health-checking is considered gone, and a restore refuses to run while any host is still connected
	// The remaining values are irrelevant here, because this provider never registers a host nor processes alarms
	cfg := components.NewProviderConfig()
	cfg.HostHealthCheckDeadline = ActorsHostHealthCheckDeadline(common.EnvConfig.HAEnabled)

	log := slog.Default().With("scope", "actors-backup")

	var (
		provider components.ActorProvider
		err      error
	)
	switch v := providerOpts.(type) {
	case postgres.PostgresProviderOptions:
		provider, err = postgres.NewPostgresProvider(log, v, cfg)
	case sqlite.SQLiteProviderOptions:
		provider, err = sqlite.NewSQLiteProvider(log, v, cfg)
	default:
		err = fmt.Errorf("unsupported provider options type: %T", providerOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create actor provider: %w", err)
	}

	// Init applies the provider's schema migrations, so this also works against a database the actor host has never run against
	err = provider.Init(ctx)
	if err != nil {
		_ = provider.Close()
		return nil, fmt.Errorf("failed to initialize actor provider: %w", err)
	}

	return provider, nil
}

// getProviderOption wraps the shared provider options in the host option the local host expects
func (o *NewActorsOpts) getProviderOption() (local.HostOption, error) {
	providerOpts, err := ActorsProviderOptions(o.DB, o.Postgres)
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

func (o *NewActorsOpts) registerCronJobs(host francishost.Host) (err error) {
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
func (o *NewActorsOpts) registerRateLimiters(host francishost.Host) (actors map[string]*ratelimit.RateLimit, err error) {
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
