package service

import (
	"errors"
	"fmt"
	"github.com/stonith404/pocket-id/backend/internal/common"
	"github.com/stonith404/pocket-id/backend/internal/dto"
	"github.com/stonith404/pocket-id/backend/internal/model"
	"github.com/stonith404/pocket-id/backend/internal/model/types"
	"github.com/stonith404/pocket-id/backend/internal/utils"
	"github.com/stonith404/pocket-id/backend/internal/utils/email"
	"gorm.io/gorm"
	"log"
	"net/url"
	"strings"
	"time"
)

type UserService struct {
	db              *gorm.DB
	jwtService      *JwtService
	auditLogService *AuditLogService
	emailService    *EmailService
}

func NewUserService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, emailService *EmailService) *UserService {
	return &UserService{db: db, jwtService: jwtService, auditLogService: auditLogService, emailService: emailService}
}

func (s *UserService) ListUsers(searchTerm string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.User, utils.PaginationResponse, error) {
	var users []model.User
	query := s.db.Model(&model.User{})

	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where("email LIKE ? OR first_name LIKE ? OR username LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &users)
	return users, pagination, err
}

func (s *UserService) GetUser(userID string) (model.User, error) {
	var user model.User
	err := s.db.Preload("CustomClaims").Where("id = ?", userID).First(&user).Error
	return user, err
}

func (s *UserService) DeleteUser(userID string) error {
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	if user.LdapID != nil {
		return &common.LdapUserUpdateError{}
	}

	return s.db.Delete(&user).Error
}

func (s *UserService) CreateUser(input dto.UserCreateDto) (model.User, error) {
	user := model.User{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Username:  input.Username,
		IsAdmin:   input.IsAdmin,
		LdapID:    &input.LdapID,
	}
	if err := s.db.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.User{}, s.checkDuplicatedFields(user)
		}
		return model.User{}, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(userID string, updatedUser dto.UserCreateDto, updateOwnUser bool, allowLdapUpdate bool) (model.User, error) {
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return model.User{}, err
	}

	if user.LdapID != nil && !allowLdapUpdate {
		return model.User{}, &common.LdapUserUpdateError{}
	}

	user.FirstName = updatedUser.FirstName
	user.LastName = updatedUser.LastName
	user.Email = updatedUser.Email
	user.Username = updatedUser.Username
	if !updateOwnUser {
		user.IsAdmin = updatedUser.IsAdmin
	}

	if err := s.db.Save(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user, s.checkDuplicatedFields(user)
		}
		return user, err
	}

	return user, nil
}

func (s *UserService) RequestOneTimeAccessEmail(emailAddress, redirectPath string) error {
	var user model.User
	if err := s.db.Where("email = ?", emailAddress).First(&user).Error; err != nil {
		// Do not return error if user not found to prevent email enumeration
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	}

	oneTimeAccessToken, err := s.CreateOneTimeAccessToken(user.ID, time.Now().Add(time.Hour))
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/login/%s", common.EnvConfig.AppURL, oneTimeAccessToken)

	// Add redirect path to the link
	if strings.HasPrefix(redirectPath, "/") {
		encodedRedirectPath := url.QueryEscape(redirectPath)
		link = fmt.Sprintf("%s?redirect=%s", link, encodedRedirectPath)
	}

	go func() {
		err := SendEmail(s.emailService, email.Address{
			Name:  user.Username,
			Email: user.Email,
		}, OneTimeAccessTemplate, &OneTimeAccessTemplateData{
			Link: link,
		})
		if err != nil {
			log.Printf("Failed to send email to '%s': %v\n", user.Email, err)
		}
	}()

	return nil
}

func (s *UserService) CreateOneTimeAccessToken(userID string, expiresAt time.Time) (string, error) {
	randomString, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return "", err
	}

	oneTimeAccessToken := model.OneTimeAccessToken{
		UserID:    userID,
		ExpiresAt: datatype.DateTime(expiresAt),
		Token:     randomString,
	}

	if err := s.db.Create(&oneTimeAccessToken).Error; err != nil {
		return "", err
	}

	return oneTimeAccessToken.Token, nil
}

func (s *UserService) ExchangeOneTimeAccessToken(token string, ipAddress, userAgent string) (model.User, string, error) {
	var oneTimeAccessToken model.OneTimeAccessToken
	if err := s.db.Where("token = ? AND expires_at > ?", token, datatype.DateTime(time.Now())).Preload("User").First(&oneTimeAccessToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}
		return model.User{}, "", err
	}
	accessToken, err := s.jwtService.GenerateAccessToken(oneTimeAccessToken.User)
	if err != nil {
		return model.User{}, "", err
	}

	if err := s.db.Delete(&oneTimeAccessToken).Error; err != nil {
		return model.User{}, "", err
	}

	if ipAddress != "" && userAgent != "" {
		s.auditLogService.Create(model.AuditLogEventOneTimeAccessTokenSignIn, ipAddress, userAgent, oneTimeAccessToken.User.ID, model.AuditLogData{})
	}

	return oneTimeAccessToken.User, accessToken, nil
}

func (s *UserService) SetupInitialAdmin() (model.User, string, error) {
	var userCount int64
	if err := s.db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return model.User{}, "", err
	}
	if userCount > 1 {
		return model.User{}, "", &common.SetupAlreadyCompletedError{}
	}

	user := model.User{
		FirstName: "Admin",
		LastName:  "Admin",
		Username:  "admin",
		Email:     "admin@admin.com",
		IsAdmin:   true,
	}

	if err := s.db.Model(&model.User{}).Preload("Credentials").FirstOrCreate(&user).Error; err != nil {
		return model.User{}, "", err
	}

	if len(user.Credentials) > 0 {
		return model.User{}, "", &common.SetupAlreadyCompletedError{}
	}

	token, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

func (s *UserService) checkDuplicatedFields(user model.User) error {
	var existingUser model.User
	if s.db.Where("id != ? AND email = ?", user.ID, user.Email).First(&existingUser).Error == nil {
		return &common.AlreadyInUseError{Property: "email"}
	}

	if s.db.Where("id != ? AND username = ?", user.ID, user.Username).First(&existingUser).Error == nil {
		return &common.AlreadyInUseError{Property: "username"}
	}

	return nil
}
