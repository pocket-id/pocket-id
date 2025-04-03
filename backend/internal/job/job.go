package job

import (
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

func registerJob(scheduler gocron.Scheduler, name string, interval string, job func() error) {
	_, err := scheduler.NewJob(
		gocron.CronJob(interval, false),
		gocron.NewTask(job),
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
		log.Fatalf("Failed to register job %q: %v", name, err)
	}
}
