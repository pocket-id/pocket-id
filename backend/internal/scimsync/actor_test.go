package scimsync

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeSyncer struct {
	calls atomic.Int32
	err   error
}

func (s *fakeSyncer) SyncAll(_ context.Context) error {
	s.calls.Add(1)
	return s.err
}

func TestActorBootstrapArmsRecurringAlarm(t *testing.T) {
	host, act := newSyncActorForTest(t, &fakeSyncer{}, false)

	require.NoError(t, act.Bootstrap(t.Context(), nil))

	properties, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmRecurringSync)
	require.NoError(t, err)
	assert.Equal(t, recurringSyncInterval, properties.Interval)
	assert.WithinDuration(t, time.Now().Add(initialSyncDelay), properties.DueTime, time.Second)
}

func TestActorBootstrapIsIdempotent(t *testing.T) {
	host, act := newSyncActorForTest(t, &fakeSyncer{}, false)

	require.NoError(t, act.Bootstrap(t.Context(), nil))
	require.NoError(t, act.Bootstrap(t.Context(), nil))

	properties, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmRecurringSync)
	require.NoError(t, err)
	assert.Equal(t, recurringSyncInterval, properties.Interval)
}

func TestActorBootstrapRemovesAutomaticAlarmsWhenDisabled(t *testing.T) {
	host, act := newSyncActorForTest(t, &fakeSyncer{}, true)

	for _, name := range []string{alarmRecurringSync, alarmScheduledSync} {
		require.NoError(t, host.SetAlarm(t.Context(), ActorType, actor.SingletonActorID, name, actor.AlarmProperties{
			DueTime: time.Now().Add(time.Hour),
		}))
	}

	require.NoError(t, act.Bootstrap(t.Context(), nil))

	for _, name := range []string{alarmRecurringSync, alarmScheduledSync} {
		_, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, name)
		require.ErrorIs(t, err, actor.ErrAlarmNotFound)
	}
}

func TestActorSchedulesDebouncedSync(t *testing.T) {
	host, act := newSyncActorForTest(t, &fakeSyncer{}, false)

	_, err := act.Invoke(t.Context(), methodScheduleSync, nil)
	require.NoError(t, err)

	first, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmScheduledSync)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(scheduledSyncDelay), first.DueTime, time.Second)
	assert.Empty(t, first.Interval)

	_, err = act.Invoke(t.Context(), methodScheduleSync, nil)
	require.NoError(t, err)

	second, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmScheduledSync)
	require.NoError(t, err)
	assert.False(t, second.DueTime.Before(first.DueTime))
}

func TestActorDoesNotScheduleDebouncedSyncWhenDisabled(t *testing.T) {
	host, act := newSyncActorForTest(t, &fakeSyncer{}, true)

	_, err := act.Invoke(t.Context(), methodScheduleSync, nil)
	require.NoError(t, err)

	_, err = host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmScheduledSync)
	require.ErrorIs(t, err, actor.ErrAlarmNotFound)
}

func TestActorAlarmsRunSync(t *testing.T) {
	syncer := &fakeSyncer{}
	_, act := newSyncActorForTest(t, syncer, false)

	require.NoError(t, act.Alarm(t.Context(), alarmRecurringSync, nil))
	require.NoError(t, act.Alarm(t.Context(), alarmScheduledSync, nil))
	assert.EqualValues(t, 2, syncer.calls.Load())
}

func TestActorAlarmSwallowsSyncFailures(t *testing.T) {
	syncer := &fakeSyncer{err: errors.New("provider unavailable")}
	_, act := newSyncActorForTest(t, syncer, false)

	require.NoError(t, act.Alarm(t.Context(), alarmRecurringSync, nil))
	assert.EqualValues(t, 1, syncer.calls.Load())
}

func TestActorRejectsUnknownOperations(t *testing.T) {
	_, act := newSyncActorForTest(t, &fakeSyncer{}, false)

	_, err := act.Invoke(t.Context(), "unknown", nil)
	require.Error(t, err)

	err = act.Alarm(t.Context(), "unknown", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported alarm")
}

func TestRegisteredSingletonBootstrapsAndFires(t *testing.T) {
	syncer := &fakeSyncer{}
	host := testutils.NewActorHostForTest(t,
		func(t *testing.T, host *local.Host) {
			err := host.RegisterSingletonActor(ActorType, newActor(syncer, false))
			require.NoError(t, err)
		},
		local.WithAlarmsPollInterval(5*time.Minute),
		local.WithAlarmsFetchAheadInterval(5*time.Minute),
	)

	// The host bootstraps singleton actors asynchronously after it becomes ready
	require.Eventually(t, func() bool {
		_, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmRecurringSync)
		return err == nil
	}, 10*time.Second, 20*time.Millisecond)

	// The first occurrence is delivered without waiting for the normal five-minute alarm poll
	require.Eventually(t, func() bool {
		return syncer.calls.Load() > 0
	}, initialSyncDelay+30*time.Second, 50*time.Millisecond)
}

func TestModuleRegistersSingletonAndSchedulesClusterWideSync(t *testing.T) {
	var module *Module
	host := testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		var err error
		module, err = New(Dependencies{
			DB:     testutils.NewDatabaseForTest(t),
			Actors: host,
		})
		require.NoError(t, err)
	})

	// The host bootstraps singleton actors asynchronously after it becomes ready
	require.Eventually(t, func() bool {
		_, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmRecurringSync)
		return err == nil
	}, 10*time.Second, 20*time.Millisecond)

	module.ScheduleSync(t.Context())

	properties, err := host.GetAlarm(t.Context(), ActorType, actor.SingletonActorID, alarmScheduledSync)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(scheduledSyncDelay), properties.DueTime, time.Second)
}

// newSyncActorForTest starts a test actor host and allocates the actor without registering it
func newSyncActorForTest(t *testing.T, syncer syncer, scheduleDisabled bool) (*local.Host, *syncActor) {
	t.Helper()

	host := testutils.NewActorHostForTest(t, nil)
	act, ok := newActor(syncer, scheduleDisabled)(actor.SingletonActorID, host.Service()).(*syncActor)
	require.True(t, ok)

	return host, act
}
