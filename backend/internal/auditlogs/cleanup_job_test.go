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

func TestModuleRegistersAuditLogCleanupCronJob(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		_, err := New(Dependencies{
			DB:            db,
			Actors:        host,
			RetentionDays: 90,
		})
		require.NoError(t, err)
	})
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

	err = db.Create(&model.AuditLog{Base: model.Base{ID: "log-old"}, Event: model.AuditLogEventSignIn, UserID: user.ID}).Error
	require.NoError(t, err)
	err = db.Create(&model.AuditLog{Base: model.Base{ID: "log-recent"}, Event: model.AuditLogEventSignIn, UserID: user.ID}).Error
	require.NoError(t, err)

	// BeforeCreate stamps CreatedAt, so the log past the retention window is backdated directly
	oldCreatedAt := datatype.DateTime(time.Now().AddDate(0, 0, -retentionDays-1))
	err = db.Model(&model.AuditLog{}).Where("id = ?", "log-old").Update("created_at", oldCreatedAt).Error
	require.NoError(t, err)

	job := &cleanupJob{db: db, retentionDays: retentionDays}
	err = job.clearAuditLogs(t.Context())
	require.NoError(t, err)

	var remaining []string
	err = db.Model(&model.AuditLog{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"log-recent"}, remaining)
}

func TestNewCleanupJobCreatesCronActor(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	cronActor, err := newCleanupJob(db, 90)
	require.NoError(t, err)
	require.Equal(t, "cronjob.ClearAuditLogs", cronActor.ActorType())
}
