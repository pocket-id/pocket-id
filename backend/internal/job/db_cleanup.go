package job

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
)

func RegisterDbCleanupJobs(db *gorm.DB) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	jobs := &Jobs{db: db}

	registerJob(scheduler, "ClearWebauthnSessions", "0 3 * * *", jobs.clearWebauthnSessions)
	registerJob(scheduler, "ClearOneTimeAccessTokens", "0 3 * * *", jobs.clearOneTimeAccessTokens)
	registerJob(scheduler, "ClearOidcAuthorizationCodes", "0 3 * * *", jobs.clearOidcAuthorizationCodes)
	registerJob(scheduler, "ClearOidcRefreshTokens", "0 3 * * *", jobs.clearOidcRefreshTokens)
	registerJob(scheduler, "ClearAuditLogs", "0 3 * * *", jobs.clearAuditLogs)
	scheduler.Start()
}

type Jobs struct {
	db *gorm.DB
}

// ClearWebauthnSessions deletes WebAuthn sessions that have expired
func (j *Jobs) clearWebauthnSessions() error {
	return j.db.Delete(&model.WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOneTimeAccessTokens deletes one-time access tokens that have expired
func (j *Jobs) clearOneTimeAccessTokens() error {
	return j.db.Debug().Delete(&model.OneTimeAccessToken{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *Jobs) clearOidcAuthorizationCodes() error {
	return j.db.Delete(&model.OidcAuthorizationCode{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *Jobs) clearOidcRefreshTokens() error {
	return j.db.Delete(&model.OidcRefreshToken{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearAuditLogs deletes audit logs older than 90 days
func (j *Jobs) clearAuditLogs() error {
	return j.db.Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(time.Now().AddDate(0, 0, -90))).Error
}

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
