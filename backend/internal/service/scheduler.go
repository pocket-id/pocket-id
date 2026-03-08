package service

import (
	"context"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/go-co-op/gocron/v2"
)

// RegisterJobOpts holds optional configuration for registering a scheduled job.
type RegisterJobOpts struct {
	// RunImmediately runs the job immediately after registration.
	RunImmediately bool
	// ExtraOptions are additional gocron job options.
	ExtraOptions []gocron.JobOption
	// BackOff is an optional backoff strategy. If non-nil, the job will be wrapped
	// with automatic retry logic using the provided backoff on transient failures.
	BackOff backoff.BackOff
}

// Scheduler is an interface for registering and managing background jobs.
type Scheduler interface {
	RegisterJob(ctx context.Context, name string, def gocron.JobDefinition, job func(ctx context.Context) error, opts RegisterJobOpts) error
	RemoveJob(name string) error
}
