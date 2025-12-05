package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Source:
// https://github.com/ItalyPaleAle/traefik-forward-auth/blob/v3.5.1/pkg/utils/servicerunner.go
// Copyright (c) 2018, Thom Seddon & Contributors Copyright (c) 2023, Alessandro Segala & Contributors
// License: MIT (https://github.com/ItalyPaleAle/traefik-forward-auth/blob/v3.5.1/LICENSE.md)

func TestServiceRunner_Run(t *testing.T) {
	t.Run("successful services", func(t *testing.T) {
		// Create a service that just returns no error after 0.2s
		successService := func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}

		// Create a service runner with two success services
		runner := NewServiceRunner(successService, successService)

		// Run the services with a timeout to avoid hanging if something goes wrong
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		// Run should return nil when all services succeed
		err := runner.Run(ctx)
		require.NoError(t, err)
	})

	t.Run("service with error", func(t *testing.T) {
		// Create a service that returns an error
		expectedErr := errors.New("service failed")
		errorService := func(ctx context.Context) error {
			return expectedErr
		}

		// Create a service runner with one error service and one success service
		successService := func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}

		runner := NewServiceRunner(errorService, successService)

		// Run the services with a timeout
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		// Run should return the error
		err := runner.Run(ctx)
		require.Error(t, err)

		// The error should contain our expected error
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("service error cancels others", func(t *testing.T) {
		expectedErr := errors.New("boom")
		errorService := func(ctx context.Context) error {
			return expectedErr
		}
		waitingService := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		runner := NewServiceRunner(errorService, waitingService)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		err := runner.Run(ctx)
		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("context canceled", func(t *testing.T) {
		// Create a service that waits until context is canceled
		waitingService := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		// Create another service that returns no error quickly
		quickService := func(ctx context.Context) error {
			return nil
		}

		runner := NewServiceRunner(waitingService, quickService)

		// Create a context that we can cancel
		ctx, cancel := context.WithCancel(t.Context())

		// Run the runner in a goroutine
		errCh := make(chan error)
		go func() {
			errCh <- runner.Run(ctx)
		}()

		// Cancel the context to trigger service shutdown
		cancel()

		// Wait for the runner to finish with a timeout
		select {
		case err := <-errCh:
			require.NoError(t, err, "expected nil error (context.Canceled should be ignored)")
		case <-time.After(5 * time.Second):
			t.Fatal("test timed out waiting for runner to finish")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		// Create two services that return different errors
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		service1 := func(ctx context.Context) error {
			return err1
		}
		service2 := func(ctx context.Context) error {
			return err2
		}

		runner := NewServiceRunner(service1, service2)

		// Run the services
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		// Run should join all errors
		err := runner.Run(ctx)
		require.Error(t, err)

		// Check that both errors are included
		require.ErrorIs(t, err, err1)
		require.ErrorIs(t, err, err2)
	})
}
