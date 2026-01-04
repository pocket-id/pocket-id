package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
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
				errs = append(errs, fmt.Errorf("failed to unqueue job %q with ID %q: %w", name, job.ID().String(), err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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

func (s *Scheduler) RegisterJob(ctx context.Context, name string, def gocron.JobDefinition, job func(ctx context.Context) error, runImmediately bool, extraOptions ...gocron.JobOption) error {
	jobOptions := []gocron.JobOption{
		gocron.WithContext(ctx),
		gocron.WithName(name),
		gocron.WithEventListeners(
			gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
				slog.Info("Starting job",
					slog.String("name", name),
					slog.String("id", jobID.String()),
				)
			}),
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				slog.Info("Job run successfully",
					slog.String("name", name),
					slog.String("id", jobID.String()),
				)
			}),
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
				slog.Error("Job failed with error",
					slog.String("name", name),
					slog.String("id", jobID.String()),
					slog.Any("error", err),
				)
			}),
		),
	}

	if runImmediately {
		jobOptions = append(jobOptions, gocron.JobOption(gocron.WithStartImmediately()))
	}

	jobOptions = append(jobOptions, extraOptions...)

	_, err := s.scheduler.NewJob(def, gocron.NewTask(job), jobOptions...)

	if err != nil {
		return fmt.Errorf("failed to register job %q: %w", name, err)
	}

	return nil
}
