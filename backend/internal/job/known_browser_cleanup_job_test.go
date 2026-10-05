package job

import (
	"context"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestKnownBrowserCleanupJobRegistersCronActor(t *testing.T) {
	job, err := GetKnownBrowserCleanupJob(testutils.NewDatabaseForTest(t))
	require.NoError(t, err)
	require.Equal(t, "cronjob.ClearKnownBrowsers", job.ActorType())
	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		require.NoError(t, host.RegisterBuiltInActor(job))
	})
}

func TestKnownBrowserCleanupDeletesExpiredRecordsAcrossUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.True(t, db.Migrator().HasIndex(&model.KnownBrowser{}, "idx_known_browsers_expires_at"))
	now := time.Now()
	for _, userID := range []string{"active-user", "inactive-user"} {
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: userID}).Error)
		for _, record := range []model.KnownBrowser{
			{UserID: userID, TokenHash: "expired", ExpiresAt: now.Add(-time.Hour).Unix()},
			{UserID: userID, TokenHash: "expires-now", ExpiresAt: now.Unix()},
			{UserID: userID, TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
		} {
			require.NoError(t, db.Create(&record).Error)
		}
	}

	job := &knownBrowserCleanupJob{db: db}
	// Repeated delivery must be harmless and preserve the same unexpired records
	for range 2 {
		require.NoError(t, job.clearKnownBrowsers(t.Context()))
		var remaining []model.KnownBrowser
		require.NoError(t, db.Order("user_id").Find(&remaining).Error)
		require.Equal(t, []model.KnownBrowser{
			{UserID: "active-user", TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
			{UserID: "inactive-user", TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
		}, remaining)
	}
}

func TestKnownBrowserCleanupPropagatesErrors(t *testing.T) {
	job := &knownBrowserCleanupJob{db: testutils.NewDatabaseForTest(t)}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, job.clearKnownBrowsers(ctx), context.Canceled)
}
