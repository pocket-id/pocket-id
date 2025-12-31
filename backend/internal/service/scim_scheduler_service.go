package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
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

// ScheduleSync forces the given provider to be synced soon by
// moving its next scheduled time to 5 minutes from now.
func (s *ScimSchedulerService) ScheduleSync(providerID string) {
	s.setSyncTime(providerID, 5*time.Minute)
}

// start initializes the scheduler and begins the synchronization loop.
// Syncs happen every hour by default, but ScheduleSync can be called to schedule a sync sooner.
func (s *ScimSchedulerService) start(ctx context.Context) error {
	if err := s.refreshProviders(ctx); err != nil {
		return err
	}

	go func() {
		const (
			syncCheckInterval    = 5 * time.Second
			providerRefreshDelay = time.Minute
		)

		ticker := time.NewTicker(syncCheckInterval)
		defer ticker.Stop()
		lastProviderRefresh := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
				// Runs every 5 seconds to check if any provider is due for sync
			case <-ticker.C:
				now := time.Now()
				if now.Sub(lastProviderRefresh) >= providerRefreshDelay {
					err := s.refreshProviders(ctx)
					if err != nil {
						slog.Error("Error refreshing SCIM service providers",
							slog.Any("error", err),
						)
					} else {
						lastProviderRefresh = now
					}
				}

				var due []string
				s.mu.RLock()
				for providerID, syncTime := range s.providerSyncTime {
					if !syncTime.After(now) {
						due = append(due, providerID)
					}
				}
				s.mu.RUnlock()

				s.syncProviders(ctx, due)

			}
		}
	}()

	return nil
}

func (s *ScimSchedulerService) refreshProviders(ctx context.Context) error {
	providers, err := s.scimService.ListServiceProviders(ctx)
	if err != nil {
		return err
	}

	inAHour := time.Now().Add(time.Hour)

	s.mu.Lock()
	for _, provider := range providers {
		if _, exists := s.providerSyncTime[provider.ID]; !exists {
			s.providerSyncTime[provider.ID] = inAHour
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *ScimSchedulerService) syncProviders(ctx context.Context, providerIDs []string) {
	for _, providerID := range providerIDs {
		err := s.scimService.SyncServiceProvider(ctx, providerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Remove the provider from the schedule if it no longer exists
				s.mu.Lock()
				delete(s.providerSyncTime, providerID)
				s.mu.Unlock()
			} else {
				slog.Error("Error syncing SCIM client",
					slog.String("provider_id", providerID),
					slog.Any("error", err),
				)
			}
			continue
		}
		// A successful sync schedules the next sync in an hour
		s.setSyncTime(providerID, time.Hour)
	}
}

func (s *ScimSchedulerService) setSyncTime(providerID string, t time.Duration) {
	s.mu.Lock()
	s.providerSyncTime[providerID] = time.Now().Add(t)
	s.mu.Unlock()
}
