package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	francishost "github.com/italypaleale/francis/host"
	"github.com/italypaleale/francis/host/remote"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// actorClientConnectTimeout bounds how long a CLI command waits to join the cluster
// Francis reconnects to the runtime indefinitely, which is right for the server but would leave a command hanging against an unreachable runtime
const actorClientConnectTimeout = 30 * time.Second

// WithActorClient connects to the standalone Francis runtime, calls fn once the connection is live, and disconnects before returning.
// It's meant for CLI commands, which have no actor host of their own: the client joins the cluster only for the duration of fn, and hosts no actor while connected, so the runtime never places an actor on it.
// It requires FRANCIS_HOST to point to a standalone runtime, and returns ErrEmbeddedFrancisRuntime otherwise, since an embedded runtime is reached through the database instead.
func WithActorClient(parentCtx context.Context, envConfig *common.EnvConfigSchema, fn func(ctx context.Context, client francishost.Host) error) error {
	if envConfig.HasEmbeddedFrancisRuntime() {
		return ErrEmbeddedFrancisRuntime
	}

	log := slog.Default().With("scope", "actor-client")

	// The client hosts no actor, so it advertises no address of its own and binds nothing
	// The short grace period keeps a command from lingering on the way out, since there are no actors to drain
	client, err := remote.NewHost(append(
		remoteConnectionOptions(envConfig, log),
		remote.WithClientOnly(),
		remote.WithShutdownGracePeriod(2*time.Second),
	)...)
	if err != nil {
		return fmt.Errorf("failed to create the actor client: %w", err)
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- client.Run(ctx)
	}()

	// Every operation travels on the runtime session, so nothing can run before the client has joined the cluster
	connectCtx, connectCancel := context.WithTimeout(ctx, actorClientConnectTimeout)
	defer connectCancel()

	select {
	case <-client.Ready():
	case runErr := <-runErrCh:
		return fmt.Errorf("failed to connect to the Francis runtime: %w", runErr)
	case <-connectCtx.Done():
		cancel()
		<-runErrCh
		return fmt.Errorf("timed out connecting to the Francis runtime after %v", actorClientConnectTimeout)
	}

	fnErr := fn(ctx, client)

	// Leave the cluster before returning, so the runtime drops the registration instead of waiting for the health check to lapse
	cancel()
	runErr := <-runErrCh

	// The error from fn is the one the caller asked for, and a canceled run is just the disconnect we asked for
	switch {
	case fnErr != nil:
		return fnErr
	case runErr != nil && !errors.Is(runErr, context.Canceled):
		return fmt.Errorf("error disconnecting from the Francis runtime: %w", runErr)
	default:
		return nil
	}
}
