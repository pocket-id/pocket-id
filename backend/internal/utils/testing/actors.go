//go:build unit

// This file is only imported by unit tests

package testing

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/italypaleale/francis/components/standalone"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
)

// testActorHostPSK is the runtime pre-shared key for the test actor host
// It only needs to be a stable value of at least 32 bytes, since the test host never talks to another host
const testActorHostPSK = "pocket-id-test-actor-host-psk-32bytes"

// NewActorHostForTest starts a single-host Francis cluster backed by the in-memory provider, runs it, and waits until it is ready to serve invocations
// The register callback, if not nil, runs after the host is created but before it starts, so callers can register actors with host.RegisterActor/host.RegisterBuiltInActor (must be called before the host is running)
// The host is stopped when the test ends
// The in-memory provider keeps no state on disk, so the test never touches a real database
func NewActorHostForTest(t *testing.T, register func(t *testing.T, h *local.Host)) *local.Host {
	t.Helper()

	hostOpts := []local.HostOption{
		local.WithAddress(freeLoopbackAddr(t)),
		local.WithRuntimePSKs([]byte(testActorHostPSK)),
		local.WithStandaloneMemoryProvider(standalone.StandaloneMemoryOptions{}),
		local.WithShutdownGracePeriod(time.Second),
	}

	h, err := local.NewHost(hostOpts...)
	require.NoError(t, err)

	// Register built-in actors before the host starts
	if register != nil {
		register(t, h)
	}

	// Run the host in the background and stop it when the test ends
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		<-errCh
	})

	// Wait until the host is ready before returning, so callers can invoke actors immediately
	select {
	case <-h.Ready():
	case err = <-errCh:
		t.Fatalf("actor host stopped before becoming ready: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the actor host to become ready")
	}

	return h
}

// freeLoopbackAddr reserves a free loopback port and returns its address
// The port is released before returning, so the actor host can bind it
func freeLoopbackAddr(t *testing.T) string {
	t.Helper()

	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := lis.Addr().String()
	err = lis.Close()
	require.NoError(t, err)

	return addr
}
