package auditlogs

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// CreateSignIn records a successful login and recognizes either its browser cookie or its exact IP and User-Agent
// The caller must send the notification only after the login commits
func (s *service) CreateSignIn(ctx context.Context, event Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, emailLoginNotificationEnabled bool) SignInResult {
	entry, created := s.Create(ctx, event, ipAddress, userAgent, userID, Data{}, tx)
	result := SignInResult{AuditLog: entry, Created: created}
	if !created {
		return result
	}

	// Remember the browser even when notifications are disabled so enabling them does not forget existing browsers
	known, token, err := s.rememberBrowser(userID, browserToken)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to remember sign-in browser", slog.Any("error", err))
	}
	result.KnownBrowserToken = token
	if known || !emailLoginNotificationEnabled {
		return result
	}

	// Only earlier successful sign-ins for this user can satisfy the fallback
	var count int64
	query := tx.WithContext(ctx).Model(&AuditLog{}).
		Where("user_id = ? AND user_agent = ? AND id <> ?", userID, userAgent, entry.ID).
		Where("event IN ?", []Event{EventSignIn, EventOneTimeAccessTokenSignIn, EventRemoteSignIn})
	if ipAddress == "" {
		query = query.Where("ip_address IS NULL")
	} else {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if err := query.Count(&count).Error; err != nil {
		slog.ErrorContext(ctx, "Failed to check sign-in history", slog.Any("error", err))
		return result
	}
	result.Notify = count == 0
	return result
}

// SendSignInNotification sends only after the caller has completed the login successfully
func (s *service) SendSignInNotification(ctx context.Context, result SignInResult) {
	if !result.Created || !result.Notify {
		return
	}
	entry := result.AuditLog
	ipAddress := ""
	if entry.IpAddress != nil {
		ipAddress = *entry.IpAddress
	}
	go func() {
		// This runs in background, so use a context without cancellation (or it would be stopped when the request ends)
		// We still want to have a context derived from the request's to carry over tracing info
		innerCtx := context.WithoutCancel(ctx)

		// This runs after the request has completed, so we resolve the current config rather than threading the request's snapshot into the goroutine
		dbConfig, innerErr := s.appConfigService.GetConfig(innerCtx)
		if innerErr != nil {
			slog.ErrorContext(innerCtx, "Failed to load app configuration to send notification email", slog.Any("error", innerErr))
			return
		}

		// Note we don't use the transaction here because this is running in background
		var user model.User
		innerErr = s.db.
			WithContext(innerCtx).
			Where("id = ?", entry.UserID).
			First(&user).
			Error
		if innerErr != nil {
			slog.ErrorContext(innerCtx, "Failed to load user from database to send notification email", slog.Any("error", innerErr))
			return
		}

		if user.Email == nil {
			return
		}

		innerErr = s.emailSender.SendNewLogin(
			innerCtx,
			dbConfig,
			user.FullName(),
			*user.Email,
			ipAddress,
			entry.Country,
			entry.City,
			s.DeviceStringFromUserAgent(entry.UserAgent),
			signInMethod(entry.Event),
			entry.CreatedAt.UTC(),
		)
		if innerErr != nil {
			slog.ErrorContext(innerCtx, "Failed to send notification email", slog.Any("error", innerErr), slog.String("address", *user.Email))
			return
		}
	}()
}

// signInMethod describes the successful login rather than the credential used to approve another device
func signInMethod(event Event) string {
	switch event { //nolint:exhaustive // Other audit events are not sign-ins
	case EventSignIn:
		return "Passkey"
	case EventOneTimeAccessTokenSignIn:
		return "One-time code"
	case EventRemoteSignIn:
		return "Another device (QR code)"
	default:
		return "Unknown"
	}
}

// SignInResult defers notification delivery until the login has committed
type SignInResult struct {
	AuditLog          AuditLog
	Created           bool
	Notify            bool
	KnownBrowserToken string
}
