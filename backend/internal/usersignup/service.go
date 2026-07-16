package usersignup

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodOneTimePassword identifies one-time-password authentication, used for the initial admin setup token
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodOneTimePassword = "otp"

type Service struct {
	db          *gorm.DB
	userCreator UserCreator
	signer      TokenService
	auditLog    AuditLogger
	appConfig   AppConfigProvider
}

func newService(deps Dependencies) *Service {
	return &Service{
		db:          deps.DB,
		userCreator: deps.UserCreator,
		signer:      deps.Signer,
		auditLog:    deps.AuditLog,
		appConfig:   deps.AppConfig,
	}
}

func (s *Service) SignUp(ctx context.Context, signupData signUpDto, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	tokenProvided := signupData.Token != ""

	config := s.appConfig.GetDbConfig()
	if config.AllowUserSignups.Value != "open" && !tokenProvided {
		return model.User{}, "", &common.OpenSignupDisabledError{}
	}

	var signupToken SignupToken
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

		if signupToken.HasEmailDomainRestriction() {
			email := ""
			if signupData.Email != nil {
				email = *signupData.Email
			}
			if !signupToken.EmailMatchesDomain(email) {
				return model.User{}, "", &common.EmailDomainNotAllowedError{Domain: *signupToken.EmailDomain}
			}
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
		EmailVerified: s.appConfig.GetDbConfig().EmailsVerified.IsTrue(),
	}

	user, err := s.userCreator.CreateUserInternal(ctx, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	accessToken, err := s.signer.GenerateAccessToken(user, "")
	if err != nil {
		return model.User{}, "", err
	}

	if tokenProvided {
		s.auditLog.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"signupToken": signupToken.Token,
		}, tx)

		signupToken.UsageCount++

		err = tx.WithContext(ctx).Save(&signupToken).Error
		if err != nil {
			return model.User{}, "", err
		}
	} else {
		s.auditLog.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"method": "open_signup",
		}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

func (s *Service) SignUpInitialAdmin(ctx context.Context, signUpData signUpDto) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	setupCompleted, err := s.isInitialAdminSetupCompleted(ctx, tx)
	if err != nil {
		return model.User{}, "", err
	}
	if setupCompleted {
		return model.User{}, "", &common.SetupNotAvailableError{}
	}

	userToCreate := dto.UserCreateDto{
		FirstName:   signUpData.FirstName,
		LastName:    signUpData.LastName,
		DisplayName: strings.TrimSpace(signUpData.FirstName + " " + signUpData.LastName),
		Username:    signUpData.Username,
		Email:       signUpData.Email,
		IsAdmin:     true,
	}

	user, err := s.userCreator.CreateUserInternal(ctx, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	token, err := s.signer.GenerateAccessToken(user, authenticationMethodOneTimePassword)
	if err != nil {
		return model.User{}, "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

func (s *Service) IsInitialAdminSetupCompleted(ctx context.Context) (bool, error) {
	return s.isInitialAdminSetupCompleted(ctx, s.db)
}

func (s *Service) isInitialAdminSetupCompleted(ctx context.Context, db *gorm.DB) (bool, error) {
	var userCount int64
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("id != ?", common.StaticApiKeyUserID).
		Count(&userCount).Error; err != nil {
		return false, err
	}

	return userCount != 0, nil
}

func (s *Service) ListSignupTokens(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]SignupToken, utils.PaginationResponse, error) {
	var tokens []SignupToken
	query := s.db.WithContext(ctx).Preload("UserGroups").Model(&SignupToken{})

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &tokens)
	return tokens, pagination, err
}

func (s *Service) DeleteSignupToken(ctx context.Context, tokenID string) error {
	return s.db.WithContext(ctx).Delete(&SignupToken{}, "id = ?", tokenID).Error
}

// GetSignupTokenInfo returns a signup token by its token string.
// It's used to expose the limited, public metadata (such as the required email domain) needed to render the signup form.
func (s *Service) GetSignupTokenInfo(ctx context.Context, token string) (SignupToken, error) {
	var signupToken SignupToken
	err := s.db.
		WithContext(ctx).
		Where("token = ?", token).
		First(&signupToken).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SignupToken{}, &common.TokenInvalidOrExpiredError{}
	} else if err != nil {
		return SignupToken{}, err
	}

	return signupToken, nil
}

func (s *Service) CreateSignupToken(ctx context.Context, ttl time.Duration, usageLimit int, userGroupIDs []string, emailDomain *string) (SignupToken, error) {
	signupToken, err := newSignupToken(ttl, usageLimit, emailDomain)
	if err != nil {
		return SignupToken{}, err
	}

	var userGroups []model.UserGroup
	err = s.db.WithContext(ctx).
		Where("id IN ?", userGroupIDs).
		Find(&userGroups).
		Error
	if err != nil {
		return SignupToken{}, err
	}
	signupToken.UserGroups = userGroups

	err = s.db.WithContext(ctx).Create(signupToken).Error
	if err != nil {
		return SignupToken{}, err
	}

	return *signupToken, nil
}

func newSignupToken(ttl time.Duration, usageLimit int, emailDomain *string) (*SignupToken, error) {
	// Generate a random token
	randomString, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return nil, err
	}

	now := time.Now().Round(time.Second)
	token := &SignupToken{
		Token:       randomString,
		ExpiresAt:   datatype.DateTime(now.Add(ttl)),
		UsageLimit:  usageLimit,
		UsageCount:  0,
		EmailDomain: emailDomain,
	}

	return token, nil
}
