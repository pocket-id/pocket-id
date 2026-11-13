package auditlogs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	userAgentParser "github.com/mileusna/useragent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type service struct {
	db               *gorm.DB
	emailSender      NewLoginEmailSender
	ipLocator        iplocation.Resolver
	appConfigService appconfig.AppConfigResolver
	browserTokens    browserTokenService
}

func newService(db *gorm.DB, emailSender NewLoginEmailSender, ipLocator iplocation.Resolver, appConfigService appconfig.AppConfigResolver, browserTokens browserTokenService) *service {
	return &service{
		db:               db,
		emailSender:      emailSender,
		ipLocator:        ipLocator,
		appConfigService: appConfigService,
		browserTokens:    browserTokens,
	}
}

// Create records an action users take on their own account while they sign in or authorize an app, before they hold a credential the entry could name
func (s *service) Create(ctx context.Context, event Event, ipAddress, userAgent, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	origin := Origin{Principal: authz.Principal{UserID: userID}, IPAddress: ipAddress, UserAgent: userAgent}
	return s.Record(ctx, event, origin, userID, data, tx)
}

func (s *service) insert(ctx context.Context, entry AuditLog, ipAddress string, tx *gorm.DB) (AuditLog, bool) {
	country, city, err := s.ipLocator.GetLocationByIP(ctx, ipAddress)
	if err != nil {
		// Log the error but don't interrupt the operation
		slog.WarnContext(ctx, "Failed to get IP location", slog.String("ip", ipAddress), slog.Any("error", err))
	}
	entry.Country = country
	entry.City = city

	if ipAddress != "" {
		// Only set ipAddress if not empty, because on Postgres we use INET columns that don't allow non-null empty values
		entry.IpAddress = &ipAddress
	}

	// The user ID references a user, so empty columns store NULL instead of an empty string
	var omit []string
	for column, value := range map[string]string{"UserID": entry.UserID, "ActorType": string(entry.ActorType), "ActorID": entry.ActorID, "ActorName": entry.ActorName} {
		if value == "" {
			omit = append(omit, column)
		}
	}
	err = tx.WithContext(ctx).Omit(omit...).Create(&entry).Error
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create audit log", "error", err)
		return AuditLog{}, false
	}

	return entry, true
}

// KeepUsername copies the username into the user's entries, so they still name the user once deleting the user sets their user ID to NULL
func (s *service) KeepUsername(ctx context.Context, userID, username string, tx *gorm.DB) error {
	var data clause.Expr
	switch dialect := tx.Name(); dialect {
	case "sqlite":
		// json_set returns TEXT, so cast it back to the BLOB the column stores
		data = gorm.Expr("CAST(json_set(data, '$.username', ?) AS BLOB)", username)
	case "postgres":
		data = gorm.Expr("data || jsonb_build_object('username', ?::text)", username)
	default:
		return fmt.Errorf("unsupported database dialect: %s", dialect)
	}

	return tx.WithContext(ctx).Model(&AuditLog{}).Where("user_id = ?", userID).UpdateColumn("data", data).Error
}

// DateRange bounds a list by creation time, from inclusive and to exclusive; a zero bound is open
type DateRange struct {
	From time.Time
	To   time.Time
}

func (r DateRange) apply(query *gorm.DB) *gorm.DB {
	if !r.From.IsZero() {
		query = query.Where("created_at >= ?", datatype.DateTime(r.From))
	}
	if !r.To.IsZero() {
		query = query.Where("created_at < ?", datatype.DateTime(r.To))
	}
	return query
}

// ListAuditLogsForUser retrieves the entries about the user, leaving out what they did to other users, clients and groups
func (s *service) ListAuditLogsForUser(ctx context.Context, userID string, dateRange DateRange, listRequestOptions utils.ListRequestOptions) ([]AuditLog, utils.PaginationResponse, error) {
	var logs []AuditLog
	query := dateRange.apply(s.db.
		WithContext(ctx).
		Preload("User").
		Model(&AuditLog{}).
		Where("user_id = ?", userID))

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &logs)
	return logs, pagination, err
}

func (s *service) DeviceStringFromUserAgent(userAgent string) string {
	ua := userAgentParser.Parse(userAgent)
	// API clients such as curl or Terraform report no operating system
	if ua.OS == "" {
		return ua.Name
	}
	return ua.Name + " on " + ua.OS + " " + ua.OSVersion
}

func (s *service) ListAllAuditLogs(ctx context.Context, dateRange DateRange, listRequestOptions utils.ListRequestOptions) ([]AuditLog, utils.PaginationResponse, error) {
	var logs []AuditLog

	query := dateRange.apply(s.db.
		WithContext(ctx).
		Preload("User").
		Model(&AuditLog{}))

	// Filtering by a user also shows the actions they took
	if userIDs, ok := listRequestOptions.Filters["userID"]; ok && len(userIDs) > 0 {
		userID, _ := userIDs[0].(string)
		query = query.Where("(user_id = ? OR (actor_type = ? AND actor_id = ?))", userID, ActorUser, userID)
	}

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
