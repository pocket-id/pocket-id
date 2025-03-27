package service

import (
	"errors"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type UserGroupService struct {
	db               *gorm.DB
	appConfigService *AppConfigService
}

func NewUserGroupService(db *gorm.DB, appConfigService *AppConfigService) *UserGroupService {
	return &UserGroupService{db: db, appConfigService: appConfigService}
}

func (s *UserGroupService) List(name string, sortedPaginationRequest utils.SortedPaginationRequest) (groups []model.UserGroup, response utils.PaginationResponse, err error) {
	query := s.db.Preload("CustomClaims").Model(&model.UserGroup{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// As userCount is not a column we need to manually sort it
	isValidSortDirection := sortedPaginationRequest.Sort.Direction == "asc" || sortedPaginationRequest.Sort.Direction == "desc"
	if sortedPaginationRequest.Sort.Column == "userCount" && isValidSortDirection {
		query = query.Select("user_groups.*, COUNT(user_groups_users.user_id)").
			Joins("LEFT JOIN user_groups_users ON user_groups.id = user_groups_users.user_group_id").
			Group("user_groups.id").
			Order("COUNT(user_groups_users.user_id) " + sortedPaginationRequest.Sort.Direction)

		response, err := utils.Paginate(sortedPaginationRequest.Pagination.Page, sortedPaginationRequest.Pagination.Limit, query, &groups)
		return groups, response, err
	}

	response, err = utils.PaginateAndSort(sortedPaginationRequest, query, &groups)
	return groups, response, err
}

func (s *UserGroupService) Get(id string) (group model.UserGroup, err error) {
	err = s.db.Where("id = ?", id).Preload("CustomClaims").Preload("Users").First(&group).Error
	return group, err
}

func (s *UserGroupService) Delete(id string) error {
	var group model.UserGroup
	if err := s.db.Where("id = ?", id).First(&group).Error; err != nil {
		return err
	}

	// Disallow deleting the group if it is an LDAP group and LDAP is enabled
	if group.LdapID != nil && s.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return &common.LdapUserGroupUpdateError{}
	}

	return s.db.Delete(&group).Error
}

func (s *UserGroupService) Create(input dto.UserGroupCreateDto) (group model.UserGroup, err error) {
	group = model.UserGroup{
		FriendlyName: input.FriendlyName,
		Name:         input.Name,
	}

	if input.LdapID != "" {
		group.LdapID = &input.LdapID
	}

	if err := s.db.Preload("Users").Create(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.UserGroup{}, &common.AlreadyInUseError{Property: "name"}
		}
		return model.UserGroup{}, err
	}
	return group, nil
}

func (s *UserGroupService) Update(id string, input dto.UserGroupCreateDto, allowLdapUpdate bool) (group model.UserGroup, err error) {
	group, err = s.Get(id)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Disallow updating the group if it is an LDAP group and LDAP is enabled
	if !allowLdapUpdate && group.LdapID != nil && s.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return model.UserGroup{}, &common.LdapUserGroupUpdateError{}
	}

	group.Name = input.Name
	group.FriendlyName = input.FriendlyName

	if err := s.db.Preload("Users").Save(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.UserGroup{}, &common.AlreadyInUseError{Property: "name"}
		}
		return model.UserGroup{}, err
	}
	return group, nil
}

func (s *UserGroupService) UpdateUsers(id string, userIds []string) (group model.UserGroup, err error) {
	group, err = s.Get(id)
	if err != nil {
		return model.UserGroup{}, err
	}

	// Fetch the users based on the userIds
	var users []model.User
	if len(userIds) > 0 {
		if err := s.db.Where("id IN (?)", userIds).Find(&users).Error; err != nil {
			return model.UserGroup{}, err
		}
	}

	// Replace the current users with the new set of users
	if err := s.db.Model(&group).Association("Users").Replace(users); err != nil {
		return model.UserGroup{}, err
	}

	// Save the updated group
	if err := s.db.Save(&group).Error; err != nil {
		return model.UserGroup{}, err
	}

	return group, nil
}

func (s *UserGroupService) GetUserCountOfGroup(id string) (int64, error) {
	var group model.UserGroup
	if err := s.db.Preload("Users").Where("id = ?", id).First(&group).Error; err != nil {
		return 0, err
	}
	return s.db.Model(&group).Association("Users").Count(), nil
}
