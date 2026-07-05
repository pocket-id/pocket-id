package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
)

type Scheduler struct {
	scheduler gocron.Scheduler
}

func NewScheduler() (*Scheduler, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create a new scheduler: %w", err)
	}

	return &Scheduler{
		scheduler: scheduler,
	}, nil
}

func (s *Scheduler) RemoveJob(name string) error {
	jobs := s.scheduler.Jobs()

	var errs []error
	for _, job := range jobs {
		if job.Name() == name {
			err := s.scheduler.RemoveJob(job.ID())
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to dequeue job %q with ID %q: %w", name, job.ID().String(), err))
			}
		}
	}

	return errors.Join(errs...)
}

// Run the scheduler.
// This function blocks until the context is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	slog.Info("Starting job scheduler")
	s.scheduler.Start()

	// Block until context is canceled
	<-ctx.Done()

	err := s.scheduler.Shutdown()
	if err != nil {
		slog.Error("Error shutting down job scheduler", slog.Any("error", err))
	} else {
		slog.Info("Job scheduler shut down")
	}

	return nil
}

func (s *Scheduler) RegisterJob(ctx context.Context, name string, def gocron.JobDefinition, jobFn func(ctx context.Context) error, opts service.RegisterJobOpts) error {
	// Wrap the job in a handler that adds tracing and logging
	jobFn = jobWithObservability(name, jobFn)

	// If a BackOff strategy is provided, wrap the job with retry logic
	if opts.BackOff != nil {
		jobFn = jobWithBackOff(jobFn, opts.BackOff)
	}

	jobOptions := []gocron.JobOption{
		gocron.WithContext(ctx),
		gocron.WithName(name),
	}

	if opts.RunImmediately {
		jobOptions = append(jobOptions, gocron.JobOption(gocron.WithStartImmediately()))
	}

	jobOptions = append(jobOptions, opts.ExtraOptions...)

	_, err := s.scheduler.NewJob(def, gocron.NewTask(jobFn), jobOptions...)
	if err != nil {
		return fmt.Errorf("failed to register job %q: %w", name, err)
	}

	return nil
}

type (
	jobNameKey struct{}
	jobIDKey   struct{}
	jobFn      = func(ctx context.Context) error
)

func jobWithObservability(jobName string, job jobFn) jobFn {
	return func(ctx context.Context) error {
		// Generate a random job ID
		jobIDObj, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("failed to generate job ID: %w", err)
		}
		jobID := jobIDObj.String()

		// Save in the context
		ctx = context.WithValue(ctx, jobNameKey{}, jobName)
		ctx = context.WithValue(ctx, jobIDKey{}, jobID)

		// Create a new context with the span
		ctx, span := tracing.Start(ctx, "pocketid.job."+jobName,
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				tracing.JobID(jobID),
			),
		)
		defer tracing.End(span, err)

		// Log the start
		logger := slog.With(
			slog.String("name", jobName),
			slog.String("jobID", jobID),
		)
		start := time.Now()
		logger.InfoContext(ctx, "Starting job")

		// Run the job
		err = job(ctx)
		d := time.Since(start)
		if err != nil {
			logger.ErrorContext(ctx, "Job failed", slog.Any("error", err), slog.Duration("duration", d))
			return err
		}

		logger.InfoContext(ctx, "Job run successfully", slog.Duration("duration", d))
		return nil
	}
}

func jobWithBackOff(job jobFn, bo backoff.BackOff) jobFn {
	return func(ctx context.Context) error {
		jobName, _ := (ctx.Value(jobNameKey{})).(string)
		jobID, _ := (ctx.Value(jobIDKey{})).(string)

		_, err := backoff.Retry(
			ctx,
			func() (struct{}, error) {
				return struct{}{}, job(ctx)
			},
			backoff.WithBackOff(bo),
			backoff.WithNotify(func(err error, d time.Duration) {
				slog.WarnContext(ctx, "Job failed, retrying",
					slog.String("name", jobName),
					slog.String("jobID", jobID),
					slog.Any("error", err),
					slog.Duration("retryIn", d),
				)
			}),
		)
		return err
	}
}

func jobDefWithJitter(interval time.Duration) gocron.JobDefinition {
	const jitter = 5 * time.Minute

	return gocron.DurationRandomJob(interval-jitter, interval+jitter)
}
