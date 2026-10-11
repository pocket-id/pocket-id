package usersignup

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type testUserDeleter struct {
	db      *gorm.DB
	deleted []string
}

func (d *testUserDeleter) DeleteUser(ctx context.Context, _ *appconfig.AppConfigModel, userID string, _ bool) error {
	d.deleted = append(d.deleted, userID)
	return d.db.WithContext(ctx).Delete(&model.User{}, "id = ?", userID).Error
}

func TestCleanupJobDeletesUnverifiedSignups(t *testing.T) {
	// seed creates users whose deadline lies in the past or future, covering every condition the job checks
	seed := func(t *testing.T, db *gorm.DB) {
		t.Helper()

		past := new(datatype.DateTime(time.Now().Add(-time.Minute)))
		future := new(datatype.DateTime(time.Now().Add(time.Hour)))
		users := []model.User{
			{Base: model.Base{ID: "overdue"}, Username: "overdue", EmailVerificationDeadline: past},
			{Base: model.Base{ID: "not-yet-due"}, Username: "not-yet-due", EmailVerificationDeadline: future},
			{Base: model.Base{ID: "promoted-admin"}, Username: "promoted-admin", IsAdmin: true, EmailVerificationDeadline: past},
			{Base: model.Base{ID: "verified"}, Username: "verified", EmailVerified: true, EmailVerificationDeadline: past},
			{Base: model.Base{ID: "existing"}, Username: "existing"},
		}
		require.NoError(t, db.Create(&users).Error)
	}

	newJob := func(db *gorm.DB, mode string) (*cleanupJob, *testUserDeleter) {
		cfg := appconfig.NewTestConfig(nil)
		cfg.EmailVerification = appconfig.AppConfigValue(mode)
		deleter := &testUserDeleter{db: db}
		return &cleanupJob{db: db, appConfig: appconfig.StaticResolver{Config: cfg}, users: deleter}, deleter
	}

	t.Run("deletes only overdue unverified regular users when verification is required", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		seed(t, db)
		job, deleter := newJob(db, appconfig.EmailVerificationRequired)

		require.NoError(t, job.deleteUnverifiedSignups(t.Context()))

		require.Equal(t, []string{"overdue"}, deleter.deleted)
		var remaining []string
		require.NoError(t, db.Model(&model.User{}).Order("id").Pluck("id", &remaining).Error)
		require.Equal(t, []string{"existing", "not-yet-due", "promoted-admin", "verified"}, remaining)
	})

	t.Run("clears every deadline instead of deleting when verification is not required", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		seed(t, db)
		job, deleter := newJob(db, appconfig.EmailVerificationOptional)

		require.NoError(t, job.deleteUnverifiedSignups(t.Context()))

		require.Empty(t, deleter.deleted)
		var withDeadline int64
		require.NoError(t, db.Model(&model.User{}).Where("email_verification_deadline IS NOT NULL").Count(&withDeadline).Error)
		require.Zero(t, withDeadline)
	})
}
