package cmds

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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

// The server also serves TLS over a UNIX socket, so the healthcheck must keep dialing
// the socket instead of falling back to the host in the endpoint
func TestHealthcheckUnixSocketWithTLS(t *testing.T) {
	setTLSFiles(t)

	socketPath := filepath.Join(shortTempDir(t), "s.sock")
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/healthz", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}),
		ReadHeaderTimeout: time.Second,
		TLSConfig:         newSelfSignedTLSConfig(t),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ServeTLS(listener, "", "")
	}()

	// The connection is upgraded to HTTP/2, which keeps a graceful shutdown waiting for the idle client connection
	t.Cleanup(func() {
		require.NoError(t, server.Close())
		require.ErrorIs(t, <-errCh, http.ErrServerClosed)
	})

	result, err := healthcheck(t.Context(), healthcheckFlags{
		Endpoint:   "https://localhost:1411",
		UnixSocket: socketPath,
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, result.StatusCode)
}

// The healthcheck must reach a server that serves TLS with a self-signed certificate
func TestHealthcheckTLSSelfSignedCertificate(t *testing.T) {
	setTLSFiles(t)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/healthz", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	result, err := healthcheck(t.Context(), healthcheckFlags{
		Endpoint: server.URL,
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, result.StatusCode)
}

func TestDefaultEndpoint(t *testing.T) {
	t.Run("uses http if no TLS certificate is configured", func(t *testing.T) {
		assert.Equal(t, "http://localhost:"+common.EnvConfig.Port, defaultEndpoint())
	})

	t.Run("uses https if a TLS certificate is configured", func(t *testing.T) {
		setTLSFiles(t)
		assert.Equal(t, "https://localhost:"+common.EnvConfig.Port, defaultEndpoint())
	})
}

// t.TempDir embeds the test name, which can exceed the maximum socket path length
func shortTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "pid")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

// Only the presence of the paths matters, the healthcheck never reads the files
func setTLSFiles(t *testing.T) {
	t.Helper()

	certFile, keyFile := common.EnvConfig.TLSCertFile, common.EnvConfig.TLSKeyFile
	t.Cleanup(func() {
		common.EnvConfig.TLSCertFile, common.EnvConfig.TLSKeyFile = certFile, keyFile
	})

	common.EnvConfig.TLSCertFile = "cert.pem"
	common.EnvConfig.TLSKeyFile = "key.pem"
}

func newSelfSignedTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certificateDER},
			PrivateKey:  privateKey,
		}},
		MinVersion: tls.VersionTLS13,
	}
}
