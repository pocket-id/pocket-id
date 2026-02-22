package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
)

type UserService struct {
	db                 *gorm.DB
	jwtService         *JwtService
	auditLogService    *AuditLogService
	emailService       *EmailService
	appConfigService   *AppConfigService
	customClaimService *CustomClaimService
	appImagesService   *AppImagesService
	scimService        *ScimService
	fileStorage        storage.FileStorage
}

func NewUserService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, emailService *EmailService, appConfigService *AppConfigService, customClaimService *CustomClaimService, appImagesService *AppImagesService, scimService *ScimService, fileStorage storage.FileStorage) *UserService {
	return &UserService{
		db:                 db,
		jwtService:         jwtService,
		auditLogService:    auditLogService,
		emailService:       emailService,
		appConfigService:   appConfigService,
		customClaimService: customClaimService,
		appImagesService:   appImagesService,
		scimService:        scimService,
		fileStorage:        fileStorage,
	}
}

func (s *UserService) ListUsers(ctx context.Context, searchTerm string, listRequestOptions utils.ListRequestOptions) ([]model.User, utils.PaginationResponse, error) {
	var users []model.User
	query := s.db.WithContext(ctx).
		Model(&model.User{}).
		Preload("UserGroups").
		Preload("CustomClaims")

	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where(
			"email LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR username LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	pagination, err := utils.PaginateFilterAndSort(listRequestOptions, query, &users)

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

func (s *UserService) GetProfilePicture(ctx context.Context, userID string) (io.ReadCloser, int64, error) {
	// Validate the user ID to prevent directory traversal
	if err := uuid.Validate(userID); err != nil {
		return nil, 0, &common.InvalidUUIDError{}
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	profilePicturePath := path.Join("profile-pictures", userID+".png")

	// Try custom profile picture
	file, size, err := s.fileStorage.Open(ctx, profilePicturePath)
	if err == nil {
		return file, size, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, 0, err
	}

	// Try default global profile picture
	if s.appImagesService.IsDefaultProfilePictureSet() {
		reader, size, _, err := s.appImagesService.GetImage(ctx, "default-profile-picture")
		if err == nil {
			return reader, size, nil
		}
		if !errors.Is(err, &common.ImageNotFoundError{}) {
			return nil, 0, err
		}
	}

	// Try cached default for initials
	defaultPicturePath := path.Join("profile-pictures", "defaults", user.Initials()+".png")
	file, size, err = s.fileStorage.Open(ctx, defaultPicturePath)
	if err == nil {
		return file, size, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, 0, err
	}

	// Create and return generated default with initials
	defaultPicture, err := profilepicture.CreateDefaultProfilePicture(user.Initials())
	if err != nil {
		return nil, 0, err
	}

	// Save the default picture for future use (in a goroutine to avoid blocking)
	defaultPictureBytes := defaultPicture.Bytes()
	//nolint:contextcheck
	go func() {
		// Use bytes.NewReader because we need an io.ReadSeeker
		rErr := s.fileStorage.Save(context.Background(), defaultPicturePath, bytes.NewReader(defaultPictureBytes))
		if rErr != nil {
			slog.Error("Failed to cache default profile picture", slog.String("initials", user.Initials()), slog.Any("error", rErr))
		}
	}()

	return io.NopCloser(bytes.NewReader(defaultPictureBytes)), int64(len(defaultPictureBytes)), nil
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

func (s *UserService) UpdateProfilePicture(ctx context.Context, userID string, file io.ReadSeeker) error {
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

	profilePicturePath := path.Join("profile-pictures", userID+".png")
	err = s.fileStorage.Save(ctx, profilePicturePath, profilePicture)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID string, allowLdapDelete bool) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.deleteUserInternal(ctx, tx, userID, allowLdapDelete)
	})
	if err != nil {
		return fmt.Errorf("failed to delete user '%s': %w", userID, err)
	}

	// Storage operations must be executed outside of a transaction
	profilePicturePath := path.Join("profile-pictures", userID+".png")
	err = s.fileStorage.Delete(ctx, profilePicturePath)
	if err != nil && !storage.IsNotExist(err) {
		return fmt.Errorf("failed to delete profile picture for user '%s': %w", userID, err)
	}

	return nil
}

func (s *UserService) deleteUserInternal(ctx context.Context, tx *gorm.DB, userID string, allowLdapDelete bool) error {
	var user model.User

	err := tx.
		WithContext(ctx).
		Where("id = ?", userID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&user).
		Error
	if err != nil {
		return fmt.Errorf("failed to load user to delete: %w", err)
	}

	// Disallow deleting the user if it is an LDAP user, LDAP is enabled, and the user is not disabled
	if !allowLdapDelete && !user.Disabled && user.LdapID != nil && s.appConfigService.GetDbConfig().LdapEnabled.IsTrue() {
		return &common.LdapUserUpdateError{}
	}

	err = tx.WithContext(ctx).Delete(&user).Error
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.scimService.ScheduleSync()
	return nil
}

func (s *UserService) CreateUser(ctx context.Context, input dto.UserCreateDto) (model.User, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := s.createUserInternal(ctx, input, false, tx)
	if err != nil {
		return model.User{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *UserService) createUserInternal(ctx context.Context, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error) {
	if s.appConfigService.GetDbConfig().RequireUserEmail.IsTrue() && input.Email == nil {
		return model.User{}, &common.UserEmailNotSetError{}
	}

	var userGroups []model.UserGroup
	if len(input.UserGroupIds) > 0 {
		err := tx.
			WithContext(ctx).
			Where("id IN ?", input.UserGroupIds).
			Find(&userGroups).
			Error
		if err != nil {
			return model.User{}, err
		}
	}

	user := model.User{
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		DisplayName:   input.DisplayName,
		Email:         input.Email,
		EmailVerified: input.EmailVerified,
		Username:      input.Username,
		IsAdmin:       input.IsAdmin,
		Locale:        input.Locale,
		Disabled:      input.Disabled,
		UserGroups:    userGroups,
	}
	if input.LdapID != "" {
		user.LdapID = &input.LdapID
	}

	err := tx.WithContext(ctx).Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// Do not follow this path if we're using LDAP, as we don't want to roll-back the transaction here
		if !isLdapSync {
			tx.Rollback()
			// If we are here, the transaction is already aborted due to an error, so we pass s.db
			err = s.checkDuplicatedFields(ctx, user, s.db)
		} else {
			err = s.checkDuplicatedFields(ctx, user, tx)
		}

		return model.User{}, err
	} else if err != nil {
		return model.User{}, err
	}

	// Apply default groups and claims for new non-LDAP users
	if !isLdapSync {
		if len(input.UserGroupIds) == 0 {
			if err := s.applyDefaultGroups(ctx, &user, tx); err != nil {
				return model.User{}, err
			}
		}

		if err := s.applyDefaultCustomClaims(ctx, &user, tx); err != nil {
			return model.User{}, err
		}
	}

	s.scimService.ScheduleSync()
	return user, nil
}

func (s *UserService) applyDefaultGroups(ctx context.Context, user *model.User, tx *gorm.DB) error {
	config := s.appConfigService.GetDbConfig()

	var groupIDs []string
	v := config.SignupDefaultUserGroupIDs.Value
	if v != "" && v != "[]" {
		err := json.Unmarshal([]byte(v), &groupIDs)
		if err != nil {
			return fmt.Errorf("invalid SignupDefaultUserGroupIDs JSON: %w", err)
		}
		if len(groupIDs) > 0 {
			var groups []model.UserGroup
			err = tx.WithContext(ctx).
				Where("id IN ?", groupIDs).
				Find(&groups).
				Error
			if err != nil {
				return fmt.Errorf("failed to find default user groups: %w", err)
			}

			err = tx.WithContext(ctx).
				Model(user).
				Association("UserGroups").
				Replace(groups)
			if err != nil {
				return fmt.Errorf("failed to associate default user groups: %w", err)
			}
		}
	}
	return nil
}

func (s *UserService) applyDefaultCustomClaims(ctx context.Context, user *model.User, tx *gorm.DB) error {
	config := s.appConfigService.GetDbConfig()

	var claims []dto.CustomClaimCreateDto
	v := config.SignupDefaultCustomClaims.Value
	if v != "" && v != "[]" {
		err := json.Unmarshal([]byte(v), &claims)
		if err != nil {
			return fmt.Errorf("invalid SignupDefaultCustomClaims JSON: %w", err)
		}
		if len(claims) > 0 {
			_, err = s.customClaimService.updateCustomClaimsInternal(ctx, UserID, user.ID, claims, tx)
			if err != nil {
				return fmt.Errorf("failed to apply default custom claims: %w", err)
			}
		}
	}

	return nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID string, updatedUser dto.UserCreateDto, updateOwnUser bool, isLdapSync bool) (model.User, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := s.updateUserInternal(ctx, userID, updatedUser, updateOwnUser, isLdapSync, tx)
	if err != nil {
		return model.User{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s *UserService) updateUserInternal(ctx context.Context, userID string, updatedUser dto.UserCreateDto, updateOwnUser bool, isLdapSync bool, tx *gorm.DB) (model.User, error) {
	if s.appConfigService.GetDbConfig().RequireUserEmail.IsTrue() && updatedUser.Email == nil {
		return model.User{}, &common.UserEmailNotSetError{}
	}

	var user model.User
	err := tx.
		WithContext(ctx).
		Where("id = ?", userID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&user).
		Error
	if err != nil {
		return model.User{}, err
	}

	// Check if this is an LDAP user and LDAP is enabled
	isLdapUser := user.LdapID != nil && s.appConfigService.GetDbConfig().LdapEnabled.IsTrue()
	allowOwnAccountEdit := s.appConfigService.GetDbConfig().AllowOwnAccountEdit.IsTrue()

	if !isLdapSync && (isLdapUser || (!allowOwnAccountEdit && updateOwnUser)) {
		// Restricted update: Only locale can be changed when:
		// - User is from LDAP, OR
		// - User is editing their own account but global setting disallows self-editing
		// (Exception: LDAP sync operations can update everything)
		user.Locale = updatedUser.Locale
	} else {
		// Full update: Allow updating all personal fields
		user.FirstName = updatedUser.FirstName
		user.LastName = updatedUser.LastName
		user.DisplayName = updatedUser.DisplayName
		user.Username = updatedUser.Username
		user.Locale = updatedUser.Locale

		if (user.Email == nil && updatedUser.Email != nil) || (user.Email != nil && updatedUser.Email != nil && *user.Email != *updatedUser.Email) {
			// Email has changed, reset email verification status
			user.EmailVerified = s.appConfigService.GetDbConfig().EmailsVerified.IsTrue()
		}

		user.Email = updatedUser.Email

		// Admin-only fields: Only allow updates when not updating own account
		if !updateOwnUser {
			user.IsAdmin = updatedUser.IsAdmin
			user.EmailVerified = updatedUser.EmailVerified
			user.Disabled = updatedUser.Disabled
		}
	}

	user.UpdatedAt = new(datatype.DateTime(time.Now()))

	err = tx.
		WithContext(ctx).
		Save(&user).
		Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// Do not follow this path if we're using LDAP, as we don't want to roll-back the transaction here
		if !isLdapSync {
			tx.Rollback()
			// If we are here, the transaction is already aborted due to an error, so we pass s.db
			err = s.checkDuplicatedFields(ctx, user, s.db)
		} else {
			err = s.checkDuplicatedFields(ctx, user, tx)
		}

		return user, err
	} else if err != nil {
		return user, err
	}

	s.scimService.ScheduleSync()
	return user, nil
}

func (s *UserService) UpdateUserGroups(ctx context.Context, id string, userGroupIds []string) (user model.User, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err = s.getUserInternal(ctx, id, tx)
	if err != nil {
		return model.User{}, err
	}

	// Fetch the groups based on userGroupIds
	var groups []model.UserGroup
	if len(userGroupIds) > 0 {
		err := tx.
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

	// Update the UpdatedAt field for all affected groups
	now := time.Now()
	for _, group := range groups {
		group.UpdatedAt = new(datatype.DateTime(now))
		err = tx.WithContext(ctx).Save(&group).Error
		if err != nil {
			return model.User{}, err
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}

	s.scimService.ScheduleSync()
	return user, nil
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
func (s *UserService) ResetProfilePicture(ctx context.Context, userID string) error {
	// Validate the user ID to prevent directory traversal
	if err := uuid.Validate(userID); err != nil {
		return &common.InvalidUUIDError{}
	}

	profilePicturePath := path.Join("profile-pictures", userID+".png")
	if err := s.fileStorage.Delete(ctx, profilePicturePath); err != nil {
		return fmt.Errorf("failed to delete profile picture: %w", err)
	}
	return nil
}

func (s *UserService) disableUserInternal(ctx context.Context, tx *gorm.DB, userID string) error {
	err := tx.
		WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("disabled", true).
		Error

	if err != nil {
		return err
	}

	s.scimService.ScheduleSync()
	return nil
}

func (s *UserService) SendEmailVerification(ctx context.Context, userID string) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	if user.Email == nil {
		return &common.UserEmailNotSetError{}
	}

	randomToken, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return err
	}

	expiration := time.Now().Add(24 * time.Hour)
	emailVerificationToken := &model.EmailVerificationToken{
		UserID:    user.ID,
		Token:     randomToken,
		ExpiresAt: datatype.DateTime(expiration),
	}

	err = s.db.WithContext(ctx).Create(emailVerificationToken).Error
	if err != nil {
		return err
	}

	return SendEmail(ctx, s.emailService, email.Address{
		Name:  user.FullName(),
		Email: *user.Email,
	}, EmailVerificationTemplate, &EmailVerificationTemplateData{
		UserFullName:     user.FullName(),
		VerificationLink: common.EnvConfig.AppURL + "/verify-email?token=" + emailVerificationToken.Token,
	})
}

func (s *UserService) VerifyEmail(ctx context.Context, userID string, token string) error {
	tx := s.db.Begin()
	defer tx.Rollback()

	var emailVerificationToken model.EmailVerificationToken
	err := tx.WithContext(ctx).Where("token = ? AND user_id = ? AND expires_at > ?",
		token, userID, datatype.DateTime(time.Now())).First(&emailVerificationToken).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &common.InvalidEmailVerificationTokenError{}
	} else if err != nil {
		return err
	}

	user, err := s.getUserInternal(ctx, emailVerificationToken.UserID, tx)
	if err != nil {
		return err
	}

	user.EmailVerified = true
	user.UpdatedAt = new(datatype.DateTime(time.Now()))
	err = tx.WithContext(ctx).Save(&user).Error
	if err != nil {
		return err
	}

	err = tx.WithContext(ctx).Delete(&emailVerificationToken).Error
	if err != nil {
		return err
	}

	return tx.Commit().Error
}
