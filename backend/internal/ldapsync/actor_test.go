package ldapsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// fakeAppConfigResolver returns a fixed application configuration
type fakeAppConfigResolver struct {
	config *appconfig.AppConfigModel
	err    error
}

func (f fakeAppConfigResolver) GetConfig(_ context.Context) (*appconfig.AppConfigModel, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.config, nil
}

func TestSyncActorBootstrapArmsRecurringAlarm(t *testing.T) {
	host, act := newSyncActorForTest(t, nil, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, false)

	require.NoError(t, act.Bootstrap(t.Context(), nil))

	props, err := host.GetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync)
	require.NoError(t, err)
	assert.Equal(t, syncInterval, props.Interval)
	// The first occurrence is due right away, so a restart re-syncs the directory
	assert.WithinDuration(t, time.Now(), props.DueTime, time.Minute)
}

func TestSyncActorBootstrapIsIdempotent(t *testing.T) {
	host, act := newSyncActorForTest(t, nil, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, false)

	// Every host bootstraps the singleton, so repeating it must leave a single alarm behind rather than failing
	require.NoError(t, act.Bootstrap(t.Context(), nil))
	require.NoError(t, act.Bootstrap(t.Context(), nil))

	props, err := host.GetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync)
	require.NoError(t, err)
	assert.Equal(t, syncInterval, props.Interval)
}

func TestSyncActorBootstrapRemovesAlarmWhenScheduleDisabled(t *testing.T) {
	host, act := newSyncActorForTest(t, nil, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, true)

	// Simulate an alarm left behind by a run where the schedule was still enabled
	require.NoError(t, host.SetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync, actor.AlarmProperties{
		DueTime:  time.Now(),
		Interval: syncInterval,
	}))

	require.NoError(t, act.Bootstrap(t.Context(), nil))

	_, err := host.GetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync)
	require.ErrorIs(t, err, actor.ErrAlarmNotFound)
}

func TestSyncActorBootstrapWithScheduleDisabledAndNoAlarm(t *testing.T) {
	host, act := newSyncActorForTest(t, nil, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, true)

	// There's nothing to remove, which must not be reported as a failure
	require.NoError(t, act.Bootstrap(t.Context(), nil))

	_, err := host.GetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync)
	require.ErrorIs(t, err, actor.ErrAlarmNotFound)
}

func TestSyncActorAlarmRunsSync(t *testing.T) {
	appCfg := defaultTestLDAPAppConfig()
	service, db := newTestLdapServiceWithAppConfig(t, appCfg, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-alice"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alice"},
				"sn":          {"Jones"},
				"displayName": {""},
			}),
		),
		ldapSearchResult(),
	))

	_, act := newSyncActorForTest(t, service, fakeAppConfigResolver{config: appCfg}, false)

	require.NoError(t, act.Alarm(t.Context(), alarmSync, nil))

	var alice model.User
	require.NoError(t, db.First(&alice, "ldap_id = ?", "u-alice").Error)
	assert.Equal(t, "alice", alice.Username)
}

func TestSyncActorAlarmSkipsSyncWhenLdapIsDisabled(t *testing.T) {
	appCfg := defaultTestLDAPAppConfig()
	service, db := newTestLdapServiceWithAppConfig(t, appCfg, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID": {"u-alice"},
				"uid":       {"alice"},
				"mail":      {"alice@example.com"},
			}),
		),
		ldapSearchResult(),
	))

	disabledCfg := defaultTestLDAPAppConfig()
	disabledCfg.LdapEnabled = "false"
	_, act := newSyncActorForTest(t, service, fakeAppConfigResolver{config: disabledCfg}, false)

	require.NoError(t, act.Alarm(t.Context(), alarmSync, nil))

	var count int64
	require.NoError(t, db.Model(&model.User{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestSyncActorAlarmSwallowsSyncFailures(t *testing.T) {
	// A failed sync must not surface as an error, or the framework would eventually delete the alarm and stop synchronizing altogether
	service, _ := newTestLdapService(t, newFakeLDAPClient(ldapSearchResult(), ldapSearchResult()))
	service.clientFactory = func(_ *appconfig.AppConfigModel) (ldapClient, error) {
		return nil, errors.New("connection refused")
	}

	_, act := newSyncActorForTest(t, service, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, false)

	require.NoError(t, act.Alarm(t.Context(), alarmSync, nil))
}

func TestSyncActorAlarmSwallowsAppConfigFailures(t *testing.T) {
	service, _ := newTestLdapService(t, newFakeLDAPClient(ldapSearchResult(), ldapSearchResult()))

	_, act := newSyncActorForTest(t, service, fakeAppConfigResolver{err: errors.New("config unavailable")}, false)

	require.NoError(t, act.Alarm(t.Context(), alarmSync, nil))
}

func TestSyncActorAlarmRejectsUnknownAlarm(t *testing.T) {
	_, act := newSyncActorForTest(t, nil, fakeAppConfigResolver{config: defaultTestLDAPAppConfig()}, false)

	err := act.Alarm(t.Context(), "unknown", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported alarm")
}

func TestSyncActorRegisteredSingletonBootstrapsAndFires(t *testing.T) {
	// This exercises the wiring the unit tests above bypass: the host bootstraps the singleton on its own, and the alarm it arms is delivered back to the actor
	appCfg := defaultTestLDAPAppConfig()
	service, db := newTestLdapServiceWithAppConfig(t, appCfg, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-alice"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alice"},
				"sn":          {"Jones"},
				"displayName": {""},
			}),
		),
		ldapSearchResult(),
	))

	host := testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		err := h.RegisterSingletonActor(SyncActorType, NewSyncActor(service, fakeAppConfigResolver{config: appCfg}, false))
		require.NoError(t, err)
	})

	// The host bootstraps singletons in the background once it's ready, so wait for the alarm to show up
	require.Eventually(t, func() bool {
		_, err := host.GetAlarm(t.Context(), SyncActorType, actor.SingletonActorID, alarmSync)
		return err == nil
	}, 10*time.Second, 20*time.Millisecond, "the sync alarm was never armed")

	// The alarm is due immediately, so the first occurrence syncs the directory without waiting for the interval
	require.Eventually(t, func() bool {
		var count int64
		require.NoError(t, db.Model(&model.User{}).Where("ldap_id = ?", "u-alice").Count(&count).Error)
		return count == 1
	}, 20*time.Second, 50*time.Millisecond, "the sync alarm never ran")
}

// newSyncActorForTest starts a test actor host and allocates the sync actor against it
// The actor is not registered on the host, so the host never bootstraps or fires it on its own and the test drives it explicitly
func newSyncActorForTest(t *testing.T, service *Service, appConfig AppConfigResolver, scheduleDisabled bool) (*local.Host, *syncActor) {
	t.Helper()

	host := testutils.NewActorHostForTest(t, nil)

	act, ok := NewSyncActor(service, appConfig, scheduleDisabled)(actor.SingletonActorID, host.Service()).(*syncActor)
	require.True(t, ok)

	return host, act
}
