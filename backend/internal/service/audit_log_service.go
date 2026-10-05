package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	userAgentParser "github.com/mileusna/useragent"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type NewLoginEmailSender interface {
	SendNewLogin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, ipAddress, country, city, device, method string, dateTime time.Time) error
}

type AuditLogService struct {
	db               *gorm.DB
	emailSender      NewLoginEmailSender
	ipLocator        iplocation.Resolver
	appConfigService appconfig.AppConfigResolver
}

func NewAuditLogService(db *gorm.DB, emailSender NewLoginEmailSender, ipLocator iplocation.Resolver, appConfigService appconfig.AppConfigResolver) *AuditLogService {
	return &AuditLogService{
		db:               db,
		emailSender:      emailSender,
		ipLocator:        ipLocator,
		appConfigService: appConfigService,
	}
}

// Create creates a new audit log entry in the database
func (s *AuditLogService) Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool) {
	country, city, err := s.ipLocator.GetLocationByIP(ctx, ipAddress)
	if err != nil {
		// Log the error but don't interrupt the operation
		slog.WarnContext(ctx, "Failed to get IP location", slog.String("ip", ipAddress), slog.Any("error", err))
	}

	auditLog := model.AuditLog{
		Event:     event,
		Country:   country,
		City:      city,
		UserAgent: userAgent,
		UserID:    userID,
		Data:      data,
	}

	if ipAddress != "" {
		// Only set ipAddress if not empty, because on Postgres we use INET columns that don't allow non-null empty values
		auditLog.IpAddress = &ipAddress
	}

	// Save the audit log in the database
	err = tx.
		WithContext(ctx).
		Create(&auditLog).
		Error
	if err != nil {
		slog.Error("Failed to create audit log", "error", err)
		return model.AuditLog{}, false
	}

	return auditLog, true
}

// CreateSignIn records a successful login and recognizes either its browser cookie or its exact IP and User-Agent
// The caller must send the notification only after the login commits
func (s *AuditLogService) CreateSignIn(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, emailLoginNotificationEnabled bool) model.SignInResult {
	entry, created := s.Create(ctx, event, ipAddress, userAgent, userID, model.AuditLogData{}, tx)
	result := model.SignInResult{AuditLog: entry, Created: created}
	if !created {
		return result
	}

	// Remember the browser even when notifications are disabled so enabling them does not forget existing browsers
	known, token, err := s.rememberBrowser(ctx, tx, userID, browserToken)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to remember sign-in browser", slog.Any("error", err))
	}
	result.KnownBrowserToken = token
	if known || !emailLoginNotificationEnabled {
		return result
	}

	// Only earlier successful sign-ins for this user can satisfy the fallback
	var count int64
	query := tx.WithContext(ctx).Model(&model.AuditLog{}).
		Where("user_id = ? AND user_agent = ? AND id <> ?", userID, userAgent, entry.ID).
		Where("event IN ?", []model.AuditLogEvent{model.AuditLogEventSignIn, model.AuditLogEventOneTimeAccessTokenSignIn, model.AuditLogEventRemoteSignIn})
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

func (s *AuditLogService) rememberBrowser(ctx context.Context, tx *gorm.DB, userID, token string) (bool, string, error) {
	now := time.Now()
	expiresAt := now.Add(model.KnownBrowserLifetime).Unix()

	// Extend only an unexpired token already associated with the authenticated user
	result := tx.WithContext(ctx).Model(&model.KnownBrowser{}).
		Where("user_id = ? AND token_hash = ? AND expires_at > ?", userID, utils.CreateSha256Hash(token), now.Unix()).
		Update("expires_at", expiresAt)
	if result.Error != nil {
		return false, "", result.Error
	}
	if result.RowsAffected > 0 {
		return true, token, nil
	}

	// Replace unrecognized tokens with server-generated randomness to avoid accepting a caller-chosen browser identity
	token = rand.Text()
	record := model.KnownBrowser{UserID: userID, TokenHash: utils.CreateSha256Hash(token), ExpiresAt: expiresAt}
	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return false, "", err
	}
	return false, token, nil
}

// SendSignInNotification sends only after the caller has completed the login successfully
func (s *AuditLogService) SendSignInNotification(ctx context.Context, result model.SignInResult) {
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
func signInMethod(event model.AuditLogEvent) string {
	switch event { //nolint:exhaustive // Other audit events are not sign-ins
	case model.AuditLogEventSignIn:
		return "Passkey"
	case model.AuditLogEventOneTimeAccessTokenSignIn:
		return "One-time code"
	case model.AuditLogEventRemoteSignIn:
		return "Another device (QR code)"
	default:
		return "Unknown"
	}
}

// ListAuditLogsForUser retrieves all audit logs for a given user ID
func (s *AuditLogService) ListAuditLogsForUser(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]model.AuditLog, utils.PaginationResponse, error) {
	var logs []model.AuditLog
	query := s.db.
		WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("user_id = ?", userID)

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &logs)
	return logs, pagination, err
}

func (s *AuditLogService) DeviceStringFromUserAgent(userAgent string) string {
	ua := userAgentParser.Parse(userAgent)
	return ua.Name + " on " + ua.OS + " " + ua.OSVersion
}

func (s *AuditLogService) ListAllAuditLogs(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]model.AuditLog, utils.PaginationResponse, error) {
	var logs []model.AuditLog

	query := s.db.
		WithContext(ctx).
		Preload("User").
		Model(&model.AuditLog{})

	if clientName, ok := listRequestOptions.Filters["clientName"]; ok {
		dialect := s.db.Name()
		switch dialect {
		case "sqlite":
			query = query.Where("json_extract(data, '$.clientName') IN ?", clientName)
		case "postgres":
			query = query.Where("data->>'clientName' IN ?", clientName)
		default:
			return nil, utils.PaginationResponse{}, fmt.Errorf("unsupported database dialect: %s", dialect)
		}
	}

	if locations, ok := listRequestOptions.Filters["location"]; ok {
		mapped := make([]string, 0, len(locations))
		for _, v := range locations {
			if s, ok := v.(string); ok {
				switch s {
				case "internal":
					mapped = append(mapped, "Internal Network")
				case "external":
					mapped = append(mapped, "External Network")
				}
			}
		}
		if len(mapped) > 0 {
			query = query.Where("country IN ?", mapped)
		}
	}

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &logs)
	if err != nil {
		return nil, pagination, err
	}

	return logs, pagination, nil
}

func (s *AuditLogService) ListUsernamesWithIds(ctx context.Context) (users map[string]string, err error) {
	query := s.db.
		WithContext(ctx).
		Joins("User").
		Model(&model.AuditLog{}).
		Select(`DISTINCT "User".id, "User".username`).
		Where(`"User".username IS NOT NULL`)

	type Result struct {
		ID       string `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}

	var results []Result
	err = query.Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query user IDs: %w", err)
	}

	users = make(map[string]string, len(results))
	for _, result := range results {
		users[result.ID] = result.Username
	}

	return users, nil
}

func (s *AuditLogService) ListClientNames(ctx context.Context) (clientNames []string, err error) {
	dialect := s.db.Name()
	query := s.db.
		WithContext(ctx).
		Model(&model.AuditLog{})

	switch dialect {
	case "sqlite":
		query = query.
			Select("DISTINCT json_extract(data, '$.clientName') AS client_name").
			Where("json_extract(data, '$.clientName') IS NOT NULL")
	case "postgres":
		query = query.
			Select("DISTINCT data->>'clientName' AS client_name").
			Where("data->>'clientName' IS NOT NULL")
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", dialect)
	}

	type Result struct {
		ClientName string `gorm:"column:client_name"`
	}

	var results []Result
	err = query.Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query client IDs: %w", err)
	}

	clientNames = make([]string, len(results))
	for i, result := range results {
		clientNames[i] = result.ClientName
	}

	return clientNames, nil
}
