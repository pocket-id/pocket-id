package service

import (
	"context"
	"errors"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type UserGroupService struct {
	db          *gorm.DB
	scimService *ScimService
}

func NewUserGroupService(db *gorm.DB, scimService *ScimService) *UserGroupService {
	return &UserGroupService{db: db, scimService: scimService}
}

func (s *UserGroupService) List(ctx context.Context, name string, listRequestOptions utils.ListRequestOptions) (groups []model.UserGroup, response utils.PaginationResponse, err error) {
	query := s.db.
		WithContext(ctx).
		Preload("CustomClaims").
		Model(&model.UserGroup{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// As userCount is not a column we need to manually sort it
	if listRequestOptions.Sort.Column == "userCount" && utils.IsValidSortDirection(listRequestOptions.Sort.Direction) {
		query = query.Select("user_groups.*, COUNT(user_groups_users.user_id)").
			Joins("LEFT JOIN user_groups_users ON user_groups.id = user_groups_users.user_group_id").
			Group("user_groups.id").
			Order("COUNT(user_groups_users.user_id) " + listRequestOptions.Sort.Direction)
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &groups)
	return groups, response, err
}

func (s *UserGroupService) Get(ctx context.Context, id string) (group model.UserGroup, err error) {
	return s.getInternal(ctx, id, s.db)
}

func (s *UserGroupService) getInternal(ctx context.Context, id string, tx *gorm.DB) (group model.UserGroup, err error) {
	err = tx.
		WithContext(ctx).
		Where("id = ?", id).
		Preload("CustomClaims").
		Preload("Users").
		Preload("AllowedOidcClients").
		First(&group).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.UserGroup{}, apperror.NotFound("User group")
	}
	return group, err
}

func (s *UserGroupService) Delete(ctx context.Context, cfg *appconfig.AppConfigModel, id string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var group model.UserGroup
	err := tx.
		WithContext(ctx).
		Where("id = ?", id).
		First(&group).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.NotFound("User group")
	}
	if err != nil {
		return err
	}

	// Disallow deleting the group if it is an LDAP group and LDAP is enabled
	if group.LdapID != nil && cfg.LdapEnabled.IsTrue() {
		return apperror.LdapUserGroupUpdate()
	}

	err = tx.
		WithContext(ctx).
		Delete(&group).
		Error
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	if s.scimService != nil {
		s.scimService.ScheduleSync()
	}

	return nil
}

func (s *UserGroupService) Create(ctx context.Context, input dto.UserGroupCreateDto) (group model.UserGroup, err error) {
	return s.CreateInternal(ctx, input, s.db)
}

// CreateInternal creates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (s *UserGroupService) CreateInternal(ctx context.Context, input dto.UserGroupCreateDto, tx *gorm.DB) (model.UserGroup, error) {
	group := model.UserGroup{
		FriendlyName: input.FriendlyName,
		Name:         input.Name,
	}

	if input.LdapID != "" {
		group.LdapID = &input.LdapID
	}

	err := tx.
		WithContext(ctx).
		Preload("Users").
		Create(&group).
		Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.UserGroup{}, apperror.AlreadyInUse("name")
	} else if err != nil {
		return model.UserGroup{}, err
	}

	if s.scimService != nil {
		s.scimService.ScheduleSync()
	}

	return group, nil
}

func (s *UserGroupService) Update(ctx context.Context, cfg *appconfig.AppConfigModel, id string, input dto.UserGroupCreateDto) (group model.UserGroup, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	group, err = s.updateInternal(ctx, id, input, false, tx, cfg)
	if err != nil {
		return model.UserGroup{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.UserGroup{}, err
	}

	return group, nil
}

// UpdateInternal updates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (s *UserGroupService) UpdateInternal(ctx context.Context, cfg *appconfig.AppConfigModel, id string, input dto.UserGroupCreateDto, isLdapSync bool, tx *gorm.DB) (model.UserGroup, error) {
	return s.updateInternal(ctx, id, input, isLdapSync, tx, cfg)
}

func (s *UserGroupService) updateInternal(ctx context.Context, id string, input dto.UserGroupCreateDto, isLdapSync bool, tx *gorm.DB, cfg *appconfig.AppConfigModel) (group model.UserGroup, err error) {
	group, err = s.getInternal(ctx, id, tx)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Disallow updating the group if it is an LDAP group and LDAP is enabled
	if !isLdapSync && group.LdapID != nil {
		if cfg.LdapEnabled.IsTrue() {
			return model.UserGroup{}, apperror.LdapUserGroupUpdate()
		}
	}

	group.Name = input.Name
	group.FriendlyName = input.FriendlyName
	group.UpdatedAt = new(datatype.DateTime(time.Now()))

	err = tx.
		WithContext(ctx).
		Preload("Users").
		Save(&group).
		Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.UserGroup{}, apperror.AlreadyInUse("name")
	} else if err != nil {
		return model.UserGroup{}, err
	}

	if s.scimService != nil {
		s.scimService.ScheduleSync()
	}

	return group, nil
}

func (s *UserGroupService) UpdateUsers(ctx context.Context, id string, userIds []string) (group model.UserGroup, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	group, err = s.UpdateUsersInternal(ctx, id, userIds, tx)
	if err != nil {
		return model.UserGroup{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.UserGroup{}, err
	}

	return group, nil
}

// UpdateUsersInternal replaces the members of a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (s *UserGroupService) UpdateUsersInternal(ctx context.Context, id string, userIds []string, tx *gorm.DB) (model.UserGroup, error) {
	group, err := s.getInternal(ctx, id, tx)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Fetch the users based on the userIds
	var users []model.User
	if len(userIds) > 0 {
		err = tx.
			WithContext(ctx).
			Where("id IN (?)", userIds).
			Find(&users).
			Error
		if err != nil {
			return model.UserGroup{}, err
		}
	}

	// Replace the current users with the new set of users
	err = tx.
		WithContext(ctx).
		Model(&group).
		Association("Users").
		Replace(users)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Save the updated group
	group.UpdatedAt = new(datatype.DateTime(time.Now()))

	err = tx.
		WithContext(ctx).
		Save(&group).
		Error
	if err != nil {
		return model.UserGroup{}, err
	}

	if s.scimService != nil {
		s.scimService.ScheduleSync()
	}

	return group, nil
}

func (s *UserGroupService) GetUserCountOfGroup(ctx context.Context, id string) (int64, error) {
	// We only perform select queries here, so we can rollback in all cases
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var group model.UserGroup
	err := tx.
		WithContext(ctx).
		Preload("Users").
		Where("id = ?", id).
		First(&group).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, apperror.NotFound("User group")
	}
	if err != nil {
		return 0, err
	}
	count := tx.
		WithContext(ctx).
		Model(&group).
		Association("Users").
		Count()
	return count, nil
}

func (s *UserGroupService) UpdateAllowedOidcClient(ctx context.Context, id string, input dto.UserGroupUpdateAllowedOidcClientsDto) (group model.UserGroup, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	group, err = s.getInternal(ctx, id, tx)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Fetch the clients based on the client IDs
	var clients []model.OidcClient
	if len(input.OidcClientIDs) > 0 {
		err = tx.
			WithContext(ctx).
			Where("id IN (?)", input.OidcClientIDs).
			Find(&clients).
			Error
		if err != nil {
			return model.UserGroup{}, err
		}
	}

	// Replace the current clients with the new set of clients
	err = tx.
		WithContext(ctx).
		Model(&group).
		Association("AllowedOidcClients").
		Replace(clients)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Save the updated group
	err = tx.
		WithContext(ctx).
		Save(&group).
		Error
	if err != nil {
		return model.UserGroup{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.UserGroup{}, err
	}

	if s.scimService != nil {
		s.scimService.ScheduleSync()
	}

	return group, nil
}
