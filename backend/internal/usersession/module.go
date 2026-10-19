package usersession

import (
	"context"
	"errors"
	"fmt"

	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB              *gorm.DB
	Actors          francishost.Host
	CleanupDisabled bool
}

type Module struct {
	service *Service
}

func New(deps Dependencies) (*Module, error) {
	service := newService(deps.DB)

	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the user session cleanup job")
		}

		cleanupJob, err := newCleanupJob(service)
		if err != nil {
			return nil, err
		}
		err = deps.Actors.RegisterBuiltInActor(cleanupJob)
		if err != nil {
			return nil, fmt.Errorf("error registering user session cleanup cron actor: %w", err)
		}
	}

	return &Module{service: service}, nil
}

// Track records the session a freshly minted access token belongs to
func (m *Module) Track(ctx context.Context, tx *gorm.DB, accessToken string) error {
	return m.service.Track(ctx, tx, accessToken)
}

// Validate reports whether the session for the given token jti still exists
func (m *Module) Validate(ctx context.Context, jti string) error {
	return m.service.Validate(ctx, jti)
}

// Delete ends the session for the given token jti
func (m *Module) Delete(ctx context.Context, jti string) error {
	return m.service.Delete(ctx, jti)
}
