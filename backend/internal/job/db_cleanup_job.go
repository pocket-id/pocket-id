package job

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
)

func RegisterDbCleanupJobs(db *gorm.DB) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	jobs := &DbCleanupJobs{db: db}

	registerJob(scheduler, "ClearWebauthnSessions", "0 3 * * *", jobs.clearWebauthnSessions)
	registerJob(scheduler, "ClearOneTimeAccessTokens", "0 3 * * *", jobs.clearOneTimeAccessTokens)
	registerJob(scheduler, "ClearOidcAuthorizationCodes", "0 3 * * *", jobs.clearOidcAuthorizationCodes)
	registerJob(scheduler, "ClearOidcRefreshTokens", "0 3 * * *", jobs.clearOidcRefreshTokens)
	registerJob(scheduler, "ClearAuditLogs", "0 3 * * *", jobs.clearAuditLogs)
	scheduler.Start()
}

type DbCleanupJobs struct {
	db *gorm.DB
}

// ClearWebauthnSessions deletes WebAuthn sessions that have expired
func (j *DbCleanupJobs) clearWebauthnSessions() error {
	return j.db.Delete(&model.WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOneTimeAccessTokens deletes one-time access tokens that have expired
func (j *DbCleanupJobs) clearOneTimeAccessTokens() error {
	return j.db.Debug().Delete(&model.OneTimeAccessToken{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcAuthorizationCodes() error {
	return j.db.Delete(&model.OidcAuthorizationCode{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcRefreshTokens() error {
	return j.db.Delete(&model.OidcRefreshToken{}, "expires_at < ?", datatype.DateTime(time.Now())).Error
}

// ClearAuditLogs deletes audit logs older than 90 days
func (j *DbCleanupJobs) clearAuditLogs() error {
	return j.db.Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(time.Now().AddDate(0, 0, -90))).Error
}
