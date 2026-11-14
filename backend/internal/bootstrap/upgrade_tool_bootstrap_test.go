package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadVerified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("upgrade tool"))
	}))
	t.Cleanup(server.Close)
	sum := sha256.Sum256([]byte("upgrade tool"))

	content, err := downloadVerified(t.Context(), server.URL, hex.EncodeToString(sum[:]))
	require.NoError(t, err)
	require.Equal(t, "upgrade tool", string(content))

	// A download that isn't the pinned binary must never reach the caller, which would run it
	sum[0] ^= 1
	content, err = downloadVerified(t.Context(), server.URL, hex.EncodeToString(sum[:]))
	require.ErrorContains(t, err, "was not run")
	require.Nil(t, content)
}
