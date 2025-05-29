package job

import (
	"context"
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

	// Run every 24 hours and now
	return s.registerJob(ctx, "ClearExpiredDatabaseRecords", gocron.DurationJob(24*time.Hour), jobs.clearExpiredRecords, true)
}

type DbCleanupJobs struct {
	db *gorm.DB
}

func (j *DbCleanupJobs) clearExpiredRecords(ctx context.Context) error {
	return j.db.Transaction(func(tx *gorm.DB) (err error) {
		// Deletes WebAuthn sessions that have expired
		err = tx.
			WithContext(ctx).
			Delete(&model.WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now())).
			Error
		if err != nil {
			return fmt.Errorf("failed to clean expired WebAuthn sessions: %w", err)
		}
		slog.InfoContext(ctx, "Cleaned expired WebAuthn sessions", slog.Int64("count", tx.RowsAffected))

		// Deletes one-time access tokens that have expired
		err = tx.
			WithContext(ctx).
			Delete(&model.OneTimeAccessToken{}, "expires_at < ?", datatype.DateTime(time.Now())).
			Error
		if err != nil {
			return fmt.Errorf("failed to clean expired one-time access tokens: %w", err)
		}
		slog.InfoContext(ctx, "Cleaned expired one-time access tokens", slog.Int64("count", tx.RowsAffected))

		// Deletes OIDC authorization codes that have expired
		err = tx.
			WithContext(ctx).
			Delete(&model.OidcAuthorizationCode{}, "expires_at < ?", datatype.DateTime(time.Now())).
			Error
		if err != nil {
			return fmt.Errorf("failed to clean expired OIDC authorization codes: %w", err)
		}
		slog.InfoContext(ctx, "Cleaned expired OIDC authorization codes", slog.Int64("count", tx.RowsAffected))

		// Deletes OIDC refresh tokens that have expired
		err = tx.
			WithContext(ctx).
			Delete(&model.OidcRefreshToken{}, "expires_at < ?", datatype.DateTime(time.Now())).
			Error
		if err != nil {
			return fmt.Errorf("failed to clean expired OIDC refresh tokens: %w", err)
		}
		slog.InfoContext(ctx, "Cleaned expired OIDC refresh tokens", slog.Int64("count", tx.RowsAffected))

		// Deletes audit logs older than 90 days
		err = tx.
			WithContext(ctx).
			Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(time.Now().AddDate(0, 0, -90))).
			Error
		if err != nil {
			return fmt.Errorf("failed to delete old audit logs: %w", err)
		}
		slog.InfoContext(ctx, "Deleted old audit logs", slog.Int64("count", tx.RowsAffected))

		return nil
	})
}
