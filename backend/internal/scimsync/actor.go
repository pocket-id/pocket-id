package scimsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// The ScimSync singleton actor decides when the recurring and debounced SCIM synchronizations run

// ActorType is the actor type for the SCIM sync actor
const ActorType = "ScimSync"

const (
	// alarmRecurringSync runs the cluster-wide hourly synchronization
	alarmRecurringSync = "recurring-sync"
	// alarmScheduledSync runs the cluster-wide synchronization requested after a local change
	alarmScheduledSync = "scheduled-sync"

	// methodScheduleSync moves the debounced synchronization five minutes past the latest change
	methodScheduleSync = "schedule-sync"

	// recurringSyncInterval is how often the full synchronization runs, as the ISO8601 duration the alarm repetition expects
	recurringSyncInterval = "PT1H"
	// initialSyncDelay gives the application a moment to finish starting before the first synchronization
	initialSyncDelay = 5 * time.Second
	// scheduledSyncDelay debounces changes so several related writes produce one synchronization
	scheduledSyncDelay = 5 * time.Minute
	// alarmTimeout bounds the alarm operations performed by the actor
	alarmTimeout = 10 * time.Second
)

type syncer interface {
	SyncAll(ctx context.Context) error
}

// syncActor is the cluster-wide singleton that triggers SCIM synchronization
type syncActor struct {
	log              *slog.Logger
	syncer           syncer
	scheduleDisabled bool
	client           actor.Client[struct{}]
}

// NewActor returns the factory that allocates the SCIM sync actor
func NewActor(service *Service, scheduleDisabled bool) actor.Factory {
	return newActor(service, scheduleDisabled)
}

func newActor(syncer syncer, scheduleDisabled bool) actor.Factory {
	return func(actorID string, actorService *actor.Service) actor.Actor {
		return &syncActor{
			log: slog.With(
				slog.String("scope", "actor"),
				slog.String("actorType", ActorType),
			),
			syncer:           syncer,
			scheduleDisabled: scheduleDisabled,
			client:           actor.NewActorClient[struct{}](ActorType, actorID, actorService),
		}
	}
}

// Bootstrap implements actor.ActorBootstrapper
// The host drives it on every startup, routed to the single owning host, so it must stay idempotent
func (a *syncActor) Bootstrap(parentCtx context.Context, _ actor.Envelope) error {
	ctx, cancel := context.WithTimeout(parentCtx, alarmTimeout)
	defer cancel()

	// The test environment drives synchronization explicitly and must not inherit alarms from a previous run
	if a.scheduleDisabled {
		for _, name := range []string{alarmRecurringSync, alarmScheduledSync} {
			err := a.client.DeleteAlarm(ctx, name)
			if err != nil && !errors.Is(err, actor.ErrAlarmNotFound) {
				return fmt.Errorf("error deleting the SCIM sync alarm %q: %w", name, err)
			}
		}

		return nil
	}

	// Replacing the alarm restores a missing schedule and applies interval changes after an upgrade
	err := a.client.SetAlarm(ctx, alarmRecurringSync, actor.AlarmProperties{
		DueTime:  time.Now().Add(initialSyncDelay),
		Interval: recurringSyncInterval,
	})
	if err != nil {
		return fmt.Errorf("error setting the recurring SCIM sync alarm: %w", err)
	}

	a.log.DebugContext(parentCtx, "Registered the recurring SCIM sync alarm", slog.String("interval", recurringSyncInterval))

	return nil
}

// Invoke implements actor.ActorInvoke
func (a *syncActor) Invoke(ctx context.Context, method string, _ actor.Envelope) (any, error) {
	if method != methodScheduleSync {
		return nil, common.ErrUnsupportedActorMethod{Method: method}
	}

	if a.scheduleDisabled {
		return nil, nil
	}

	// Setting the same alarm replaces its due time, which debounces changes across every replica
	err := a.client.SetAlarm(ctx, alarmScheduledSync, actor.AlarmProperties{
		DueTime: time.Now().Add(scheduledSyncDelay),
	})
	if err != nil {
		return nil, fmt.Errorf("error setting the scheduled SCIM sync alarm: %w", err)
	}

	return nil, nil
}

// Alarm implements actor.ActorAlarm
func (a *syncActor) Alarm(ctx context.Context, name string, _ actor.Envelope) error {
	if name != alarmRecurringSync && name != alarmScheduledSync {
		return fmt.Errorf("unsupported alarm '%s' for the %s actor", name, ActorType)
	}

	a.sync(ctx)

	// A failed recurring sync must not surface as an error because exhausted alarm retries would remove the recurring schedule
	// The sync method shows a log in case of failure
	return nil
}

// sync runs one full SCIM synchronization and records its outcome
func (a *syncActor) sync(ctx context.Context) {
	a.log.InfoContext(ctx, "Starting the SCIM sync")
	start := time.Now()

	err := a.syncer.SyncAll(ctx)
	if err != nil {
		a.log.ErrorContext(ctx, "SCIM sync failed, will try again on the next run", slog.Duration("duration", time.Since(start)), slog.Any("error", err))
		return
	}

	a.log.InfoContext(ctx, "SCIM sync completed", slog.Duration("duration", time.Since(start)))
}
