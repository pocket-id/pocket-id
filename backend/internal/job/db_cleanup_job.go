package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

func (s *Scheduler) RegisterDbCleanupJobs(ctx context.Context, db *gorm.DB) error {
	jobs := &DbCleanupJobs{db: db}

	// Run every 24 hours (but with some jitter so they don't run at the exact same time), and now
	def := gocron.DurationRandomJob(24*time.Hour-2*time.Minute, 24*time.Hour+2*time.Minute)
	return errors.Join(
		s.registerJob(ctx, "ClearWebauthnSessions", def, jobs.clearWebauthnSessions, true),
		s.registerJob(ctx, "ClearOneTimeAccessTokens", def, jobs.clearOneTimeAccessTokens, true),
		s.registerJob(ctx, "ClearSignupTokens", def, jobs.clearSignupTokens, true),
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
	st := j.db.
		WithContext(ctx).
		Delete(&model.WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired WebAuthn sessions: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired WebAuthn sessions", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearOneTimeAccessTokens deletes one-time access tokens that have expired
func (j *DbCleanupJobs) clearOneTimeAccessTokens(ctx context.Context) error {
	st := j.db.
		WithContext(ctx).
		Delete(&model.OneTimeAccessToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired one-time access tokens: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired one-time access tokens", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearSignupTokens deletes signup tokens that have expired
func (j *DbCleanupJobs) clearSignupTokens(ctx context.Context) error {
	// Delete tokens that are expired OR have reached their usage limit
	st := j.db.
		WithContext(ctx).
		Delete(&model.SignupToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired tokens: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired tokens", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcAuthorizationCodes(ctx context.Context) error {
	st := j.db.
		WithContext(ctx).
		Delete(&model.OidcAuthorizationCode{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired OIDC authorization codes: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired OIDC authorization codes", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearOidcAuthorizationCodes deletes OIDC authorization codes that have expired
func (j *DbCleanupJobs) clearOidcRefreshTokens(ctx context.Context) error {
	st := j.db.
		WithContext(ctx).
		Delete(&model.OidcRefreshToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired OIDC refresh tokens: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired OIDC refresh tokens", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearAuditLogs deletes audit logs older than 90 days
func (j *DbCleanupJobs) clearAuditLogs(ctx context.Context) error {
	st := j.db.
		WithContext(ctx).
		Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(time.Now().AddDate(0, 0, -90)))
	if st.Error != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", st.Error)
	}

	slog.InfoContext(ctx, "Deleted old audit logs", slog.Int64("count", st.RowsAffected))

	return nil
}
