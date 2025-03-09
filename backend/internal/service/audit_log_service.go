package service

import (
	"log"

	userAgentParser "github.com/mileusna/useragent"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	"gorm.io/gorm"
)

type AuditLogService struct {
	db               *gorm.DB
	appConfigService *AppConfigService
	emailService     *EmailService
	geoliteService   *GeoLiteService
}

func NewAuditLogService(db *gorm.DB, appConfigService *AppConfigService, emailService *EmailService, geoliteService *GeoLiteService) *AuditLogService {
	return &AuditLogService{db: db, appConfigService: appConfigService, emailService: emailService, geoliteService: geoliteService}
}

// Create creates a new audit log entry in the database
func (s *AuditLogService) Create(event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData) model.AuditLog {
	country, city, err := s.geoliteService.GetLocationByIP(ipAddress)
	if err != nil {
		log.Printf("Failed to get IP location: %v\n", err)
	}

	auditLog := model.AuditLog{
		Event:     event,
		IpAddress: ipAddress,
		Country:   country,
		City:      city,
		UserAgent: userAgent,
		UserID:    userID,
		Data:      data,
	}

	// Save the audit log in the database
	if err := s.db.Create(&auditLog).Error; err != nil {
		log.Printf("Failed to create audit log: %v\n", err)
		return model.AuditLog{}
	}

	return auditLog
}

// CreateNewSignInWithEmail creates a new audit log entry in the database and sends an email if the device hasn't been used before
func (s *AuditLogService) CreateNewSignInWithEmail(ipAddress, userAgent, userID string) model.AuditLog {
	createdAuditLog := s.Create(model.AuditLogEventSignIn, ipAddress, userAgent, userID, model.AuditLogData{})

	// Count the number of times the user has logged in from the same device
	var count int64
	err := s.db.Model(&model.AuditLog{}).Where("user_id = ? AND ip_address = ? AND user_agent = ?", userID, ipAddress, userAgent).Count(&count).Error
	if err != nil {
		log.Printf("Failed to count audit logs: %v\n", err)
		return createdAuditLog
	}

	// If the user hasn't logged in from the same device before and email notifications are enabled, send an email
	if s.appConfigService.DbConfig.EmailLoginNotificationEnabled.Value == "true" && count <= 1 {
		go func() {
			var user model.User
			s.db.Where("id = ?", userID).First(&user)

			err := SendEmail(s.emailService, email.Address{
				Name:  user.Username,
				Email: user.Email,
			}, NewLoginTemplate, &NewLoginTemplateData{
				IPAddress: ipAddress,
				Country:   createdAuditLog.Country,
				City:      createdAuditLog.City,
				Device:    s.DeviceStringFromUserAgent(userAgent),
				DateTime:  createdAuditLog.CreatedAt.UTC(),
			})
			if err != nil {
				log.Printf("Failed to send email to '%s': %v\n", user.Email, err)
			}
		}()
	}

	return createdAuditLog
}

// ListAuditLogsForUser retrieves all audit logs for a given user ID
func (s *AuditLogService) ListAuditLogsForUser(userID string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.AuditLog, utils.PaginationResponse, error) {
	var logs []model.AuditLog
	query := s.db.Model(&model.AuditLog{}).Where("user_id = ?", userID)

	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &logs)
	return logs, pagination, err
}

func (s *AuditLogService) DeviceStringFromUserAgent(userAgent string) string {
	ua := userAgentParser.Parse(userAgent)
	return ua.Name + " on " + ua.OS + " " + ua.OSVersion
}

func (s *AuditLogService) ListAllAuditLogs(sortedPaginationRequest utils.SortedPaginationRequest) ([]model.AuditLog, utils.PaginationResponse, error) {
	var logs []model.AuditLog

	// Create the base query
	query := s.db.Model(&model.AuditLog{})

	// Apply filters if they exist
	if sortedPaginationRequest.Filters != nil {
		if userId, ok := sortedPaginationRequest.Filters["userId"]; ok && userId != "" {
			query = query.Where("user_id = ?", userId)
		}

		if event, ok := sortedPaginationRequest.Filters["event"]; ok && event != "" {
			query = query.Where("event = ?", event)
		}

		if clientId, ok := sortedPaginationRequest.Filters["clientId"]; ok && clientId != "" {
			query = query.Where("data->>'clientId' = ?", clientId)
		}
	}

	// Apply pagination and sorting
	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &logs)
	if err != nil {
		return nil, pagination, err
	}

	// Create a map to cache user lookups and avoid duplicate DB calls
	userCache := make(map[string]model.User)

	// For each audit log, fetch the associated user info if needed
	for i := range logs {
		if logs[i].UserID != "" {
			// Check if user is already in cache
			if user, found := userCache[logs[i].UserID]; found {
				logs[i].Username = user.Username
			} else {
				// Fetch user from database
				var user model.User
				if err := s.db.Where("id = ?", logs[i].UserID).First(&user).Error; err == nil {
					logs[i].Username = user.Username
					userCache[logs[i].UserID] = user
				}
			}
		}
	}

	return logs, pagination, nil
}
