package usersignup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

const (
	// cleanupJobInterval is how often overdue signups are deleted, which bounds how long an account can outlive its deadline
	cleanupJobInterval = time.Hour
	// cleanupJobJitter spreads each occurrence around its scheduled time, so it doesn't coincide with the other cleanup jobs
	cleanupJobJitter = 5 * time.Minute
)

// UserDeleter deletes a user together with everything that references it
type UserDeleter interface {
	DeleteUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, allowLdapDelete bool) error
}

type cleanupJob struct {
	db        *gorm.DB
	appConfig appconfig.AppConfigResolver
	users     UserDeleter
}

func newCleanupJob(db *gorm.DB, appConfig appconfig.AppConfigResolver, users UserDeleter) (*cronjob.CronJob, error) {
	job := &cleanupJob{
		db:        db,
		appConfig: appConfig,
		users:     users,
	}

	cronActor, err := cronjob.New(
		"DeleteUnverifiedSignups",
		cronjob.WithJob(job.deleteUnverifiedSignups),
		cronjob.WithInterval(cleanupJobInterval),
		cronjob.WithJitter(cleanupJobJitter),
		// Also run right after the job is first registered, so accounts that missed their deadline while Pocket ID wasn't running are removed at startup
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating unverified signup cleanup cron job: %w", err)
	}

	return cronActor, nil
}

// deleteUnverifiedSignups deletes self-signed-up accounts that did not verify their email before their deadline
func (j *cleanupJob) deleteUnverifiedSignups(ctx context.Context) error {
	dbConfig, err := j.appConfig.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("error loading app config: %w", err)
	}

	// Deadlines only apply while verification is required
	// Clearing them otherwise keeps accounts that were used in the meantime from being deleted once verification is required again
	if dbConfig.EmailVerification.String() != appconfig.EmailVerificationRequired {
		st := j.db.
			WithContext(ctx).
			Model(&model.User{}).
			Where("email_verification_deadline IS NOT NULL").
			Update("email_verification_deadline", nil)
		if st.Error != nil {
			return fmt.Errorf("failed to clear email verification deadlines: %w", st.Error)
		}
		if st.RowsAffected > 0 {
			slog.InfoContext(ctx, "Cleared email verification deadlines because verification is no longer required", slog.Int64("count", st.RowsAffected))
		}
		return nil
	}

	// Load the overdue accounts
	// Admins are exempt from email verification, so an account promoted to admin before its deadline is kept
	var userIDs []string
	err = j.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where("email_verification_deadline < ? AND email_verified = ? AND is_admin = ?", datatype.DateTime(time.Now()), false, false).
		Pluck("id", &userIDs).
		Error
	if err != nil {
		return fmt.Errorf("failed to list accounts that missed their email verification deadline: %w", err)
	}
	if len(userIDs) == 0 {
		return nil
	}

	// Delete each account through the user service so its client sessions, SCIM peers and profile picture are cleaned up as well
	// A failure is logged and the remaining accounts are still processed
	deleted := 0
	for _, userID := range userIDs {
		err = j.users.DeleteUser(ctx, dbConfig, userID, false)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to delete an account that missed its email verification deadline", slog.String("userID", userID), slog.Any("error", err))
			continue
		}
		deleted++
	}

	slog.InfoContext(ctx, "Deleted accounts that missed their email verification deadline", slog.Int("count", deleted))

	return nil
}
