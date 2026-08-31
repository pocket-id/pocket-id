// Package auditlogs owns the background maintenance of the audit log table.
package auditlogs

import (
	"errors"
	"fmt"

	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	// RetentionDays is how long audit logs are kept before the cleanup job deletes them
	RetentionDays int

	// CleanupDisabled skips registering the cleanup cron job, for example in tests
	CleanupDisabled bool
}

type Module struct{}

func New(deps Dependencies) (*Module, error) {
	// Register the cleanup job for audit logs past the retention window
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the audit log cleanup cron job")
		}

		cleanupJob, err := newCleanupJob(deps.DB, deps.RetentionDays)
		if err != nil {
			return nil, err
		}

		err = deps.Actors.RegisterBuiltInActor(cleanupJob)
		if err != nil {
			return nil, fmt.Errorf("error registering audit log cleanup cron actor: %w", err)
		}
	}

	return &Module{}, nil
}
