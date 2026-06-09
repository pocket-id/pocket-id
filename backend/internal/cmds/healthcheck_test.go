package cmds

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckTCPSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/healthz", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	result, err := healthcheck(t.Context(), healthcheckFlags{
		Endpoint: server.URL,
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, result.StatusCode)
	require.Equal(t, server.URL+"/healthz", result.URL)
}

func TestHealthcheckFailsOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := healthcheck(t.Context(), healthcheckFlags{
		Endpoint: server.URL,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "unexpected status 500")
}

func TestHealthcheckUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "pocket-id.sock")
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/healthz", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}),
		ReadHeaderTimeout: time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, server.Shutdown(ctx))
		require.ErrorIs(t, <-errCh, http.ErrServerClosed)
	})

	result, err := healthcheck(t.Context(), healthcheckFlags{
		Endpoint:   "http://localhost:1411",
		UnixSocket: socketPath,
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, result.StatusCode)
	require.Equal(t, "http://localhost:1411/healthz", result.URL)
}
