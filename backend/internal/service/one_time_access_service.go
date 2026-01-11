package service

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OneTimeAccessService struct {
	db               *gorm.DB
	userService      *UserService
	appConfigService *AppConfigService
	jwtService       *JwtService
	auditLogService  *AuditLogService
	emailService     *EmailService
}

func NewOneTimeAccessService(db *gorm.DB, userService *UserService, jwtService *JwtService, auditLogService *AuditLogService, emailService *EmailService, appConfigService *AppConfigService) *OneTimeAccessService {
	return &OneTimeAccessService{
		db:               db,
		userService:      userService,
		appConfigService: appConfigService,
		jwtService:       jwtService,
		auditLogService:  auditLogService,
		emailService:     emailService,
	}
}

func (s *OneTimeAccessService) RequestOneTimeAccessEmailAsAdmin(ctx context.Context, userID string, ttl time.Duration) error {
	isDisabled := !s.appConfigService.GetDbConfig().EmailOneTimeAccessAsAdminEnabled.IsTrue()
	if isDisabled {
		return &common.OneTimeAccessDisabledError{}
	}

	_, err := s.requestOneTimeAccessEmailInternal(ctx, userID, "", ttl, false)
	return err
}

func (s *OneTimeAccessService) RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx context.Context, userID, redirectPath string) (string, error) {
	isDisabled := !s.appConfigService.GetDbConfig().EmailOneTimeAccessAsUnauthenticatedEnabled.IsTrue()
	if isDisabled {
		return "", &common.OneTimeAccessDisabledError{}
	}

	var userId string
	err := s.db.Model(&model.User{}).Select("id").Where("email = ?", userID).First(&userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Do not return error if user not found to prevent email enumeration
		return "", nil
	} else if err != nil {
		return "", err
	}

	deviceToken, err := s.requestOneTimeAccessEmailInternal(ctx, userId, redirectPath, 15*time.Minute, true)
	if err != nil {
		return "", err
	} else if deviceToken == nil {
		return "", errors.New("device token expected but not returned")
	}

	return *deviceToken, nil
}

func (s *OneTimeAccessService) requestOneTimeAccessEmailInternal(ctx context.Context, userID, redirectPath string, ttl time.Duration, withDeviceToken bool) (*string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Email == nil {
		return nil, &common.UserEmailNotSetError{}
	}

	oneTimeAccessToken, deviceToken, err := s.createOneTimeAccessTokenInternal(ctx, user.ID, ttl, withDeviceToken, tx)
	if err != nil {
		return nil, err
	}
	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	// We use a background context here as this is running in a goroutine
	//nolint:contextcheck
	go func() {
		span := trace.SpanFromContext(ctx)
		innerCtx := trace.ContextWithSpan(context.Background(), span)

		link := common.EnvConfig.AppURL + "/lc"
		linkWithCode := link + "/" + oneTimeAccessToken

		// Add redirect path to the link
		if strings.HasPrefix(redirectPath, "/") {
			encodedRedirectPath := url.QueryEscape(redirectPath)
			linkWithCode = linkWithCode + "?redirect=" + encodedRedirectPath
		}

		errInternal := SendEmail(innerCtx, s.emailService, email.Address{
			Name:  user.FullName(),
			Email: *user.Email,
		}, OneTimeAccessTemplate, &OneTimeAccessTemplateData{
			Code:              oneTimeAccessToken,
			LoginLink:         link,
			LoginLinkWithCode: linkWithCode,
			ExpirationString:  utils.DurationToString(ttl),
		})
		if errInternal != nil {
			slog.ErrorContext(innerCtx, "Failed to send one-time access token email", slog.Any("error", errInternal), slog.String("address", *user.Email))
			return
		}
	}()

	return deviceToken, nil
}

func (s *OneTimeAccessService) CreateOneTimeAccessToken(ctx context.Context, userID string, ttl time.Duration) (token string, err error) {
	token, _, err = s.createOneTimeAccessTokenInternal(ctx, userID, ttl, false, s.db)
	return token, err
}

func (s *OneTimeAccessService) createOneTimeAccessTokenInternal(ctx context.Context, userID string, ttl time.Duration, withDeviceToken bool, tx *gorm.DB) (token string, deviceToken *string, err error) {
	oneTimeAccessToken, err := NewOneTimeAccessToken(userID, ttl, withDeviceToken)
	if err != nil {
		return "", nil, err
	}

	err = tx.WithContext(ctx).Create(oneTimeAccessToken).Error
	if err != nil {
		return "", nil, err
	}

	return oneTimeAccessToken.Token, oneTimeAccessToken.DeviceToken, nil
}

func (s *OneTimeAccessService) ExchangeOneTimeAccessToken(ctx context.Context, token, deviceToken, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var oneTimeAccessToken model.OneTimeAccessToken
	err := tx.
		WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, datatype.DateTime(time.Now())).
		Preload("User").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&oneTimeAccessToken).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}
		return model.User{}, "", err
	}
	if oneTimeAccessToken.DeviceToken != nil && deviceToken != *oneTimeAccessToken.DeviceToken {
		return model.User{}, "", &common.DeviceCodeInvalid{}
	}

	accessToken, err := s.jwtService.GenerateAccessToken(oneTimeAccessToken.User)
	if err != nil {
		return model.User{}, "", err
	}

	err = tx.
		WithContext(ctx).
		Delete(&oneTimeAccessToken).
		Error
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLogService.Create(ctx, model.AuditLogEventOneTimeAccessTokenSignIn, ipAddress, userAgent, oneTimeAccessToken.User.ID, model.AuditLogData{}, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return oneTimeAccessToken.User, accessToken, nil
}

func NewOneTimeAccessToken(userID string, ttl time.Duration, withDeviceToken bool) (*model.OneTimeAccessToken, error) {
	// If expires at is less than 15 minutes, use a 6-character token instead of 16
	tokenLength := 16
	if ttl <= 15*time.Minute {
		tokenLength = 6
	}

	token, err := utils.GenerateRandomUnambiguousString(tokenLength)
	if err != nil {
		return nil, err
	}

	var deviceToken *string
	if withDeviceToken {
		dt, err := utils.GenerateRandomAlphanumericString(16)
		if err != nil {
			return nil, err
		}
		deviceToken = &dt
	}

	now := time.Now().Round(time.Second)
	o := &model.OneTimeAccessToken{
		UserID:      userID,
		ExpiresAt:   datatype.DateTime(now.Add(ttl)),
		Token:       token,
		DeviceToken: deviceToken,
	}

	return o, nil
}
