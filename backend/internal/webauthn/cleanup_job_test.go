package webauthn

import (
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestNewCleanupJobsCreatesOneCronActorPerTable(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	jobs, err := newCleanupJobs(db)
	require.NoError(t, err)

	actorTypes := make([]string, len(jobs))
	for i, j := range jobs {
		actorTypes[i] = j.ActorType()
	}
	require.Equal(t, []string{
		"cronjob.ClearWebauthnSessions",
		"cronjob.ClearReauthenticationTokens",
	}, actorTypes)

	// Every job must be registrable on an actor host
	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		for _, j := range jobs {
			rErr := host.RegisterBuiltInActor(j)
			require.NoError(t, rErr)
		}
	})
}

func TestWebauthnCleanupJobsDeleteExpiredRows(t *testing.T) {
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

	var (
		past   = datatype.DateTime(time.Now().Add(-time.Hour))
		future = datatype.DateTime(time.Now().Add(time.Hour))
	)

	err = db.Create(&WebauthnSession{Base: model.Base{ID: "session-expired"}, Challenge: "c-expired", ExpiresAt: past}).Error
	require.NoError(t, err)
	err = db.Create(&WebauthnSession{Base: model.Base{ID: "session-active"}, Challenge: "c-active", ExpiresAt: future}).Error
	require.NoError(t, err)

	err = db.Create(&ReauthenticationToken{Base: model.Base{ID: "token-expired"}, Token: "t-expired", ExpiresAt: past, UserID: user.ID}).Error
	require.NoError(t, err)
	err = db.Create(&ReauthenticationToken{Base: model.Base{ID: "token-active"}, Token: "t-active", ExpiresAt: future, UserID: user.ID}).Error
	require.NoError(t, err)

	jobs := &cleanupJobs{db: db}
	err = jobs.clearWebauthnSessions(t.Context())
	require.NoError(t, err)
	err = jobs.clearReauthenticationTokens(t.Context())
	require.NoError(t, err)

	var remaining []string
	err = db.Model(&WebauthnSession{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"session-active"}, remaining)

	err = db.Model(&ReauthenticationToken{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"token-active"}, remaining)
}
