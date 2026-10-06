package auditlogs

import (
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestModuleCleanupDeletesLogsPastRetention(t *testing.T) {
	const retentionDays = 7
	db := testutils.NewDatabaseForTest(t)

	// Preserve a recent record while proving that the registered job honors the configured retention window
	require.NoError(t, db.Create(&AuditLog{Base: model.Base{ID: "expired-log"}, Event: EventSignIn}).Error)
	require.NoError(t, db.Create(&AuditLog{Base: model.Base{ID: "recent-log"}, Event: EventSignIn}).Error)
	require.NoError(t, db.Model(&AuditLog{}).Where("id = ?", "expired-log").Update("created_at", datatype.DateTime(time.Now().AddDate(0, 0, -retentionDays-1))).Error)

	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		_, err := New(Dependencies{
			DB: db, Actors: host, RetentionDays: retentionDays,
		})
		require.NoError(t, err)
	})

	require.Eventually(t, func() bool {
		var remaining []string
		if db.Model(&AuditLog{}).Pluck("id", &remaining).Error != nil {
			return false
		}
		return len(remaining) == 1 && remaining[0] == "recent-log"
	}, 5*time.Second, 10*time.Millisecond)
}

func TestNewCleanupJobsPreserveAuditActorName(t *testing.T) {
	jobs, err := newCleanupJobs(testutils.NewDatabaseForTest(t), 90)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, "cronjob.ClearAuditLogs", jobs[0].ActorType())
}
