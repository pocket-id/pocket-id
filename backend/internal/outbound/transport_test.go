package outbound

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, value string) *http.Client {
	t.Helper()

	policy, err := ParsePolicy(value, nil)
	require.NoError(t, err)
	return &http.Client{Transport: NewTransport(nil, PurposeSCIM, policy)}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect from the "localhost" name to the loopback IP, which is only allowed by address
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, server.URL+"/target", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server
}

func get(t *testing.T, client *http.Client, rawURL string) error {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	return nil
}

func TestTransportBlocksDisallowedAddresses(t *testing.T) {
	server := newTestServer(t)

	// Capture the log output to check that the hint names the variables to change
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	err := get(t, newTestClient(t, ""), server.URL)

	blockedErr, ok := errors.AsType[*BlockedError](err)
	require.True(t, ok, "expected a BlockedError, got %v", err)
	assert.Equal(t, PurposeSCIM, blockedErr.Purpose)
	assert.Equal(t, "127.0.0.1", blockedErr.Addr.String())
	assert.Contains(t, err.Error(), "OUTBOUND_ALLOWED_HOSTS_SCIM")
}

func TestTransportAllowsConfiguredAddresses(t *testing.T) {
	server := newTestServer(t)

	require.NoError(t, get(t, newTestClient(t, "loopback"), server.URL))
	require.NoError(t, get(t, newTestClient(t, "127.0.0.1"), server.URL))
	require.NoError(t, get(t, newTestClient(t, "127.0.0.0/8"), server.URL))
}

func TestTransportAllowsConfiguredHostnames(t *testing.T) {
	server := newTestServer(t)
	localhostURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)

	require.NoError(t, get(t, newTestClient(t, "localhost"), localhostURL))

	// Trust is tied to the name, so the loopback IP itself stays blocked
	err := get(t, newTestClient(t, "localhost"), server.URL)
	_, ok := errors.AsType[*BlockedError](err)
	require.True(t, ok, "expected a BlockedError, got %v", err)
}

func TestTransportChecksRedirects(t *testing.T) {
	server := newTestServer(t)
	localhostURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)

	// The first hop goes to a trusted name, the redirect to an address that isn't allowed
	err := get(t, newTestClient(t, "localhost"), localhostURL+"/redirect")
	_, ok := errors.AsType[*BlockedError](err)
	require.True(t, ok, "expected a BlockedError, got %v", err)
}

func TestTransportRejectsUnsupportedSchemes(t *testing.T) {
	transport := NewTransport(nil, PurposeSCIM, Policy{})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "ftp://8.8.8.8/file", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req) //nolint:bodyclose // No response is returned on error
	require.ErrorContains(t, err, "unsupported URL scheme")
}
