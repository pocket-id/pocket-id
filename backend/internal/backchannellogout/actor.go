package backchannellogout

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/italypaleale/francis/actor"
)

// ActorType is the actor type that delivers back-channel logout tokens as durable jobs
const ActorType = "BackchannelLogoutNotifier"

// methodDeliver is the job method that delivers one logout token to one client
const methodDeliver = "deliver"

const (
	// deliveryConcurrency caps the notifier actors active at once, so revoking access for many users still finishes in reasonable time without flooding clients
	deliveryConcurrency = 4

	// deliveryMaxAttempts caps the delivery attempts for one notification, so an unreachable client is not retried forever
	deliveryMaxAttempts = 3
)

// notifierActor delivers the logout tokens scheduled as jobs
// Jobs make the notifications durable: they survive a restart and failed deliveries are retried up to deliveryMaxAttempts times
type notifierActor struct {
	service *Service
}

func (s *Service) newNotifierActor(_ string, _ *actor.Service) actor.Actor {
	return &notifierActor{service: s}
}

// Job implements actor.ActorJob
func (a *notifierActor) Job(ctx context.Context, method string, data actor.Envelope) error {
	if method != methodDeliver {
		return fmt.Errorf("%w: unsupported method '%s'", actor.ErrJobPermanentFailure, method)
	}
	if data == nil {
		return fmt.Errorf("%w: job input is empty", actor.ErrJobPermanentFailure)
	}

	var t target
	err := data.Decode(&t)
	if err != nil {
		return fmt.Errorf("%w: job input is not a valid target: %w", actor.ErrJobPermanentFailure, err)
	}

	return a.service.sendLogoutToken(ctx, t)
}

// JobFailed implements actor.ActorJobFailed
func (a *notifierActor) JobFailed(ctx context.Context, _ string, _ string, data actor.Envelope, jobErr error) error {
	var t target
	if data != nil {
		_ = data.Decode(&t)
	}

	slog.ErrorContext(ctx, "Giving up on delivering back-channel logout token",
		slog.String("clientId", t.ClientID),
		slog.String("userId", t.UserID),
		slog.String("logoutUrl", t.LogoutURL),
		slog.Any("error", jobErr),
	)
	return nil
}
