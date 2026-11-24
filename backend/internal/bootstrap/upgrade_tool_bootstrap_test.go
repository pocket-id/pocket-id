package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
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

func TestUpgradeLegacyDatabaseWaitsForAnotherReplica(t *testing.T) {
	db := testutils.NewPostgresDatabaseForTest(t)
	sqlDb, err := db.DB()
	require.NoError(t, err)
	previousProvider := common.EnvConfig.DbProvider
	common.EnvConfig.DbProvider = common.DbProviderPostgres
	t.Cleanup(func() {
		common.EnvConfig.DbProvider = previousProvider
	})

	// Another replica holds the lock while it upgrades a legacy database
	_, err = sqlDb.ExecContext(t.Context(), `CREATE TABLE schema_migrations (version bigint)`)
	require.NoError(t, err)
	other, err := lock.NewPostgresTableLocker(lock.WithTableLockID(upgradeLockID))
	require.NoError(t, err)
	require.NoError(t, other.Lock(t.Context(), sqlDb))

	done := make(chan error, 1)
	go func() {
		done <- upgradeLegacyDatabase(t.Context(), sqlDb)
	}()
	select {
	case err = <-done:
		t.Fatalf("returned while another replica held the upgrade lock: %v", err)
	case <-time.After(time.Second):
	}

	// Once the other replica finished the upgrade, this one must find nothing left to do instead of running the tool again
	_, err = sqlDb.ExecContext(t.Context(), `DROP TABLE schema_migrations`)
	require.NoError(t, err)
	require.NoError(t, other.Unlock(t.Context(), sqlDb))
	require.NoError(t, <-done)
}
