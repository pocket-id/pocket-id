package auditlogs

import (
	"context"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestModuleRegistersBothCleanupCronJobs(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	// Seed both tables so immediate cleanup proves that the module registered both jobs
	require.NoError(t, db.Create(&AuditLog{Base: model.Base{ID: "expired-log"}, Event: EventSignIn}).Error)
	require.NoError(t, db.Model(&AuditLog{}).Where("id = ?", "expired-log").Update("created_at", datatype.DateTime(time.Now().AddDate(0, 0, -100))).Error)
	require.NoError(t, db.Create(&knownBrowser{UserID: "inactive-user", TokenHash: "expired", ExpiresAt: time.Now().Add(-time.Hour).Unix()}).Error)

	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		_, err := New(Dependencies{
			DB:            db,
			Actors:        host,
			RetentionDays: 90,
		})
		require.NoError(t, err)
	})

	require.Eventually(t, func() bool {
		var logs, browsers int64
		if db.Model(&AuditLog{}).Count(&logs).Error != nil || db.Model(&knownBrowser{}).Count(&browsers).Error != nil {
			return false
		}
		return logs == 0 && browsers == 0
	}, 5*time.Second, 10*time.Millisecond)
}

func TestModuleRequiresActorHostForCleanupJob(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	_, err := New(Dependencies{DB: db, RetentionDays: 90})
	require.ErrorContains(t, err, "actor host is required")

	// With the cleanup disabled there is nothing to register, so the actor host is not needed
	_, err = New(Dependencies{DB: db, RetentionDays: 90, CleanupDisabled: true})
	require.NoError(t, err)
}

func TestAuditLogCleanupJobDeletesLogsPastRetention(t *testing.T) {
	const retentionDays = 90

	db := testutils.NewDatabaseForTest(t)
	user := model.User{
		Base:        model.Base{ID: "cleanup-job-user"},
		Username:    "cleanup-job-user",
		FirstName:   "Cleanup",
		LastName:    "Job",
		DisplayName: "Cleanup Job",
	}
	err := db.Create(&user).Error
	require.NoError(t, err)

	err = db.Create(&AuditLog{Base: model.Base{ID: "log-old"}, Event: EventSignIn, UserID: user.ID}).Error
	require.NoError(t, err)
	err = db.Create(&AuditLog{Base: model.Base{ID: "log-recent"}, Event: EventSignIn, UserID: user.ID}).Error
	require.NoError(t, err)

	// BeforeCreate stamps CreatedAt, so the log past the retention window is backdated directly
	oldCreatedAt := datatype.DateTime(time.Now().AddDate(0, 0, -retentionDays-1))
	err = db.Model(&AuditLog{}).Where("id = ?", "log-old").Update("created_at", oldCreatedAt).Error
	require.NoError(t, err)

	job := &cleanupJobs{db: db, retentionDays: retentionDays}
	err = job.clearAuditLogs(t.Context())
	require.NoError(t, err)

	var remaining []string
	err = db.Model(&AuditLog{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"log-recent"}, remaining)
}

func TestNewCleanupJobsPreserveActorNames(t *testing.T) {
	jobs, err := newCleanupJobs(testutils.NewDatabaseForTest(t), 90)
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	require.Equal(t, "cronjob.ClearAuditLogs", jobs[0].ActorType())
	require.Equal(t, "cronjob.ClearKnownBrowsers", jobs[1].ActorType())
}

func TestKnownBrowserCleanupDeletesExpiredRecordsAcrossUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.True(t, db.Migrator().HasIndex(&knownBrowser{}, "idx_known_browsers_expires_at"))
	now := time.Now()
	for _, userID := range []string{"active-user", "inactive-user"} {
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: userID}).Error)
		for _, record := range []knownBrowser{
			{UserID: userID, TokenHash: "expired", ExpiresAt: now.Add(-time.Hour).Unix()},
			{UserID: userID, TokenHash: "expires-now", ExpiresAt: now.Unix()},
			{UserID: userID, TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
		} {
			require.NoError(t, db.Create(&record).Error)
		}
	}

	job := &cleanupJobs{db: db}
	// Repeated delivery must be harmless and preserve the same unexpired records
	for range 2 {
		require.NoError(t, job.clearKnownBrowsers(t.Context()))
		var remaining []knownBrowser
		require.NoError(t, db.Order("user_id").Find(&remaining).Error)
		require.Equal(t, []knownBrowser{
			{UserID: "active-user", TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
			{UserID: "inactive-user", TokenHash: "valid", ExpiresAt: now.Add(time.Hour).Unix()},
		}, remaining)
	}
}

func TestKnownBrowserCleanupPropagatesErrors(t *testing.T) {
	job := &cleanupJobs{db: testutils.NewDatabaseForTest(t)}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, job.clearKnownBrowsers(ctx), context.Canceled)
}
