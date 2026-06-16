package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type CustomFieldValueService struct {
	db               *gorm.DB
	appConfigService *AppConfigService
}

func NewCustomFieldValueService(db *gorm.DB, appConfigService *AppConfigService) *CustomFieldValueService {
	return &CustomFieldValueService{db: db, appConfigService: appConfigService}
}

func customFieldAppliesTo(field dto.CustomFieldDto, idType idType) bool {
	switch field.Target {
	case dto.CustomFieldTargetBoth:
		return true
	case dto.CustomFieldTargetUser:
		return idType == UserID
	case dto.CustomFieldTargetGroup:
		return idType == UserGroupID
	default:
		return false
	}
}

// isReservedOIDCClaim checks if a key is reserved by standard OIDC claims, e.g. email or preferred_username.
func isReservedOIDCClaim(key string) bool {
	switch key {
	case "given_name",
		"family_name",
		"name",
		"email",
		"email_verified",
		"preferred_username",
		"display_name",
		"groups",
		TokenTypeClaim,
		"sub",
		"iss",
		"aud",
		"exp",
		"iat",
		"auth_time",
		"nonce",
		"acr",
		"amr",
		"azp",
		"nbf",
		"jti":
		return true
	default:
		return false
	}
}

// idType is the type of the id used to identify the user or user group
type idType string

const (
	UserID      idType = "user_id"
	UserGroupID idType = "user_group_id"
)

// UpdateCustomFieldValuesForUser updates the custom field values for a user.
func (s *CustomFieldValueService) UpdateCustomFieldValuesForUser(ctx context.Context, userID string, customFieldValues []dto.CustomFieldValueCreateDto) ([]model.CustomFieldValue, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	updatedCustomFieldValues, err := s.updateCustomFieldValuesInternal(ctx, UserID, userID, customFieldValues, tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return updatedCustomFieldValues, nil
}

// updateSelfEditableCustomFieldValuesForUser updates only the custom fields a user is allowed to edit themselves.
func (s *CustomFieldValueService) updateSelfEditableCustomFieldValuesForUser(ctx context.Context, userID string, customFieldValues []dto.CustomFieldValueCreateDto, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	fields, err := s.GetConfiguredCustomFieldsForTarget(UserID)
	if err != nil {
		return nil, err
	}

	editableFields := make([]dto.CustomFieldDto, 0, len(fields))
	for _, field := range fields {
		if !field.UserEditable {
			continue
		}
		editableFields = append(editableFields, field)
	}

	return s.updateCustomFieldValuesForFields(ctx, UserID, userID, customFieldValues, editableFields, tx)
}

func (s *CustomFieldValueService) updateCustomFieldValuesForFields(ctx context.Context, idType idType, ownerID string, customFieldValues []dto.CustomFieldValueCreateDto, fields []dto.CustomFieldDto, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	normalizedCustomFieldValues, err := validateCustomFieldValuesAgainstFields(customFieldValues, fields)
	if err != nil {
		return nil, err
	}

	fieldIDs := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldIDs = append(fieldIDs, field.ID)
	}

	valuesByFieldID := make(map[string]dto.CustomFieldValueCreateDto, len(normalizedCustomFieldValues))
	fieldIDsToKeep := make([]string, 0, len(normalizedCustomFieldValues))
	for _, customFieldValue := range normalizedCustomFieldValues {
		valuesByFieldID[customFieldValue.CustomFieldID] = customFieldValue
		fieldIDsToKeep = append(fieldIDsToKeep, customFieldValue.CustomFieldID)
	}

	if len(fieldIDs) > 0 {
		deleteQuery := tx.WithContext(ctx).
			Where(string(idType)+" = ? AND custom_field_id IN ?", ownerID, fieldIDs)
		if len(fieldIDsToKeep) > 0 {
			deleteQuery = deleteQuery.Where("custom_field_id NOT IN ?", fieldIDsToKeep)
		}
		if err := deleteQuery.Delete(&model.CustomFieldValue{}).Error; err != nil {
			return nil, err
		}
	}

	for _, customFieldValue := range valuesByFieldID {
		value := model.CustomFieldValue{
			CustomFieldID: customFieldValue.CustomFieldID,
			Value:         customFieldValue.Value,
		}
		switch idType {
		case UserID:
			value.UserID = &ownerID
		case UserGroupID:
			value.UserGroupID = &ownerID
		}
		if err := tx.
			WithContext(ctx).
			Where(string(idType)+" = ? AND custom_field_id = ?", ownerID, customFieldValue.CustomFieldID).
			Assign(&value).
			FirstOrCreate(&model.CustomFieldValue{}).
			Error; err != nil {
			return nil, err
		}
	}

	switch idType {
	case UserID:
		return s.GetCustomFieldValuesForUser(ctx, ownerID, tx)
	case UserGroupID:
		return s.GetCustomFieldValuesForUserGroup(ctx, ownerID, tx)
	default:
		return nil, nil
	}
}

// UpdateCustomFieldValuesForUserGroup updates the custom field values for a user group.
func (s *CustomFieldValueService) UpdateCustomFieldValuesForUserGroup(ctx context.Context, userGroupID string, customFieldValues []dto.CustomFieldValueCreateDto) ([]model.CustomFieldValue, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	updatedCustomFieldValues, err := s.updateCustomFieldValuesInternal(ctx, UserGroupID, userGroupID, customFieldValues, tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return updatedCustomFieldValues, nil
}

// updateCustomFieldValuesInternal updates the custom field values for a user or user group within a transaction.
func (s *CustomFieldValueService) updateCustomFieldValuesInternal(ctx context.Context, idType idType, value string, customFieldValues []dto.CustomFieldValueCreateDto, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	fields, err := s.GetConfiguredCustomFieldsForTarget(idType)
	if err != nil {
		return nil, err
	}

	customFieldValues, err = validateCustomFieldValuesAgainstFields(customFieldValues, fields)
	if err != nil {
		return nil, err
	}

	var existingCustomFieldValues []model.CustomFieldValue
	err = tx.
		WithContext(ctx).
		Where(string(idType), value).
		Find(&existingCustomFieldValues).
		Error
	if err != nil {
		return nil, err
	}

	// Delete values that are not in the new list.
	for _, existingCustomFieldValue := range existingCustomFieldValues {
		found := false
		for _, customFieldValue := range customFieldValues {
			if customFieldValue.CustomFieldID == existingCustomFieldValue.CustomFieldID {
				found = true
				break
			}
		}

		if !found {
			err = tx.
				WithContext(ctx).
				Delete(&existingCustomFieldValue).
				Error
			if err != nil {
				return nil, err
			}
		}
	}

	// Add or update custom field values.
	for _, inputCustomFieldValue := range customFieldValues {
		customFieldValue := model.CustomFieldValue{
			CustomFieldID: inputCustomFieldValue.CustomFieldID,
			Value:         inputCustomFieldValue.Value,
		}

		switch idType {
		case UserID:
			customFieldValue.UserID = &value
		case UserGroupID:
			customFieldValue.UserGroupID = &value
		}

		// Update the value if it already exists or create a new one.
		err = tx.
			WithContext(ctx).
			Where(string(idType)+" = ? AND custom_field_id = ?", value, inputCustomFieldValue.CustomFieldID).
			Assign(&customFieldValue).
			FirstOrCreate(&model.CustomFieldValue{}).
			Error
		if err != nil {
			return nil, err
		}
	}

	// Get the updated custom field values.
	var updatedCustomFieldValues []model.CustomFieldValue
	err = tx.
		WithContext(ctx).
		Where(string(idType)+" = ?", value).
		Find(&updatedCustomFieldValues).
		Error
	if err != nil {
		return nil, err
	}

	return updatedCustomFieldValues, nil
}

func (s *CustomFieldValueService) GetCustomFieldValuesForUser(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	var customFieldValues []model.CustomFieldValue
	err := tx.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&customFieldValues).
		Error
	if err != nil {
		return nil, err
	}
	return s.applyDefaultCustomFieldValues(UserID, userID, customFieldValues)
}

func (s *CustomFieldValueService) GetCustomFieldValuesForUserGroup(ctx context.Context, userGroupID string, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	var customFieldValues []model.CustomFieldValue
	err := tx.
		WithContext(ctx).
		Where("user_group_id = ?", userGroupID).
		Find(&customFieldValues).
		Error
	if err != nil {
		return nil, err
	}
	return s.applyDefaultCustomFieldValues(UserGroupID, userGroupID, customFieldValues)
}

// GetCustomFieldValuesForUserWithUserGroups returns the custom field values of a user and all user groups the user is a member of,
// prioritizing the user's values over user group values for the same custom field.
func (s *CustomFieldValueService) GetCustomFieldValuesForUserWithUserGroups(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomFieldValue, error) {
	customFieldValues, err := s.GetCustomFieldValuesForUser(ctx, userID, tx)
	if err != nil {
		return nil, err
	}

	valuesByFieldID := make(map[string]model.CustomFieldValue)
	for _, customFieldValue := range customFieldValues {
		valuesByFieldID[customFieldValue.CustomFieldID] = customFieldValue
	}

	// Get all user groups of the user
	var userGroupsOfUser []model.UserGroup
	err = tx.
		WithContext(ctx).
		Preload("CustomFieldValues").
		Joins("JOIN user_groups_users ON user_groups_users.user_group_id = user_groups.id").
		Where("user_groups_users.user_id = ?", userID).
		Find(&userGroupsOfUser).Error
	if err != nil {
		return nil, err
	}

	// Add only non-duplicate custom fields from user groups
	for _, userGroup := range userGroupsOfUser {
		groupCustomFieldValues, err := s.applyDefaultCustomFieldValues(UserGroupID, userGroup.ID, userGroup.CustomFieldValues)
		if err != nil {
			return nil, err
		}
		for _, groupCustomFieldValue := range groupCustomFieldValues {
			if _, exists := valuesByFieldID[groupCustomFieldValue.CustomFieldID]; !exists {
				valuesByFieldID[groupCustomFieldValue.CustomFieldID] = groupCustomFieldValue
			}
		}
	}

	finalCustomFieldValues := make([]model.CustomFieldValue, 0, len(valuesByFieldID))
	for _, customFieldValue := range valuesByFieldID {
		finalCustomFieldValues = append(finalCustomFieldValues, customFieldValue)
	}

	return finalCustomFieldValues, nil
}

func (s *CustomFieldValueService) applyDefaultCustomFieldValues(idType idType, ownerID string, customFieldValues []model.CustomFieldValue) ([]model.CustomFieldValue, error) {
	fields, err := s.GetConfiguredCustomFieldsForTarget(idType)
	if err != nil {
		return nil, err
	}

	valuesByFieldID := make(map[string]struct{}, len(customFieldValues))
	for _, customFieldValue := range customFieldValues {
		valuesByFieldID[customFieldValue.CustomFieldID] = struct{}{}
	}

	effectiveCustomFieldValues := append([]model.CustomFieldValue{}, customFieldValues...)
	for _, field := range fields {
		if field.DefaultValue == "" {
			continue
		}
		if _, ok := valuesByFieldID[field.ID]; ok {
			continue
		}

		customFieldValue := model.CustomFieldValue{
			CustomFieldID: field.ID,
			Value:         field.DefaultValue,
		}
		switch idType {
		case UserID:
			customFieldValue.UserID = &ownerID
		case UserGroupID:
			customFieldValue.UserGroupID = &ownerID
		}
		effectiveCustomFieldValues = append(effectiveCustomFieldValues, customFieldValue)
	}

	return effectiveCustomFieldValues, nil
}

func (s *CustomFieldValueService) GetConfiguredCustomFieldsForTarget(idType idType) ([]dto.CustomFieldDto, error) {
	fields, err := ParseCustomFieldDefinitions(s.appConfigService.GetDbConfig().CustomFields.Value)
	if err != nil {
		return nil, err
	}

	filteredFields := make([]dto.CustomFieldDto, 0, len(fields))
	for _, field := range fields {
		if customFieldAppliesTo(field, idType) {
			filteredFields = append(filteredFields, field)
		}
	}

	return filteredFields, nil
}

func ParseCustomFieldDefinitions(value string) ([]dto.CustomFieldDto, error) {
	if value == "" {
		return nil, nil
	}

	var fields []dto.CustomFieldDto
	if err := json.Unmarshal([]byte(value), &fields); err != nil {
		return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("invalid custom fields JSON: %v", err)}
	}

	seenIDs := make(map[string]struct{}, len(fields))
	seenKeys := make(map[string]struct{}, len(fields))
	for i, field := range fields {
		field.Key = strings.TrimSpace(field.Key)
		fields[i].Key = field.Key

		if err := dto.ValidateStruct(field); err != nil {
			return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s is invalid: %v", field.Key, err)}
		}

		if _, ok := seenIDs[field.ID]; ok {
			return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field id %s is already defined", field.ID)}
		}
		seenIDs[field.ID] = struct{}{}
		if isReservedOIDCClaim(field.Key) {
			return nil, &common.ReservedCustomFieldError{Key: field.Key}
		}
		if _, ok := seenKeys[field.Key]; ok {
			return nil, &common.DuplicateCustomFieldError{Key: field.Key}
		}
		seenKeys[field.Key] = struct{}{}

		if field.ValidationRegex != "" {
			if field.Type != dto.CustomFieldTypeString {
				return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s can only use regex validation for text values", field.Key)}
			}
		}
		if field.Required && field.DefaultValue == "" {
			return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s requires a default value", field.Key)}
		}
		if field.DefaultValue != "" {
			if err := validateCustomFieldValue(dto.CustomFieldValueCreateDto{CustomFieldID: field.ID, Value: field.DefaultValue}, field); err != nil {
				return nil, err
			}
		}
	}

	return fields, nil
}

func validateCustomFieldValuesAgainstFields(customFieldValues []dto.CustomFieldValueCreateDto, fields []dto.CustomFieldDto) ([]dto.CustomFieldValueCreateDto, error) {
	fieldsByID := make(map[string]dto.CustomFieldDto, len(fields))
	for _, field := range fields {
		fieldsByID[field.ID] = field
	}

	valuesByFieldID := make(map[string]dto.CustomFieldValueCreateDto, len(customFieldValues))
	for _, customFieldValue := range customFieldValues {
		field, ok := fieldsByID[customFieldValue.CustomFieldID]
		if !ok {
			continue
		}

		customFieldValue.CustomFieldID = field.ID
		customFieldValue.Key = field.Key

		if _, ok := valuesByFieldID[customFieldValue.CustomFieldID]; ok {
			return nil, &common.DuplicateCustomFieldError{Key: field.Key}
		}

		if field.Type != dto.CustomFieldTypeBoolean && customFieldValue.Value == "" && !field.Required {
			continue
		}

		if err := validateCustomFieldValue(customFieldValue, field); err != nil {
			return nil, err
		}
		valuesByFieldID[customFieldValue.CustomFieldID] = customFieldValue
	}

	normalizedCustomFieldValues := make([]dto.CustomFieldValueCreateDto, 0, len(valuesByFieldID))
	for _, field := range fields {
		customFieldValue, ok := valuesByFieldID[field.ID]
		if ok {
			normalizedCustomFieldValues = append(normalizedCustomFieldValues, customFieldValue)
			continue
		}

		if field.DefaultValue != "" {
			normalizedCustomFieldValues = append(normalizedCustomFieldValues, dto.CustomFieldValueCreateDto{
				CustomFieldID: field.ID,
				Key:           field.Key,
				Value:         field.DefaultValue,
			})
			continue
		}

		if field.Required {
			return nil, &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s is required", field.Key)}
		}
	}

	return normalizedCustomFieldValues, nil
}

func validateCustomFieldValue(customFieldValue dto.CustomFieldValueCreateDto, field dto.CustomFieldDto) error {
	if field.Required && field.Type != dto.CustomFieldTypeBoolean && customFieldValue.Value == "" {
		return &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s is required", field.Key)}
	}

	switch field.Type {
	case dto.CustomFieldTypeString:
		if field.ValidationRegex != "" {
			matches, err := regexp.MatchString(field.ValidationRegex, customFieldValue.Value)
			if err != nil {
				return &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s has invalid validation regex: %v", field.Key, err)}
			}
			if !matches {
				if field.ValidationErrorMessage != "" {
					return &common.CustomFieldValidationError{Message: field.ValidationErrorMessage}
				}
				return &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s does not match the required format", field.Key)}
			}
		}
		return nil
	case dto.CustomFieldTypeNumber:
		if _, err := strconv.ParseFloat(customFieldValue.Value, 64); err != nil {
			return &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s must be a number", field.Key)}
		}
	case dto.CustomFieldTypeBoolean:
		if _, err := strconv.ParseBool(customFieldValue.Value); err != nil {
			return &common.CustomFieldValidationError{Message: fmt.Sprintf("custom field %s must be a boolean", field.Key)}
		}
	}

	return nil
}

func customFieldValueTokenValue(customFieldValue model.CustomFieldValue, field *dto.CustomFieldDto) (any, error) {
	if field != nil {
		switch field.Type {
		case dto.CustomFieldTypeString:
			return customFieldValue.Value, nil
		case dto.CustomFieldTypeNumber:
			value, err := strconv.ParseFloat(customFieldValue.Value, 64)
			if err != nil {
				return nil, err
			}
			return value, nil
		case dto.CustomFieldTypeBoolean:
			value, err := strconv.ParseBool(customFieldValue.Value)
			if err != nil {
				return nil, err
			}
			return value, nil
		}
	}

	var jsonValue any
	if err := json.Unmarshal([]byte(customFieldValue.Value), &jsonValue); err == nil {
		return jsonValue, nil
	}

	return customFieldValue.Value, nil
}
