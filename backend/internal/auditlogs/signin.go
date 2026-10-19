package auditlogs

import (
	"context"
	"log/slog"

	userAgentParser "github.com/mileusna/useragent"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// CreateSignIn records a successful login and applies the global notification policy
// The caller must send the notification only after the login commits
func (s *service) CreateSignIn(ctx context.Context, event Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, notificationMode appconfig.AppConfigValue) SignInResult {
	data := Data{}
	if s.isNewBrowser(ctx, tx, userID, userAgent) {
		data["newBrowser"] = "true"
	}
	entry, created := s.Create(ctx, event, ipAddress, userAgent, userID, data, tx)
	result := SignInResult{AuditLog: entry, Created: created}
	if !created {
		return result
	}

	// Only browser recognition uses a persistent cookie; other modes rely on the selected notification policy
	switch notificationMode {
	case appconfig.LoginNotificationAlways:
		result.Notify = true
		return result
	case appconfig.LoginNotificationBrowserRecognition:
		known, token, err := s.rememberBrowser(userID, browserToken)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to remember sign-in browser", slog.Any("error", err))
		}
		result.KnownBrowserToken = token
		if known {
			return result
		}
	case appconfig.LoginNotificationIPAndUserAgent:
		// Check sign-in history without reading or renewing the browser token
	default:
		return result
	}

	// Only earlier successful sign-ins for this user can satisfy the fallback
	var count int64
	query := tx.WithContext(ctx).Model(&AuditLog{}).
		Where("user_id = ? AND user_agent = ? AND id <> ?", userID, userAgent, entry.ID).
		Where("event IN ?", signInEvents)
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

var signInEvents = []Event{EventSignIn, EventOneTimeAccessTokenSignIn, EventRemoteSignIn}

// isNewBrowser reports whether the user has signed in before, but never with this browser on this operating system
// It ignores versions, so a browser update is not a new browser, and it never flags an account's first sign-in
func (s *service) isNewBrowser(ctx context.Context, tx *gorm.DB, userID, userAgent string) bool {
	var agents []string
	err := tx.WithContext(ctx).Model(&AuditLog{}).
		Where("user_id = ? AND event IN ?", userID, signInEvents).
		Distinct().
		Pluck("user_agent", &agents).
		Error
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check sign-in browsers", slog.Any("error", err))
		return false
	}
	if len(agents) == 0 {
		return false
	}

	family := browserFamily(userAgent)
	for _, agent := range agents {
		if browserFamily(agent) == family {
			return false
		}
	}
	return true
}

func browserFamily(userAgent string) string {
	ua := userAgentParser.Parse(userAgent)
	return ua.Name + "|" + ua.OS
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
