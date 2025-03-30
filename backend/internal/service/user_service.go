package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
)

type UserService struct {
	db               *gorm.DB
	jwtService       *JwtService
	auditLogService  *AuditLogService
	emailService     *EmailService
	appConfigService *AppConfigService
}

func NewUserService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, emailService *EmailService, appConfigService *AppConfigService) *UserService {
	return &UserService{db: db, jwtService: jwtService, auditLogService: auditLogService, emailService: emailService, appConfigService: appConfigService}
}

func (s *UserService) ListUsers(ctx context.Context, searchTerm string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.User, utils.PaginationResponse, error) {
	var users []model.User
	query := s.db.WithContext(ctx).Model(&model.User{})

	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where("email LIKE ? OR first_name LIKE ? OR username LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &users)
	return users, pagination, err
}

func (s *UserService) GetUser(ctx context.Context, userID string) (model.User, error) {
	return s.getUserInternal(ctx, userID, s.db)
}

func (s *UserService) getUserInternal(ctx context.Context, userID string, tx *gorm.DB) (model.User, error) {
	var user model.User
	err := tx.
		WithContext(ctx).
		Preload("UserGroups").
		Preload("CustomClaims").
		Where("id = ?", userID).
		First(&user).
		Error
	return user, err
}

func (s *UserService) GetProfilePicture(ctx context.Context, userID string) (io.Reader, int64, error) {
	// Validate the user ID to prevent directory traversal
	if err := uuid.Validate(userID); err != nil {
		return nil, 0, &common.InvalidUUIDError{}
	}

	profilePicturePath := common.EnvConfig.UploadPath + "/profile-pictures/" + userID + ".png"
	file, err := os.Open(profilePicturePath)
	if err == nil {
		// Get the file size
		fileInfo, err := file.Stat()
		if err != nil {
			return nil, 0, err
		}
		return file, fileInfo.Size(), nil
	}

	// If the file does not exist, return the default profile picture
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	defaultPicture, err := profilepicture.CreateDefaultProfilePicture(user.FirstName, user.LastName)
	if err != nil {
		return nil, 0, err
	}

	return defaultPicture, int64(defaultPicture.Len()), nil
}

func (s *UserService) GetUserGroups(ctx context.Context, userID string) ([]model.UserGroup, error) {
	var user model.User
	err := s.db.
		WithContext(ctx).
		Preload("UserGroups").
		Where("id = ?", userID).
		First(&user).
		Error
	if err != nil {
		return nil, err
	}
	return user.UserGroups, nil
}

func (s *UserService) UpdateProfilePicture(userID string, file io.Reader) error {
	// Validate the user ID to prevent directory traversal
	err := uuid.Validate(userID)
	if err != nil {
		return &common.InvalidUUIDError{}
	}

	// Convert the image to a smaller square image
	profilePicture, err := profilepicture.CreateProfilePicture(file)
	if err != nil {
		return err
	}

	// Ensure the directory exists
	profilePictureDir := common.EnvConfig.UploadPath + "/profile-pictures"
	err = os.MkdirAll(profilePictureDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Create the profile picture file
	err = utils.SaveFileStream(profilePicture, profilePictureDir+"/"+userID+".png")
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID string, allowLdapDelete bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.deleteUserInternal(ctx, userID, allowLdapDelete, tx)
	})
}

func (s *UserService) deleteUserInternal(ctx context.Context, userID string, allowLdapDelete bool, tx *gorm.DB) error {
	var user model.User

	err := tx.
		WithContext(ctx).
		Where("id = ?", userID).
		First(&user).
		Error
	if err != nil {
		return err
	}

	// Disallow deleting the user if it is an LDAP user and LDAP is enabled
	if !allowLdapDelete && user.LdapID != nil && s.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return &common.LdapUserUpdateError{}
	}

	// Delete the profile picture
	profilePicturePath := common.EnvConfig.UploadPath + "/profile-pictures/" + userID + ".png"
	err = os.Remove(profilePicturePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return tx.WithContext(ctx).Delete(&user).Error
}

func (s *UserService) CreateUser(ctx context.Context, input dto.UserCreateDto) (model.User, error) {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	user, err := s.createUserInternal(ctx, input, tx)
	if err != nil {
		return model.User{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *UserService) createUserInternal(ctx context.Context, input dto.UserCreateDto, tx *gorm.DB) (model.User, error) {
	user := model.User{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Username:  input.Username,
		IsAdmin:   input.IsAdmin,
		Locale:    input.Locale,
	}
	if input.LdapID != "" {
		user.LdapID = &input.LdapID
	}

	err := tx.WithContext(ctx).Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		tx.Rollback()

		// If we are here, the transaction is already aborted due to an error, so we pass s.db
		err = s.checkDuplicatedFields(ctx, user, s.db)
		return model.User{}, err
	} else if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID string, updatedUser dto.UserCreateDto, updateOwnUser bool, allowLdapUpdate bool) (model.User, error) {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	user, err := s.updateUserInternal(ctx, userID, updatedUser, updateOwnUser, allowLdapUpdate, tx)
	if err != nil {
		return model.User{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *UserService) updateUserInternal(ctx context.Context, userID string, updatedUser dto.UserCreateDto, updateOwnUser bool, allowLdapUpdate bool, tx *gorm.DB) (model.User, error) {
	var user model.User
	err := tx.
		WithContext(ctx).
		Where("id = ?", userID).
		First(&user).
		Error
	if err != nil {
		return model.User{}, err
	}

	// Disallow updating the user if it is an LDAP group and LDAP is enabled
	if !allowLdapUpdate && user.LdapID != nil && s.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return model.User{}, &common.LdapUserUpdateError{}
	}

	user.FirstName = updatedUser.FirstName
	user.LastName = updatedUser.LastName
	user.Email = updatedUser.Email
	user.Username = updatedUser.Username
	user.Locale = updatedUser.Locale
	if !updateOwnUser {
		user.IsAdmin = updatedUser.IsAdmin
	}

	err = tx.
		WithContext(ctx).
		Save(&user).
		Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		tx.Rollback()

		// If we are here, the transaction is already aborted due to an error, so we pass s.db
		err = s.checkDuplicatedFields(ctx, user, s.db)
		return user, err
	} else if err != nil {
		return user, err
	}

	return user, nil
}

func (s *UserService) RequestOneTimeAccessEmail(ctx context.Context, emailAddress, redirectPath string) error {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	isDisabled := !s.appConfigService.DbConfig.EmailOneTimeAccessEnabled.IsTrue()
	if isDisabled {
		return &common.OneTimeAccessDisabledError{}
	}

	var user model.User
	err := tx.
		WithContext(ctx).
		Where("email = ?", emailAddress).
		First(&user).
		Error
	if err != nil {
		// Do not return error if user not found to prevent email enumeration
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	}

	oneTimeAccessToken, err := s.createOneTimeAccessTokenInternal(ctx, user.ID, time.Now().Add(15*time.Minute), tx)
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	// We use a background context here as this is running in a goroutine
	//nolint:contextcheck
	go func() {
		innerCtx := context.Background()

		link := common.EnvConfig.AppURL + "/lc"
		linkWithCode := link + "/" + oneTimeAccessToken

		// Add redirect path to the link
		if strings.HasPrefix(redirectPath, "/") {
			encodedRedirectPath := url.QueryEscape(redirectPath)
			linkWithCode = linkWithCode + "?redirect=" + encodedRedirectPath
		}

		errInternal := SendEmail(innerCtx, s.emailService, email.Address{
			Name:  user.Username,
			Email: user.Email,
		}, OneTimeAccessTemplate, &OneTimeAccessTemplateData{
			Code:              oneTimeAccessToken,
			LoginLink:         link,
			LoginLinkWithCode: linkWithCode,
		})
		if errInternal != nil {
			log.Printf("Failed to send email to '%s': %v\n", user.Email, errInternal)
		}
	}()

	return nil
}

func (s *UserService) CreateOneTimeAccessToken(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	return s.createOneTimeAccessTokenInternal(ctx, userID, expiresAt, s.db)
}

func (s *UserService) createOneTimeAccessTokenInternal(ctx context.Context, userID string, expiresAt time.Time, tx *gorm.DB) (string, error) {
	// If expires at is less than 15 minutes, use an 6 character token instead of 16
	tokenLength := 16
	if time.Until(expiresAt) <= 15*time.Minute {
		tokenLength = 6
	}

	randomString, err := utils.GenerateRandomAlphanumericString(tokenLength)
	if err != nil {
		return "", err
	}

	oneTimeAccessToken := model.OneTimeAccessToken{
		UserID:    userID,
		ExpiresAt: datatype.DateTime(expiresAt),
		Token:     randomString,
	}

	if err := tx.WithContext(ctx).Create(&oneTimeAccessToken).Error; err != nil {
		return "", err
	}

	return oneTimeAccessToken.Token, nil
}

func (s *UserService) ExchangeOneTimeAccessToken(ctx context.Context, token string, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	var oneTimeAccessToken model.OneTimeAccessToken
	err := tx.
		WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, datatype.DateTime(time.Now())).Preload("User").
		First(&oneTimeAccessToken).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}
		return model.User{}, "", err
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

	if ipAddress != "" && userAgent != "" {
		s.auditLogService.Create(model.AuditLogEventOneTimeAccessTokenSignIn, ipAddress, userAgent, oneTimeAccessToken.User.ID, model.AuditLogData{}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return oneTimeAccessToken.User, accessToken, nil
}

func (s *UserService) UpdateUserGroups(ctx context.Context, id string, userGroupIds []string) (user model.User, err error) {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	user, err = s.getUserInternal(ctx, id, tx)
	if err != nil {
		return model.User{}, err
	}

	// Fetch the groups based on userGroupIds
	var groups []model.UserGroup
	if len(userGroupIds) > 0 {
		err = tx.
			WithContext(ctx).
			Where("id IN (?)", userGroupIds).
			Find(&groups).
			Error
		if err != nil {
			return model.User{}, err
		}
	}

	// Replace the current groups with the new set of groups
	err = tx.
		WithContext(ctx).
		Model(&user).
		Association("UserGroups").
		Replace(groups)
	if err != nil {
		return model.User{}, err
	}

	// Save the updated user
	err = tx.WithContext(ctx).Save(&user).Error
	if err != nil {
		return model.User{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *UserService) SetupInitialAdmin(ctx context.Context) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		// This is a no-op if the transaction has been committed already
		tx.Rollback()
	}()

	var userCount int64
	if err := tx.WithContext(ctx).Model(&model.User{}).Count(&userCount).Error; err != nil {
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

	if err := tx.WithContext(ctx).Model(&model.User{}).Preload("Credentials").FirstOrCreate(&user).Error; err != nil {
		return model.User{}, "", err
	}

	if len(user.Credentials) > 0 {
		return model.User{}, "", &common.SetupAlreadyCompletedError{}
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

func (s *UserService) checkDuplicatedFields(ctx context.Context, user model.User, tx *gorm.DB) error {
	var result struct {
		Found bool
	}
	err := tx.
		WithContext(ctx).
		Raw(`SELECT EXISTS(SELECT 1 FROM users WHERE id != ? AND email = ?) AS found`, user.ID, user.Email).
		First(&result).
		Error
	if err != nil {
		return err
	}
	if result.Found {
		return &common.AlreadyInUseError{Property: "email"}
	}

	err = tx.
		WithContext(ctx).
		Raw(`SELECT EXISTS(SELECT 1 FROM users WHERE id != ? AND username = ?) AS found`, user.ID, user.Username).
		First(&result).
		Error
	if err != nil {
		return err
	}
	if result.Found {
		return &common.AlreadyInUseError{Property: "username"}
	}

	return nil
}

// ResetProfilePicture deletes a user's custom profile picture
func (s *UserService) ResetProfilePicture(userID string) error {
	// Validate the user ID to prevent directory traversal
	if err := uuid.Validate(userID); err != nil {
		return &common.InvalidUUIDError{}
	}

	// Build path to profile picture
	profilePicturePath := common.EnvConfig.UploadPath + "/profile-pictures/" + userID + ".png"

	// Check if file exists and delete it
	if _, err := os.Stat(profilePicturePath); err == nil {
		if err := os.Remove(profilePicturePath); err != nil {
			return fmt.Errorf("failed to delete profile picture: %w", err)
		}
	} else if !os.IsNotExist(err) {
		// If any error other than "file not exists"
		return fmt.Errorf("failed to check if profile picture exists: %w", err)
	}
	// It's okay if the file doesn't exist - just means there's no custom picture to delete

	return nil
}
