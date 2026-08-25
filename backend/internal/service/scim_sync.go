package service

import (
	"context"
)

// ScimSyncScheduler schedules a cluster-wide SCIM synchronization after application data changes
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}
