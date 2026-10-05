package auditlogs

import (
	"context"
	"fmt"
	"log/slog"

	userAgentParser "github.com/mileusna/useragent"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type service struct {
	db               *gorm.DB
	emailSender      NewLoginEmailSender
	ipLocator        iplocation.Resolver
	appConfigService appconfig.AppConfigResolver
}

func newService(db *gorm.DB, emailSender NewLoginEmailSender, ipLocator iplocation.Resolver, appConfigService appconfig.AppConfigResolver) *service {
	return &service{
		db:               db,
		emailSender:      emailSender,
		ipLocator:        ipLocator,
		appConfigService: appConfigService,
	}
}

// Create creates a new audit log entry in the database
func (s *service) Create(ctx context.Context, event Event, ipAddress, userAgent, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	country, city, err := s.ipLocator.GetLocationByIP(ctx, ipAddress)
	if err != nil {
		// Log the error but don't interrupt the operation
		slog.WarnContext(ctx, "Failed to get IP location", slog.String("ip", ipAddress), slog.Any("error", err))
	}

	auditLog := AuditLog{
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
		slog.ErrorContext(ctx, "Failed to create audit log", "error", err)
		return AuditLog{}, false
	}

	return auditLog, true
}

// ListAuditLogsForUser retrieves all audit logs for a given user ID
func (s *service) ListAuditLogsForUser(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]AuditLog, utils.PaginationResponse, error) {
	var logs []AuditLog
	query := s.db.
		WithContext(ctx).
		Model(&AuditLog{}).
		Where("user_id = ?", userID)

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &logs)
	return logs, pagination, err
}

func (s *service) DeviceStringFromUserAgent(userAgent string) string {
	ua := userAgentParser.Parse(userAgent)
	return ua.Name + " on " + ua.OS + " " + ua.OSVersion
}

func (s *service) ListAllAuditLogs(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]AuditLog, utils.PaginationResponse, error) {
	var logs []AuditLog

	query := s.db.
		WithContext(ctx).
		Preload("User").
		Model(&AuditLog{})

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

func (s *service) ListUsernamesWithIds(ctx context.Context) (users map[string]string, err error) {
	query := s.db.
		WithContext(ctx).
		Joins("User").
		Model(&AuditLog{}).
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

func (s *service) ListClientNames(ctx context.Context) (clientNames []string, err error) {
	dialect := s.db.Name()
	query := s.db.
		WithContext(ctx).
		Model(&AuditLog{})

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
