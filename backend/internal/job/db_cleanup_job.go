package job

import (
	"context"
	"errors"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

func (s *Scheduler) RegisterDbCleanupJobs(ctx context.Context, db *gorm.DB) error {
	jobs := &DbCleanupJobs{db: db}

	// Run every 12 hours, but with some jitter so they don't run at the exact same time
	def := gocron.DurationRandomJob(12*time.Hour-2*time.Minute, 12*time.Hour+2*time.Minute)
	return errors.Join(
		s.registerJob(ctx, "ClearWebauthnSessions", def, jobs.clearWebauthnSessions, true),
		s.registerJob(ctx, "ClearOneTimeAccessTokens", def, jobs.clearOneTimeAccessTokens, true),
		s.registerJob(ctx, "ClearOidcAuthorizationCodes", def, jobs.clearOidcAuthorizationCodes, true),
		s.registerJob(ctx, "ClearOidcRefreshTokens", def, jobs.clearOidcRefreshTokens, true),
		s.registerJob(ctx, "ClearAuditLogs", def, jobs.clearAuditLogs, true),
	)
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
