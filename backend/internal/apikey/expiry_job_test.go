package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type expiryJobTestConfig struct {
	config *appconfig.AppConfigModel
}

func (c expiryJobTestConfig) GetConfig(context.Context) (*appconfig.AppConfigModel, error) {
	return c.config, nil
}

type expiryJobTestSender struct {
	keyNames []string
}

func (s *expiryJobTestSender) SendAPIKeyExpiringSoon(_ context.Context, _ *appconfig.AppConfigModel, _, _, _, apiKeyName string, _ time.Time) error {
	s.keyNames = append(s.keyNames, apiKeyName)
	return nil
}

func TestModuleRegistersAPIKeyExpiryCronJob(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	config := expiryJobTestConfig{config: &appconfig.AppConfigModel{}}
	sender := &expiryJobTestSender{}

	testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		t.Helper()
		_, err := New(t.Context(), Dependencies{
			DB:          db,
			Actors:      host,
			AppConfig:   config,
			EmailSender: sender,
		})
		require.NoError(t, err)
	})
}

func TestAPIKeyExpiryCronJobNotifiesAndMarksExpiringKeys(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	userEmail := "expiry-job@example.com"
	user := model.User{
		Username:    "expiry-job-user",
		Email:       &userEmail,
		FirstName:   "Expiry",
		LastName:    "Job",
		DisplayName: "Expiry Job",
	}
	require.NoError(t, db.Create(&user).Error)

	now := time.Now()
	expiringKey := ApiKey{
		Name:      "Expiring",
		Key:       "expiring-hash",
		ExpiresAt: datatype.DateTime(now.Add(3 * 24 * time.Hour)),
		UserID:    user.ID,
	}
	laterKey := ApiKey{
		Name:      "Later",
		Key:       "later-hash",
		ExpiresAt: datatype.DateTime(now.Add(30 * 24 * time.Hour)),
		UserID:    user.ID,
	}
	require.NoError(t, db.Create(&expiringKey).Error)
	require.NoError(t, db.Create(&laterKey).Error)

	service, err := newService(t.Context(), db, "")
	require.NoError(t, err)
	config := expiryJobTestConfig{config: &appconfig.AppConfigModel{
		EmailApiKeyExpirationEnabled: "true",
	}}
	sender := &expiryJobTestSender{}

	cronActor, err := newExpiryJob(service, config, sender)
	require.NoError(t, err)
	require.Equal(t, "cronjob.ExpiredApiKeyEmailJob", cronActor.ActorType())

	job := &expiryJob{
		service:   service,
		appConfig: config,
		email:     sender,
	}
	require.NoError(t, job.checkAndNotifyExpiringAPIKeys(t.Context()))
	require.Equal(t, []string{"Expiring"}, sender.keyNames)

	require.NoError(t, db.First(&expiringKey, "id = ?", expiringKey.ID).Error)
	require.True(t, expiringKey.ExpirationEmailSent)
	require.NoError(t, db.First(&laterKey, "id = ?", laterKey.ID).Error)
	require.False(t, laterKey.ExpirationEmailSent)
}
