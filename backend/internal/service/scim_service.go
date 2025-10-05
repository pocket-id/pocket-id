package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type ScimService struct {
	db               *gorm.DB
	userService      *UserService
	userGroupService *UserGroupService
	appConfigService *AppConfigService
}

func NewScimService(db *gorm.DB, userService *UserService, userGroupService *UserGroupService, appConfigService *AppConfigService) *ScimService {
	return &ScimService{
		db:               db,
		userService:      userService,
		userGroupService: userGroupService,
		appConfigService: appConfigService,
	}
}

// ListUsers returns a list of users in SCIM format
func (s *ScimService) ListUsers(ctx context.Context, startIndex, count int, filter string) (*dto.ScimListResponse, error) {
	var users []model.User
	query := s.db.WithContext(ctx).Preload("UserGroups")

	// Apply filter if provided (basic userName filter support)
	if filter != "" {
		// Simple filter parser for userName eq "value"
		if strings.Contains(filter, "userName eq") {
			parts := strings.Split(filter, "\"")
			if len(parts) >= 2 {
				username := parts[1]
				query = query.Where("username = ?", username)
			}
		}
	}

	// Get total count
	var totalResults int64
	if err := query.Model(&model.User{}).Count(&totalResults).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := 0
	if startIndex > 1 {
		offset = startIndex - 1
	}
	query = query.Offset(offset).Limit(count)

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	// Convert to SCIM format
	scimUsers := make([]dto.ScimUser, len(users))
	for i, user := range users {
		scimUsers[i] = s.userToScim(&user)
	}

	return &dto.ScimListResponse{
		Schemas:      []string{dto.ScimSchemaListResponse},
		TotalResults: int(totalResults),
		StartIndex:   startIndex,
		ItemsPerPage: len(scimUsers),
		Resources:    scimUsers,
	}, nil
}

// GetUser returns a user by ID in SCIM format
func (s *ScimService) GetUser(ctx context.Context, id string) (*dto.ScimUser, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "User", ID: id}
		}
		return nil, err
	}

	scimUser := s.userToScim(&user)
	return &scimUser, nil
}

// CreateUser creates a new user from SCIM data
func (s *ScimService) CreateUser(ctx context.Context, scimUser *dto.ScimUser) (*dto.ScimUser, error) {
	// Convert SCIM user to internal user model
	user := model.User{
		Username: scimUser.UserName,
		Disabled: !scimUser.Active,
		IsAdmin:  false,
	}

	if scimUser.Name != nil {
		user.FirstName = scimUser.Name.GivenName
		user.LastName = scimUser.Name.FamilyName
		user.DisplayName = scimUser.Name.Formatted
	}

	if len(scimUser.Emails) > 0 {
		email := scimUser.Emails[0].Value
		user.Email = &email
	}

	if scimUser.Locale != "" {
		user.Locale = &scimUser.Locale
	}

	// Create the user in the database
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, err
	}

	// Load associations
	if err := s.db.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}

	result := s.userToScim(&user)
	return &result, nil
}

// UpdateUser updates an existing user (PUT - full replacement)
func (s *ScimService) UpdateUser(ctx context.Context, id string, scimUser *dto.ScimUser) (*dto.ScimUser, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "User", ID: id}
		}
		return nil, err
	}

	// Update fields
	user.Username = scimUser.UserName
	user.Disabled = !scimUser.Active

	if scimUser.Name != nil {
		user.FirstName = scimUser.Name.GivenName
		user.LastName = scimUser.Name.FamilyName
		user.DisplayName = scimUser.Name.Formatted
	}

	if len(scimUser.Emails) > 0 {
		email := scimUser.Emails[0].Value
		user.Email = &email
	} else {
		user.Email = nil
	}

	if scimUser.Locale != "" {
		user.Locale = &scimUser.Locale
	} else {
		user.Locale = nil
	}

	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}

	// Reload with associations
	if err := s.db.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}

	result := s.userToScim(&user)
	return &result, nil
}

// PatchUser applies partial updates to a user (PATCH)
func (s *ScimService) PatchUser(ctx context.Context, id string, patchOp *dto.ScimPatchOp) (*dto.ScimUser, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "User", ID: id}
		}
		return nil, err
	}

	// Process operations
	for _, op := range patchOp.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			if err := s.applyPatchReplace(&user, op.Path, op.Value); err != nil {
				return nil, err
			}
		case "add":
			if err := s.applyPatchAdd(&user, op.Path, op.Value); err != nil {
				return nil, err
			}
		case "remove":
			if err := s.applyPatchRemove(&user, op.Path); err != nil {
				return nil, err
			}
		}
	}

	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}

	// Reload with associations
	if err := s.db.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}

	result := s.userToScim(&user)
	return &result, nil
}

// DeleteUser soft deletes a user (sets disabled to true)
func (s *ScimService) DeleteUser(ctx context.Context, id string) error {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &common.ScimResourceNotFoundError{ResourceType: "User", ID: id}
		}
		return err
	}

	// Soft delete by disabling the user
	user.Disabled = true
	return s.db.WithContext(ctx).Save(&user).Error
}

// ListGroups returns a list of groups in SCIM format
func (s *ScimService) ListGroups(ctx context.Context, startIndex, count int, filter string) (*dto.ScimListResponse, error) {
	var groups []model.UserGroup
	query := s.db.WithContext(ctx).Preload("Users")

	// Apply filter if provided
	if filter != "" {
		if strings.Contains(filter, "displayName eq") {
			parts := strings.Split(filter, "\"")
			if len(parts) >= 2 {
				displayName := parts[1]
				query = query.Where("friendly_name = ?", displayName)
			}
		}
	}

	// Get total count
	var totalResults int64
	if err := query.Model(&model.UserGroup{}).Count(&totalResults).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := 0
	if startIndex > 1 {
		offset = startIndex - 1
	}
	query = query.Offset(offset).Limit(count)

	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}

	// Convert to SCIM format
	scimGroups := make([]dto.ScimGroup, len(groups))
	for i, group := range groups {
		scimGroups[i] = s.groupToScim(&group)
	}

	return &dto.ScimListResponse{
		Schemas:      []string{dto.ScimSchemaListResponse},
		TotalResults: int(totalResults),
		StartIndex:   startIndex,
		ItemsPerPage: len(scimGroups),
		Resources:    scimGroups,
	}, nil
}

// GetGroup returns a group by ID in SCIM format
func (s *ScimService) GetGroup(ctx context.Context, id string) (*dto.ScimGroup, error) {
	var group model.UserGroup
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "Group", ID: id}
		}
		return nil, err
	}

	scimGroup := s.groupToScim(&group)
	return &scimGroup, nil
}

// CreateGroup creates a new group from SCIM data
func (s *ScimService) CreateGroup(ctx context.Context, scimGroup *dto.ScimGroup) (*dto.ScimGroup, error) {
	group := model.UserGroup{
		FriendlyName: scimGroup.DisplayName,
		Name:         utils.NormalizeGroupName(scimGroup.DisplayName),
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the group
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Add members if provided
	if len(scimGroup.Members) > 0 {
		var users []model.User
		userIDs := make([]string, len(scimGroup.Members))
		for i, member := range scimGroup.Members {
			userIDs[i] = member.Value
		}
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Model(&group).Association("Users").Append(users); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload with associations
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", group.ID).Error; err != nil {
		return nil, err
	}

	result := s.groupToScim(&group)
	return &result, nil
}

// UpdateGroup updates an existing group (PUT - full replacement)
func (s *ScimService) UpdateGroup(ctx context.Context, id string, scimGroup *dto.ScimGroup) (*dto.ScimGroup, error) {
	var group model.UserGroup
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "Group", ID: id}
		}
		return nil, err
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update fields
	group.FriendlyName = scimGroup.DisplayName
	group.Name = utils.NormalizeGroupName(scimGroup.DisplayName)

	if err := tx.Save(&group).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Replace members
	if err := tx.Model(&group).Association("Users").Clear(); err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(scimGroup.Members) > 0 {
		var users []model.User
		userIDs := make([]string, len(scimGroup.Members))
		for i, member := range scimGroup.Members {
			userIDs[i] = member.Value
		}
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Model(&group).Association("Users").Append(users); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload with associations
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", group.ID).Error; err != nil {
		return nil, err
	}

	result := s.groupToScim(&group)
	return &result, nil
}

// PatchGroup applies partial updates to a group (PATCH)
func (s *ScimService) PatchGroup(ctx context.Context, id string, patchOp *dto.ScimPatchOp) (*dto.ScimGroup, error) {
	var group model.UserGroup
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.ScimResourceNotFoundError{ResourceType: "Group", ID: id}
		}
		return nil, err
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Process operations
	for _, op := range patchOp.Operations {
		switch strings.ToLower(op.Op) {
		case "add":
			if strings.ToLower(op.Path) == "members" {
				if err := s.applyGroupMemberAdd(tx, &group, op.Value); err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		case "remove":
			if strings.ToLower(op.Path) == "members" {
				if err := s.applyGroupMemberRemove(tx, &group, op.Value); err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		case "replace":
			if op.Path == "displayName" || op.Path == "" {
				if displayName, ok := op.Value.(string); ok {
					group.FriendlyName = displayName
					group.Name = utils.NormalizeGroupName(displayName)
					if err := tx.Save(&group).Error; err != nil {
						tx.Rollback()
						return nil, err
					}
				}
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload with associations
	if err := s.db.WithContext(ctx).Preload("Users").First(&group, "id = ?", group.ID).Error; err != nil {
		return nil, err
	}

	result := s.groupToScim(&group)
	return &result, nil
}

// DeleteGroup deletes a group
func (s *ScimService) DeleteGroup(ctx context.Context, id string) error {
	var group model.UserGroup
	if err := s.db.WithContext(ctx).First(&group, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &common.ScimResourceNotFoundError{ResourceType: "Group", ID: id}
		}
		return err
	}

	return s.db.WithContext(ctx).Delete(&group).Error
}

// GetServiceProviderConfig returns the SCIM service provider configuration
func (s *ScimService) GetServiceProviderConfig() *dto.ScimServiceProviderConfig {
	return &dto.ScimServiceProviderConfig{
		Schemas: []string{dto.ScimSchemaServiceProviderConfig},
		Patch: dto.ScimSupported{
			Supported: true,
		},
		Bulk: dto.ScimBulkSupported{
			Supported: false,
		},
		Filter: dto.ScimFilterSupported{
			Supported:  true,
			MaxResults: 200,
		},
		ChangePassword: dto.ScimSupported{
			Supported: false,
		},
		Sort: dto.ScimSupported{
			Supported: false,
		},
		Etag: dto.ScimSupported{
			Supported: false,
		},
		AuthenticationSchemes: []dto.ScimAuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication scheme using the OAuth Bearer Token Standard",
				SpecURI:     "http://www.rfc-editor.org/info/rfc6750",
				Primary:     true,
			},
		},
		Meta: &dto.ScimMeta{
			ResourceType: "ServiceProviderConfig",
			Location:     "/scim/v2/ServiceProviderConfig",
		},
	}
}

// GetResourceTypes returns the supported SCIM resource types
func (s *ScimService) GetResourceTypes() []dto.ScimResourceType {
	return []dto.ScimResourceType{
		{
			Schemas:     []string{dto.ScimSchemaResourceType},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/scim/v2/Users",
			Description: "User Account",
			Schema:      dto.ScimSchemaCore,
			Meta: &dto.ScimMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/User",
			},
		},
		{
			Schemas:     []string{dto.ScimSchemaResourceType},
			ID:          "Group",
			Name:        "Group",
			Endpoint:    "/scim/v2/Groups",
			Description: "Group",
			Schema:      dto.ScimSchemaGroup,
			Meta: &dto.ScimMeta{
				ResourceType: "ResourceType",
				Location:     "/scim/v2/ResourceTypes/Group",
			},
		},
	}
}

// GetSchemas returns the supported SCIM schemas
func (s *ScimService) GetSchemas() []dto.ScimSchema {
	return []dto.ScimSchema{
		{
			ID:          dto.ScimSchemaCore,
			Name:        "User",
			Description: "User Account",
			Attributes: []dto.ScimSchemaAttribute{
				{Name: "userName", Type: "string", Required: true, Mutability: "readWrite", Returned: "default", Uniqueness: "server"},
				{Name: "name", Type: "complex", Mutability: "readWrite", Returned: "default"},
				{Name: "emails", Type: "complex", MultiValued: true, Mutability: "readWrite", Returned: "default"},
				{Name: "active", Type: "boolean", Mutability: "readWrite", Returned: "default"},
				{Name: "groups", Type: "complex", MultiValued: true, Mutability: "readOnly", Returned: "default"},
			},
		},
		{
			ID:          dto.ScimSchemaGroup,
			Name:        "Group",
			Description: "Group",
			Attributes: []dto.ScimSchemaAttribute{
				{Name: "displayName", Type: "string", Required: true, Mutability: "readWrite", Returned: "default"},
				{Name: "members", Type: "complex", MultiValued: true, Mutability: "readWrite", Returned: "default"},
			},
		},
	}
}

// Helper functions

func (s *ScimService) userToScim(user *model.User) dto.ScimUser {
	scimUser := dto.ScimUser{
		Schemas:  []string{dto.ScimSchemaCore},
		ID:       user.ID,
		UserName: user.Username,
		Active:   !user.Disabled,
		Name: &dto.ScimUserName{
			GivenName:  user.FirstName,
			FamilyName: user.LastName,
			Formatted:  user.DisplayName,
		},
		Meta: &dto.ScimMeta{
			ResourceType: "User",
			Created:      user.CreatedAt.ToTime().Format(time.RFC3339),
			Location:     fmt.Sprintf("/scim/v2/Users/%s", user.ID),
		},
	}

	if user.Email != nil && *user.Email != "" {
		scimUser.Emails = []dto.ScimEmail{
			{
				Value:   *user.Email,
				Primary: true,
			},
		}
	}

	if user.Locale != nil {
		scimUser.Locale = *user.Locale
	}

	// Add group memberships
	if len(user.UserGroups) > 0 {
		scimUser.Groups = make([]dto.ScimGroupRef, len(user.UserGroups))
		for i, group := range user.UserGroups {
			scimUser.Groups[i] = dto.ScimGroupRef{
				Value:   group.ID,
				Display: group.FriendlyName,
				Ref:     fmt.Sprintf("/scim/v2/Groups/%s", group.ID),
			}
		}
	}

	return scimUser
}

func (s *ScimService) groupToScim(group *model.UserGroup) dto.ScimGroup {
	scimGroup := dto.ScimGroup{
		Schemas:     []string{dto.ScimSchemaGroup},
		ID:          group.ID,
		DisplayName: group.FriendlyName,
		Meta: &dto.ScimMeta{
			ResourceType: "Group",
			Created:      group.CreatedAt.ToTime().Format(time.RFC3339),
			Location:     fmt.Sprintf("/scim/v2/Groups/%s", group.ID),
		},
	}

	if len(group.Users) > 0 {
		scimGroup.Members = make([]dto.ScimMember, len(group.Users))
		for i, user := range group.Users {
			scimGroup.Members[i] = dto.ScimMember{
				Value:   user.ID,
				Display: user.Username,
				Ref:     fmt.Sprintf("/scim/v2/Users/%s", user.ID),
			}
		}
	}

	return scimGroup
}

func (s *ScimService) applyPatchReplace(user *model.User, path string, value interface{}) error {
	switch strings.ToLower(path) {
	case "active":
		if active, ok := value.(bool); ok {
			user.Disabled = !active
		}
	case "username":
		if username, ok := value.(string); ok {
			user.Username = username
		}
	case "name.givenname":
		if givenName, ok := value.(string); ok {
			user.FirstName = givenName
		}
	case "name.familyname":
		if familyName, ok := value.(string); ok {
			user.LastName = familyName
		}
	}
	return nil
}

func (s *ScimService) applyPatchAdd(user *model.User, path string, value interface{}) error {
	// For now, treat add the same as replace for simplicity
	return s.applyPatchReplace(user, path, value)
}

func (s *ScimService) applyPatchRemove(user *model.User, path string) error {
	switch strings.ToLower(path) {
	case "emails":
		user.Email = nil
	}
	return nil
}

func (s *ScimService) applyGroupMemberAdd(tx *gorm.DB, group *model.UserGroup, value interface{}) error {
	members, ok := value.([]interface{})
	if !ok {
		// Try single member
		if memberMap, ok := value.(map[string]interface{}); ok {
			members = []interface{}{memberMap}
		} else {
			return nil
		}
	}

	userIDs := make([]string, 0)
	for _, member := range members {
		if memberMap, ok := member.(map[string]interface{}); ok {
			if userID, ok := memberMap["value"].(string); ok {
				userIDs = append(userIDs, userID)
			}
		}
	}

	if len(userIDs) > 0 {
		var users []model.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}
		if err := tx.Model(group).Association("Users").Append(users); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScimService) applyGroupMemberRemove(tx *gorm.DB, group *model.UserGroup, value interface{}) error {
	members, ok := value.([]interface{})
	if !ok {
		// Try single member
		if memberMap, ok := value.(map[string]interface{}); ok {
			members = []interface{}{memberMap}
		} else {
			return nil
		}
	}

	userIDs := make([]string, 0)
	for _, member := range members {
		if memberMap, ok := member.(map[string]interface{}); ok {
			if userID, ok := memberMap["value"].(string); ok {
				userIDs = append(userIDs, userID)
			}
		}
	}

	if len(userIDs) > 0 {
		var users []model.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}
		if err := tx.Model(group).Association("Users").Delete(users); err != nil {
			return err
		}
	}

	return nil
}
