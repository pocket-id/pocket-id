package oidc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type recordingAlarmClient struct {
	actor.Client[struct{}]

	name       string
	properties actor.AlarmProperties
}

func (c *recordingAlarmClient) SetAlarm(_ context.Context, name string, properties actor.AlarmProperties) error {
	c.name = name
	c.properties = properties
	return nil
}

func TestMetadataClientsActorBootstrapSchedulesCleanup(t *testing.T) {
	client := &recordingAlarmClient{}
	metadataActor := &metadataClientsActor{client: client}

	require.NoError(t, metadataActor.Bootstrap(t.Context(), nil))
	require.Equal(t, metadataClientsCleanupAlarm, client.name)
	require.Equal(t, metadataClientsCleanupInterval.String(), client.properties.Interval)
	require.WithinDuration(t, time.Now(), client.properties.DueTime, time.Second)
}

func TestMetadataClientsActorCleanup(t *testing.T) {
	const (
		staleID    = "https://stale.example/cimd"
		freshID    = "https://fresh.example/cimd"
		unknownID  = "https://unknown.example/cimd"
		standardID = "standard-client"
	)

	seed := func(t *testing.T, retention time.Duration) *metadataClientsActor {
		t.Helper()

		db := testutils.NewDatabaseForTest(t)
		now := time.Now()
		staleMetadata := datatype.DateTime(now.AddDate(0, -7, 0))
		freshMetadata := datatype.DateTime(now.AddDate(0, -1, 0))
		clients := []model.OidcClient{
			{
				Base:              model.Base{ID: staleID},
				Name:              "stale",
				ClientType:        model.OidcClientTypeCIMD,
				MetadataExpiresAt: &staleMetadata,
			},
			{
				Base:              model.Base{ID: freshID},
				Name:              "fresh",
				ClientType:        model.OidcClientTypeCIMD,
				MetadataExpiresAt: &freshMetadata,
			},
			{
				Base:       model.Base{ID: unknownID},
				Name:       "unknown",
				ClientType: model.OidcClientTypeCIMD,
			},
			{
				Base:              model.Base{ID: standardID},
				Name:              "standard",
				ClientType:        model.OidcClientTypeStandard,
				MetadataExpiresAt: &staleMetadata,
			},
		}
		require.NoError(t, db.Create(&clients).Error)

		return newMetadataClientsActor(
			actor.SingletonActorID,
			nil,
			db,
			func(context.Context) (time.Duration, error) { return retention, nil },
		).(*metadataClientsActor)
	}

	remainingIDs := func(t *testing.T, metadataActor *metadataClientsActor) map[string]bool {
		t.Helper()

		var remaining []model.OidcClient
		require.NoError(t, metadataActor.db.Find(&remaining).Error)
		ids := make(map[string]bool, len(remaining))
		for _, client := range remaining {
			ids[client.ID] = true
		}
		return ids
	}

	t.Run("deletes only inactive metadata clients", func(t *testing.T) {
		metadataActor := seed(t, 180*24*time.Hour)
		require.NoError(t, metadataActor.Alarm(t.Context(), metadataClientsCleanupAlarm, nil))

		ids := remainingIDs(t, metadataActor)
		require.False(t, ids[staleID])
		require.True(t, ids[freshID])
		require.True(t, ids[unknownID])
		require.True(t, ids[standardID])
	})

	t.Run("non-positive retention disables cleanup", func(t *testing.T) {
		metadataActor := seed(t, 0)
		before := remainingIDs(t, metadataActor)
		require.NoError(t, metadataActor.Alarm(t.Context(), metadataClientsCleanupAlarm, nil))

		require.Equal(t, before, remainingIDs(t, metadataActor))
	})

	t.Run("rejects unknown alarms", func(t *testing.T) {
		metadataActor := seed(t, 180*24*time.Hour)
		require.Error(t, metadataActor.Alarm(t.Context(), "unknown", nil))
	})

	t.Run("propagates retention lookup failures", func(t *testing.T) {
		metadataActor := &metadataClientsActor{
			getClientRetention: func(context.Context) (time.Duration, error) {
				return 0, errors.New("config unavailable")
			},
		}
		require.ErrorContains(t, metadataActor.Alarm(t.Context(), metadataClientsCleanupAlarm, nil), "config unavailable")
	})
}
