package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserSignUpService struct {
	db               *gorm.DB
	userService      *UserService
	jwtService       *JwtService
	auditLogService  *AuditLogService
	appConfigService *AppConfigService
}

func NewUserSignupService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, appConfigService *AppConfigService, userService *UserService) *UserSignUpService {
	return &UserSignUpService{
		db:               db,
		jwtService:       jwtService,
		auditLogService:  auditLogService,
		appConfigService: appConfigService,
		userService:      userService,
	}
}

func (s *UserSignUpService) SignUp(ctx context.Context, signupData dto.SignUpDto, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	tokenProvided := signupData.Token != ""

	config := s.appConfigService.GetDbConfig()
	if config.AllowUserSignups.Value != "open" && !tokenProvided {
		return model.User{}, "", &common.OpenSignupDisabledError{}
	}

	var signupToken model.SignupToken
	var userGroupIDs []string
	if tokenProvided {
		err := tx.
			WithContext(ctx).
			Preload("UserGroups").
			Where("token = ?", signupData.Token).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&signupToken).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.User{}, "", &common.TokenInvalidOrExpiredError{}
			}
			return model.User{}, "", err
		}

		if !signupToken.IsValid() {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}

		for _, group := range signupToken.UserGroups {
			userGroupIDs = append(userGroupIDs, group.ID)
		}
	}

	userToCreate := dto.UserCreateDto{
		Username:      signupData.Username,
		Email:         signupData.Email,
		FirstName:     signupData.FirstName,
		LastName:      signupData.LastName,
		DisplayName:   strings.TrimSpace(signupData.FirstName + " " + signupData.LastName),
		UserGroupIds:  userGroupIDs,
		EmailVerified: s.appConfigService.GetDbConfig().EmailsVerified.IsTrue(),
	}

	user, err := s.userService.createUserInternal(ctx, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return model.User{}, "", err
	}

	if tokenProvided {
		s.auditLogService.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"signupToken": signupToken.Token,
		}, tx)

		signupToken.UsageCount++

		err = tx.WithContext(ctx).Save(&signupToken).Error
		if err != nil {
			return model.User{}, "", err

		}
	} else {
		s.auditLogService.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"method": "open_signup",
		}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

func (s *UserSignUpService) SignUpInitialAdmin(ctx context.Context, signUpData dto.SignUpDto) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var userCount int64
	if err := tx.WithContext(ctx).Model(&model.User{}).
		Where("id != ?", staticApiKeyUserID).
		Count(&userCount).Error; err != nil {
		return model.User{}, "", err
	}
	if userCount != 0 {
		return model.User{}, "", &common.SetupAlreadyCompletedError{}
	}

	userToCreate := dto.UserCreateDto{
		FirstName:   signUpData.FirstName,
		LastName:    signUpData.LastName,
		DisplayName: strings.TrimSpace(signUpData.FirstName + " " + signUpData.LastName),
		Username:    signUpData.Username,
		Email:       signUpData.Email,
		IsAdmin:     true,
	}

	user, err := s.userService.createUserInternal(ctx, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	token, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return model.User{}, "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

func (s *UserSignUpService) ListSignupTokens(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]model.SignupToken, utils.PaginationResponse, error) {
	var tokens []model.SignupToken
	query := s.db.WithContext(ctx).Preload("UserGroups").Model(&model.SignupToken{})

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &tokens)
	return tokens, pagination, err
}

func (s *UserSignUpService) DeleteSignupToken(ctx context.Context, tokenID string) error {
	return s.db.WithContext(ctx).Delete(&model.SignupToken{}, "id = ?", tokenID).Error
}

func (s *UserSignUpService) CreateSignupToken(ctx context.Context, ttl time.Duration, usageLimit int, userGroupIDs []string) (model.SignupToken, error) {
	signupToken, err := NewSignupToken(ttl, usageLimit)
	if err != nil {
		return model.SignupToken{}, err
	}

	var userGroups []model.UserGroup
	err = s.db.WithContext(ctx).
		Where("id IN ?", userGroupIDs).
		Find(&userGroups).
		Error
	if err != nil {
		return model.SignupToken{}, err
	}
	signupToken.UserGroups = userGroups

	err = s.db.WithContext(ctx).Create(signupToken).Error
	if err != nil {
		return model.SignupToken{}, err
	}

	return *signupToken, nil
}

func NewSignupToken(ttl time.Duration, usageLimit int) (*model.SignupToken, error) {
	// Generate a random token
	randomString, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return nil, err
	}

	now := time.Now().Round(time.Second)
	token := &model.SignupToken{
		Token:      randomString,
		ExpiresAt:  datatype.DateTime(now.Add(ttl)),
		UsageLimit: usageLimit,
		UsageCount: 0,
	}

	return token, nil
}
