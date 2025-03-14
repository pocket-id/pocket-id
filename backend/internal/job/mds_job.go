package job

import (
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type MdsJobs struct {
	mdsService *service.MdsService
}

func RegisterMdsJobs(mdsService *service.MdsService) {
	jobs := &MdsJobs{mdsService: mdsService}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	registerJob(scheduler, "UpdateMdsData", "0 * * * *", jobs.updateMdsData)

	// Run the job immediately on startup (optional)
	if err := jobs.updateMdsData(); err != nil {
		log.Printf("Failed to update MDS data: %s", err)
	}

	scheduler.Start()
}

func (j *MdsJobs) updateMdsData() error {
	return j.mdsService.UpdateAaguidMap()
}
