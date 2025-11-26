package utils

import (
	"context"
	"errors"
)

// Source:
// https://github.com/ItalyPaleAle/traefik-forward-auth/blob/v3.5.1/pkg/utils/servicerunner.go
// Copyright (c) 2018, Thom Seddon & Contributors Copyright (c) 2023, Alessandro Segala & Contributors
// License: MIT (https://github.com/ItalyPaleAle/traefik-forward-auth/blob/v3.5.1/LICENSE.md)

// Service is a background service
type Service func(ctx context.Context) error

// ServiceRunner oversees a number of services running in background
type ServiceRunner struct {
	services []Service
}

// NewServiceRunner creates a new ServiceRunner
func NewServiceRunner(services ...Service) *ServiceRunner {
	return &ServiceRunner{
		services: services,
	}
}

// Run all background services
func (r *ServiceRunner) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)
	for _, service := range r.services {
		go func(service Service) {
			// Run the service
			rErr := service(ctx)

			// Ignore context canceled errors here as they generally indicate that the service is stopping
			if rErr != nil && !errors.Is(rErr, context.Canceled) {
				cancel()
				errCh <- rErr
				return
			}
			errCh <- nil
		}(service)
	}

	// Wait for all services to return
	errs := make([]error, 0)
	for range len(r.services) {
		err := <-errCh
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
