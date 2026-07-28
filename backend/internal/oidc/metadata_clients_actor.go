package oidc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// MetadataClientsActorType is the actor type for cluster-wide metadata client maintenance
const MetadataClientsActorType = "OIDCMetadataClients"

const (
	metadataClientsCleanupAlarm    = "cleanupInactive"
	metadataClientsCleanupInterval = 24 * time.Hour
)

// metadataClientsActor coordinates collection-wide maintenance while the database remains the source of truth for clients
type metadataClientsActor struct {
	log                *slog.Logger
	client             actor.Client[struct{}]
	db                 *gorm.DB
	getClientRetention func(context.Context) (time.Duration, error)
}

// newMetadataClientsActor allocates the singleton actor that owns metadata client maintenance
func newMetadataClientsActor(actorID string, service *actor.Service, db *gorm.DB, getClientRetention func(context.Context) (time.Duration, error)) actor.Actor {
	return &metadataClientsActor{
		log: slog.With(
			slog.String("scope", "actor"),
			slog.String("actorType", MetadataClientsActorType),
			slog.String("actorID", actorID),
		),
		client:             actor.NewActorClient[struct{}](MetadataClientsActorType, actorID, service),
		db:                 db,
		getClientRetention: getClientRetention,
	}
}

// registerMetadataClientsActor registers the singleton before the actor host starts
func registerMetadataClientsActor(actors *local.Host, db *gorm.DB, getClientRetention func(context.Context) (time.Duration, error)) error {
	err := actors.RegisterSingletonActor(
		MetadataClientsActorType,
		func(actorID string, service *actor.Service) actor.Actor {
			return newMetadataClientsActor(actorID, service, db, getClientRetention)
		},
		local.WithMaxAttempts(5),
		local.WithInitialRetryDelay(time.Second),
	)
	if err != nil {
		return fmt.Errorf("error registering the %s actor: %w", MetadataClientsActorType, err)
	}

	return nil
}

// Bootstrap installs the durable repeating cleanup alarm and safely replaces an existing schedule
func (a *metadataClientsActor) Bootstrap(parentCtx context.Context, _ actor.Envelope) error {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	err := a.client.SetAlarm(ctx, metadataClientsCleanupAlarm, actor.AlarmProperties{
		DueTime:  time.Now(),
		Interval: metadataClientsCleanupInterval.String(),
	})
	if err != nil {
		return fmt.Errorf("error scheduling inactive metadata client cleanup: %w", err)
	}

	return nil
}

// Alarm handles durable maintenance callbacks for the metadata client collection
func (a *metadataClientsActor) Alarm(ctx context.Context, name string, _ actor.Envelope) error {
	if name != metadataClientsCleanupAlarm {
		return fmt.Errorf("unsupported alarm %q", name)
	}

	retention, err := a.getClientRetention(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve dynamic client retention: %w", err)
	}
	if retention <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-retention)
	st := a.db.
		WithContext(ctx).
		Where("client_type = ?", model.OidcClientTypeCIMD).
		Where("metadata_expires_at IS NOT NULL AND metadata_expires_at < ?", datatype.DateTime(cutoff)).
		Delete(&model.OidcClient{})
	if st.Error != nil {
		return fmt.Errorf("failed to delete inactive metadata clients: %w", st.Error)
	}

	a.log.InfoContext(ctx, "Deleted inactive metadata clients", slog.Int64("count", st.RowsAffected))

	return nil
}
