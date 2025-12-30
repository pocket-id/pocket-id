package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// ScimSchedulerService schedules and triggers periodic synchronization
// of SCIM service providers. Each provider is tracked independently,
// and sync operations are run at or after their scheduled time.
type ScimSchedulerService struct {
	scimService      *ScimService
	providerSyncTime map[string]time.Time
	mu               sync.RWMutex
}

func NewScimSchedulerService(ctx context.Context, scimService *ScimService) (*ScimSchedulerService, error) {
	s := &ScimSchedulerService{
		scimService:      scimService,
		providerSyncTime: make(map[string]time.Time),
	}

	err := s.start(ctx)
	return s, err
}

// start initializes the scheduler and begins the synchronization loop.
// Syncs happen every hour by default, but ScheduleSync can be called to schedule a sync sooner.
func (s *ScimSchedulerService) start(ctx context.Context) error {
	providers, err := s.scimService.ListServiceProviders(ctx)
	if err != nil {
		return err
	}

	inAHour := time.Now().Add(time.Hour)
	for _, provider := range providers {
		s.providerSyncTime[provider.ID] = inAHour
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
				// Runs every 5 seconds to check if any provider is due for sync
			case <-ticker.C:
				s.mu.RLock()
				for providerID, syncTime := range s.providerSyncTime {
					if syncTime.Before(time.Now()) {
						err = s.scimService.SyncServiceProvider(ctx, providerID)
						if err != nil {
							slog.Error("Error syncing SCIM client", slog.String("provider_id", providerID), slog.Any("error", err))
							return
						}
						// A successful sync schedules the next sync in an hour
						s.setSyncTime(providerID, time.Hour)
					}
				}
				s.mu.RUnlock()
			}
		}
	}()
	return nil
}

// ScheduleSync forces the given provider to be synced soon by
// moving its next scheduled time to 5 minutes from now.
func (s *ScimSchedulerService) ScheduleSync(providerID string) {
	s.setSyncTime(providerID, 5*time.Minute)
}

func (s *ScimSchedulerService) setSyncTime(providerID string, t time.Duration) {
	s.mu.Lock()
	s.providerSyncTime[providerID] = time.Now().Add(t)
	s.mu.Unlock()
}
