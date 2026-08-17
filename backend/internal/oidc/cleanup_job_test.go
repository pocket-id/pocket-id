package oidc

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
		"cronjob.ClearOAuth2Sessions",
		"cronjob.ClearOAuth2JTIs",
		"cronjob.ClearInteractionSessions",
	}, actorTypes)

	// Every job must be registrable on an actor host
	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		for _, j := range jobs {
			err := host.RegisterBuiltInActor(j)
			require.NoError(t, err)
		}
	})
}

func TestOIDCCleanupJobsDeleteExpiredRows(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	err := db.Create(&model.OidcClient{Base: model.Base{ID: "cleanup-job-client"}, Name: "Cleanup Job Client"}).Error
	require.NoError(t, err)

	var (
		past   = datatype.DateTime(time.Now().Add(-time.Hour))
		future = datatype.DateTime(time.Now().Add(time.Hour))
	)

	err = db.Create(&OAuth2Session{
		Base: model.Base{ID: "session-expired"}, Kind: "access_token", Key: "k-expired", RequestID: "r1",
		ClientID: "cleanup-job-client", Active: true, RequestData: `{"client_id":"cleanup-job-client"}`, ExpiresAt: &past,
	}).Error
	require.NoError(t, err)
	err = db.Create(&OAuth2Session{
		Base: model.Base{ID: "session-active"}, Kind: "access_token", Key: "k-active", RequestID: "r2",
		ClientID: "cleanup-job-client", Active: true, RequestData: `{"client_id":"cleanup-job-client"}`, ExpiresAt: &future,
	}).Error
	require.NoError(t, err)

	err = db.Create(&clientAssertionJTI{Base: model.Base{ID: "jti-expired"}, JTI: "expired", ExpiresAt: past}).Error
	require.NoError(t, err)
	err = db.Create(&clientAssertionJTI{Base: model.Base{ID: "jti-active"}, JTI: "active", ExpiresAt: future}).Error
	require.NoError(t, err)

	err = db.Create(&InteractionSession{
		Base: model.Base{ID: "interaction-abandoned"}, Scopes: datatype.StringList{"openid"},
		ClientID: "cleanup-job-client", RequestedAt: datatype.DateTime(time.Now()), Parameters: map[string]string{},
	}).Error
	require.NoError(t, err)
	err = db.Create(&InteractionSession{
		Base: model.Base{ID: "interaction-pending"}, Scopes: datatype.StringList{"openid"},
		ClientID: "cleanup-job-client", RequestedAt: datatype.DateTime(time.Now()), Parameters: map[string]string{},
	}).Error
	require.NoError(t, err)
	// BeforeCreate stamps CreatedAt, so the abandoned session is backdated past its lifetime directly
	abandonedCreatedAt := datatype.DateTime(time.Now().Add(-interactionSessionLifetime - time.Minute))
	err = db.Model(&InteractionSession{}).Where("id = ?", "interaction-abandoned").Update("created_at", abandonedCreatedAt).Error
	require.NoError(t, err)

	jobs := &cleanupJobs{db: db}
	err = jobs.clearOAuth2Sessions(t.Context())
	require.NoError(t, err)
	err = jobs.clearOAuth2JTIs(t.Context())
	require.NoError(t, err)
	err = jobs.clearInteractionSessions(t.Context())
	require.NoError(t, err)

	var remaining []string
	err = db.Model(&OAuth2Session{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"session-active"}, remaining)

	err = db.Model(&clientAssertionJTI{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"jti-active"}, remaining)

	err = db.Model(&InteractionSession{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.Equal(t, []string{"interaction-pending"}, remaining)
}
