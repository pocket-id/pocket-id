package ldapsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"
)

// The LdapSync singleton actor decides when the recurring LDAP synchronization runs.

// SyncActorType is the actor type for the LDAP sync actor
const SyncActorType = "LdapSync"

const (
	// alarmSync is the name of the repeating alarm that runs the synchronization
	alarmSync = "sync"

	// syncInterval is how often the synchronization runs, as the ISO8601 duration the alarm repetition expects
	// There's no jitter: the alarm is cluster-wide, so there are no replicas to spread apart
	syncInterval = "PT1H"

	// Delay the initial sync by 5s
	initialSyncDelay = 5 * time.Second

	// alarmTimeout bounds the alarm operations performed by the actor
	alarmTimeout = 10 * time.Second
)

// syncActor is the cluster-wide singleton that triggers the recurring LDAP synchronization
type syncActor struct {
	log       *slog.Logger
	service   *Service
	appConfig AppConfigResolver
	// scheduleDisabled removes the alarm instead of arming it, for environments that drive syncs explicitly
	scheduleDisabled bool
	client           actor.Client[struct{}]
}

// NewSyncActor returns the factory that allocates the LDAP sync actor
func NewSyncActor(service *Service, appConfig AppConfigResolver, scheduleDisabled bool) actor.Factory {
	return func(actorID string, actorService *actor.Service) actor.Actor {
		return &syncActor{
			log: slog.With(
				slog.String("scope", "actor"),
				slog.String("actorType", SyncActorType),
			),
			service:          service,
			appConfig:        appConfig,
			scheduleDisabled: scheduleDisabled,
			// The actor keeps no state of its own: the client is only used to manage the alarm
			client: actor.NewActorClient[struct{}](SyncActorType, actorID, actorService),
		}
	}
}

// Bootstrap implements actor.ActorBootstrapper
// The host drives it on every startup, routed to the single owning host, so it must stay idempotent
func (a *syncActor) Bootstrap(parentCtx context.Context, _ actor.Envelope) error {
	ctx, cancel := context.WithTimeout(parentCtx, alarmTimeout)
	defer cancel()

	// The schedule may have been enabled in a previous run, so make sure a leftover alarm doesn't keep firing
	if a.scheduleDisabled {
		err := a.client.DeleteAlarm(ctx, alarmSync)
		if err != nil && !errors.Is(err, actor.ErrAlarmNotFound) {
			return fmt.Errorf("error deleting the LDAP sync alarm: %w", err)
		}

		return nil
	}

	// Setting the alarm replaces whatever is registered, which both restores an alarm that was lost and picks up a change to the interval
	// It's due right away (with a small delay) so the directory is synchronized as soon as the cluster starts, matching what the pre-actor scheduled job did
	err := a.client.SetAlarm(ctx, alarmSync, actor.AlarmProperties{
		DueTime:  time.Now().Add(initialSyncDelay),
		Interval: syncInterval,
	})
	if err != nil {
		return fmt.Errorf("error setting the LDAP sync alarm: %w", err)
	}

	a.log.DebugContext(parentCtx, "Registered the recurring LDAP sync alarm", slog.String("interval", syncInterval))

	return nil
}

// Alarm implements actor.ActorAlarm
func (a *syncActor) Alarm(ctx context.Context, name string, _ actor.Envelope) error {
	if name != alarmSync {
		return fmt.Errorf("unsupported alarm '%s' for the %s actor", name, SyncActorType)
	}

	a.sync(ctx)

	// A failed sync never surfaces as an error: the framework would retry the occurrence and then delete the alarm once the attempts run out, which would stop the synchronization altogether
	// The next occurrence comes around on its own, exactly like the pre-actor scheduled job
	return nil
}

// sync runs one synchronization, unless LDAP is disabled
// It logs failures rather than returning them, since the alarm has nowhere useful to send the error
func (a *syncActor) sync(ctx context.Context) {
	dbConfig, err := a.appConfig.GetConfig(ctx)
	if err != nil {
		a.log.ErrorContext(ctx, "Failed to load the app configuration, skipping the LDAP sync", slog.Any("error", err))
		return
	}

	if !dbConfig.LdapEnabled.IsTrue() {
		a.log.DebugContext(ctx, "LDAP is disabled, skipping the sync")
		return
	}

	a.log.InfoContext(ctx, "Starting the LDAP sync")
	start := time.Now()

	err = a.service.SyncAll(ctx, dbConfig)
	if err != nil {
		a.log.ErrorContext(ctx, "LDAP sync failed, will try again on the next run", slog.Duration("duration", time.Since(start)), slog.Any("error", err))
		return
	}

	a.log.InfoContext(ctx, "LDAP sync completed", slog.Duration("duration", time.Since(start)))
}
