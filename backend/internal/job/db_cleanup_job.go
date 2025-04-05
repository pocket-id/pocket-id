package job

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

func RegisterDbCleanupJobs(ctx context.Context, db *gorm.DB) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	jobs := &DbCleanupJobs{db: db}

	registerJob(ctx, scheduler, "ClearWebauthnSessions", "0 3 * * *", jobs.clearWebauthnSessions)
	registerJob(ctx, scheduler, "ClearOneTimeAccessTokens", "0 3 * * *", jobs.clearOneTimeAccessTokens)
	registerJob(ctx, scheduler, "ClearOidcAuthorizationCodes", "0 3 * * *", jobs.clearOidcAuthorizationCodes)
	registerJob(ctx, scheduler, "ClearOidcRefreshTokens", "0 3 * * *", jobs.clearOidcRefreshTokens)
	registerJob(ctx, scheduler, "ClearAuditLogs", "0 3 * * *", jobs.clearAuditLogs)
	scheduler.Start()
}

type DbCleanupJobs struct {
	db *gorm.DB
}

// ClearWebauthnSessions deletes WebAuthn sessions that have expired
func (j *DbCleanupJobs) clearWebauthnSessions(ctx context.Context) error {
	return j.db.
		WithContext(ctx).
		Delete(&model.WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now())).
		Error
}

// ClearOneTimeAccessTokens deletes one-time access tokens that have expired
func (j *DbCleanupJobs) clearOneTimeAccessTokens(ctx context.Context) error {
	return j.db.
		WithContext(ctx).
		Delete(&model.OneTimeAccessToken{}, "expires_at < ?", datatype.DateTime(time.Now())).
		Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcAuthorizationCodes(ctx context.Context) error {
	return j.db.
		WithContext(ctx).
		Delete(&model.OidcAuthorizationCode{}, "expires_at < ?", datatype.DateTime(time.Now())).
		Error
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcRefreshTokens(ctx context.Context) error {
	return j.db.
		WithContext(ctx).
		Delete(&model.OidcRefreshToken{}, "expires_at < ?", datatype.DateTime(time.Now())).
		Error
}

// ClearAuditLogs deletes audit logs older than 90 days
func (j *DbCleanupJobs) clearAuditLogs(ctx context.Context) error {
	return j.db.
		WithContext(ctx).
		Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(time.Now().AddDate(0, 0, -90))).
		Error
}
