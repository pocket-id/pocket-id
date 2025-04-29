package bootstrap

import (
	"context"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/signals"
)

func Bootstrap() error {
	// Get a context that is canceled when the application is stopping
	ctx := signals.SignalContext(context.Background())

	initApplicationImages()

	// Perform migrations for changes
	migrateConfigDBConnstring()
	migrateKey()

	// Connect to the database
	db := newDatabase()

	// Create all services
	svc, err := initServices(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Init the job scheduler
	scheduler, err := initScheduler(ctx, db, svc)
	if err != nil {
		return fmt.Errorf("failed to initialize scheduler: %w", err)
	}
	svc.backgroundServices = append(svc.backgroundServices, scheduler)

	// Init the router
	router := initRouter(db, svc)
	svc.backgroundServices = append(svc.backgroundServices, router)

	// Run all background serivces
	// This call blocks until the context is canceled
	err = utils.
		NewServiceRunner(svc.backgroundServices...).
		Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run services: %w", err)
	}

	return nil
}
