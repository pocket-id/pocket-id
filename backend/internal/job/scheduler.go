package job

import (
	"context"
	"fmt"
	"log"

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

// Run the scheduler.
// This function blocks until the context is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	log.Println("Starting job scheduler")
	s.scheduler.Start()

	// Block until context is canceled
	<-ctx.Done()

	err := s.scheduler.Shutdown()
	if err != nil {
		log.Printf("[WARN] Error shutting down job scheduler: %v", err)
	} else {
		log.Println("Job scheduler shut down")
	}

	return nil
}

func (s *Scheduler) registerJob(ctx context.Context, name string, interval string, job func(ctx context.Context) error) error {
	_, err := s.scheduler.NewJob(
		gocron.CronJob(interval, false),
		gocron.NewTask(job),
		gocron.WithContext(ctx),
		gocron.WithEventListeners(
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				log.Printf("Job %q run successfully", name)
			}),
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
				log.Printf("Job %q failed with error: %v", name, err)
			}),
		),
	)

	if err != nil {
		return fmt.Errorf("failed to register job %q: %w", name, err)
	}

	return nil
}
