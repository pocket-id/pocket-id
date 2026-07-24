package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

func (s *Scheduler) RegisterDbCleanupJobs(ctx context.Context, db *gorm.DB, appConfigService *appconfig.AppConfigService) error {
	jobs := &DbCleanupJobs{db: db, appConfigService: appConfigService}

	newBackOff := func() *backoff.ExponentialBackOff {
		bo := backoff.NewExponentialBackOff()
		bo.Multiplier = 4
		bo.RandomizationFactor = 0.1
		bo.InitialInterval = time.Second
		bo.MaxInterval = 45 * time.Second
		return bo
	}

	// Use exponential backoff for each DB cleanup job so transient query failures are retried automatically rather than causing an immediate job failure
	return errors.Join(
		s.RegisterJob(ctx, "ClearWebauthnSessions", jobDefWithJitter(24*time.Hour), jobs.clearWebauthnSessions, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearOneTimeAccessTokens", jobDefWithJitter(24*time.Hour), jobs.clearOneTimeAccessTokens, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearSignupTokens", jobDefWithJitter(24*time.Hour), jobs.clearSignupTokens, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearEmailVerificationTokens", jobDefWithJitter(24*time.Hour), jobs.clearEmailVerificationTokens, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearOAuth2Sessions", jobDefWithJitter(24*time.Hour), jobs.clearOAuth2Sessions, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearOAuth2JTIs", jobDefWithJitter(24*time.Hour), jobs.clearOAuth2JTIs, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearInteractionSessions", jobDefWithJitter(24*time.Hour), jobs.clearInteractionSessions, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearReauthenticationTokens", jobDefWithJitter(24*time.Hour), jobs.clearReauthenticationTokens, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearAuditLogs", jobDefWithJitter(24*time.Hour), jobs.clearAuditLogs, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
		s.RegisterJob(ctx, "ClearInactiveDynamicClients", jobDefWithJitter(24*time.Hour), jobs.clearInactiveDynamicClients, service.RegisterJobOpts{RunImmediately: true, BackOff: newBackOff()}),
	)
}

type DbCleanupJobs struct {
	db               *gorm.DB
	appConfigService *appconfig.AppConfigService
}

// clearWebauthnSessions deletes expired WebAuthn challenge sessions.
func (j *DbCleanupJobs) clearWebauthnSessions(ctx context.Context) error {
	count, err := webauthn.CleanupExpiredSessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean expired WebAuthn sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned expired WebAuthn sessions", slog.Int64("count", count))

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

// clearSignupTokens deletes signup tokens that have expired
func (j *DbCleanupJobs) clearSignupTokens(ctx context.Context) error {
	count, err := usersignup.CleanupExpiredSignupTokens(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean expired signup tokens: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned expired signup tokens", slog.Int64("count", count))

	return nil
}

// clearOAuth2Sessions deletes expired and invalidated OAuth2 sessions.
func (j *DbCleanupJobs) clearOAuth2Sessions(ctx context.Context) error {
	count, err := oidc.CleanupExpiredOAuth2Sessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean OAuth2 sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned OAuth2 sessions", slog.Int64("count", count))

	return nil
}

// clearOAuth2JTIs deletes expired JWT IDs used for client assertion replay protection.
func (j *DbCleanupJobs) clearOAuth2JTIs(ctx context.Context) error {
	count, err := oidc.CleanupExpiredClientAssertionJTIs(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean OAuth2 client assertion JTIs: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned OAuth2 client assertion JTIs", slog.Int64("count", count))

	return nil
}

// clearInteractionSessions deletes abandoned OIDC interaction sessions.
func (j *DbCleanupJobs) clearInteractionSessions(ctx context.Context) error {
	count, err := oidc.CleanupAbandonedInteractionSessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean interaction sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned interaction sessions", slog.Int64("count", count))

	return nil
}

// clearReauthenticationTokens deletes expired reauthentication tokens.
// What counts as expired is owned by the webauthn module.
func (j *DbCleanupJobs) clearReauthenticationTokens(ctx context.Context) error {
	count, err := webauthn.CleanupExpiredReauthenticationTokens(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean expired reauthentication tokens: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned expired reauthentication tokens", slog.Int64("count", count))

	return nil
}

// ClearAuditLogs deletes audit logs older than the configured retention window
func (j *DbCleanupJobs) clearAuditLogs(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -common.EnvConfig.AuditLogRetentionDays)

	st := j.db.
		WithContext(ctx).
		Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(cutoff))
	if st.Error != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", st.Error)
	}

	slog.InfoContext(ctx, "Deleted old audit logs", slog.Int64("count", st.RowsAffected))

	return nil
}

// clearInactiveDynamicClients deletes dynamically registered clients that have
// been inactive for longer than the configured retention window.
//
// For CIMD, the metadata_expires_at column is bumped every time a client's
// metadata document is resolved, so a value far in the past means the client
// has been inactive since then.
//
// A retention of 0 (or less) disables the cleanup.
func (j *DbCleanupJobs) clearInactiveDynamicClients(ctx context.Context) error {
	retention := j.appConfigService.GetDynamicClientRetention()
	if retention <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-retention)

	st := j.db.
		WithContext(ctx).
		Where("client_type = ?", model.OidcClientTypeCIMD).
		Where("metadata_expires_at IS NOT NULL AND metadata_expires_at < ?", datatype.DateTime(cutoff)).
		Delete(&model.OidcClient{})
	if st.Error != nil {
		return fmt.Errorf("failed to delete inactive dynamic clients: %w", st.Error)
	}

	slog.InfoContext(ctx, "Deleted inactive dynamic clients", slog.Int64("count", st.RowsAffected))

	return nil
}

// ClearEmailVerificationTokens deletes email verification tokens that have expired
func (j *DbCleanupJobs) clearEmailVerificationTokens(ctx context.Context) error {
	st := j.db.
		WithContext(ctx).
		Delete(&model.EmailVerificationToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	if st.Error != nil {
		return fmt.Errorf("failed to clean expired email verification tokens: %w", st.Error)
	}

	slog.InfoContext(ctx, "Cleaned expired email verification tokens", slog.Int64("count", st.RowsAffected))
	return nil
}
