package service

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCustomFieldDefinitionsValidatesRegex(t *testing.T) {
	_, err := ParseCustomFieldDefinitions(`[{"id":"89bc9c8f-2cd8-4cfd-82c5-5fa14e874f03","key":"department","displayName":"Department","type":"string","target":"user","required":false,"validationRegex":"["}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid validation regex")

	_, err = ParseCustomFieldDefinitions(`[{"id":"353555d9-7de8-4320-a10f-5ca4c122a363","key":"age","displayName":"Age","type":"number","target":"user","required":false,"validationRegex":"^[0-9]+$"}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can only use regex validation for text values")

	_, err = ParseCustomFieldDefinitions(`[{"id":"fe2bc740-6193-4ef2-b1e6-2408a691a98c","key":"department","displayName":"Department","type":"string","target":"user","required":true}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a default value")

	_, err = ParseCustomFieldDefinitions(`[{"id":"be42096c-3dc0-4a9c-8074-086b9f866286","key":"department","displayName":"Department","type":"string","target":"user","required":true,"validationRegex":"^ENG-[0-9]+$","defaultValue":"Sales"}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match the required format")
}

func TestParseCustomFieldDefinitionsValidatesKey(t *testing.T) {
	fields, err := ParseCustomFieldDefinitions(`[{"id":"36c1e786-c9e9-4daf-ab51-502ab8efc9ea","key":" department ","displayName":"Department","type":"string","target":"user","required":false}]`)
	require.NoError(t, err)
	require.Len(t, fields, 1)
	assert.Equal(t, "department", fields[0].Key)

	_, err = ParseCustomFieldDefinitions(`[{"id":"c0e41fb3-59c7-488a-8edb-57e94e9f15ac","key":"   ","displayName":"Empty","type":"string","target":"user","required":false}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom field key is required")

	_, err = ParseCustomFieldDefinitions(`[{"id":"8b2ff8eb-bcf5-4866-b690-1a5b6f9da56c","key":"email","displayName":"Email","type":"string","target":"user","required":false}]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

func TestValidateCustomFieldValuesAgainstFieldsAppliesRegex(t *testing.T) {
	fields := []dto.CustomFieldDto{
		{
			ID:                     "4ca0513e-e223-4900-8c5e-303acac4d021",
			Key:                    "employee_id",
			DisplayName:            "Employee ID",
			Type:                   dto.CustomFieldTypeString,
			ValidationRegex:        "^EMP-[0-9]+$",
			ValidationErrorMessage: "Employee ID must start with EMP-",
		},
	}

	_, err := validateCustomFieldValuesAgainstFields([]dto.CustomFieldValueCreateDto{
		{CustomFieldID: "4ca0513e-e223-4900-8c5e-303acac4d021", Value: "INVALID"},
	}, fields)
	require.Error(t, err)
	assert.Equal(t, "Employee ID must start with EMP-", err.Error())

	values, err := validateCustomFieldValuesAgainstFields([]dto.CustomFieldValueCreateDto{
		{CustomFieldID: "4ca0513e-e223-4900-8c5e-303acac4d021", Value: "EMP-123"},
	}, fields)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "4ca0513e-e223-4900-8c5e-303acac4d021", values[0].CustomFieldID)
	assert.Equal(t, "EMP-123", values[0].Value)
}

func TestValidateCustomFieldValuesAgainstFieldsUsesDefaultValue(t *testing.T) {
	fields := []dto.CustomFieldDto{
		{
			ID:           "4225f448-f189-47d5-97d6-90292cc5bf9e",
			Key:          "department",
			DisplayName:  "Department",
			Type:         dto.CustomFieldTypeString,
			Required:     true,
			DefaultValue: "Engineering",
		},
		{
			ID:           "398c23a4-c2e7-4b87-b6df-ed6bf1810579",
			Key:          "active",
			DisplayName:  "Active",
			Type:         dto.CustomFieldTypeBoolean,
			Required:     true,
			DefaultValue: "false",
		},
	}

	values, err := validateCustomFieldValuesAgainstFields(nil, fields)
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.Equal(t, dto.CustomFieldValueCreateDto{CustomFieldID: "4225f448-f189-47d5-97d6-90292cc5bf9e", Key: "department", Value: "Engineering"}, values[0])
	assert.Equal(t, dto.CustomFieldValueCreateDto{CustomFieldID: "398c23a4-c2e7-4b87-b6df-ed6bf1810579", Key: "active", Value: "false"}, values[1])
}

func TestGetCustomFieldValuesAppliesDefaultValues(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{Username: "alice", FirstName: "Alice", DisplayName: "Alice"}
	require.NoError(t, db.Create(&user).Error)
	group := model.UserGroup{Name: "engineering", FriendlyName: "Engineering"}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Model(&user).Association("UserGroups").Append(&group))

	appConfigService := NewTestAppConfigService(&model.AppConfig{
		CustomFields: model.AppConfigVariable{Value: `[
			{"id":"81b3c82a-46c8-49c0-9559-a31df8586ef1","key":"department","displayName":"Department","type":"string","target":"user","required":false,"defaultValue":"Engineering"},
			{"id":"b064d601-bc94-4ecf-a5cb-b783f3de0281","key":"group_label","displayName":"Group label","type":"string","target":"group","required":false,"defaultValue":"Employee"}
		]`},
	})
	service := NewCustomFieldValueService(db, appConfigService)

	userValues, err := service.GetCustomFieldValuesForUser(t.Context(), user.ID, db)
	require.NoError(t, err)
	require.Len(t, userValues, 1)
	assert.Equal(t, "81b3c82a-46c8-49c0-9559-a31df8586ef1", userValues[0].CustomFieldID)
	assert.Equal(t, "Engineering", userValues[0].Value)
	require.NotNil(t, userValues[0].UserID)

	groupValues, err := service.GetCustomFieldValuesForUserGroup(t.Context(), group.ID, db)
	require.NoError(t, err)
	require.Len(t, groupValues, 1)
	assert.Equal(t, "b064d601-bc94-4ecf-a5cb-b783f3de0281", groupValues[0].CustomFieldID)
	assert.Equal(t, "Employee", groupValues[0].Value)
	require.NotNil(t, groupValues[0].UserGroupID)

	combinedValues, err := service.GetCustomFieldValuesForUserWithUserGroups(t.Context(), user.ID, db)
	require.NoError(t, err)
	require.Len(t, combinedValues, 2)
	valuesByFieldID := map[string]string{}
	for _, value := range combinedValues {
		valuesByFieldID[value.CustomFieldID] = value.Value
	}
	assert.Equal(t, "Engineering", valuesByFieldID["81b3c82a-46c8-49c0-9559-a31df8586ef1"])
	assert.Equal(t, "Employee", valuesByFieldID["b064d601-bc94-4ecf-a5cb-b783f3de0281"])
}

func TestUpdateSelfEditableCustomFieldValuesForUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{Username: "alice", FirstName: "Alice", DisplayName: "Alice"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create([]model.CustomFieldValue{
		{UserID: &user.ID, CustomFieldID: "608a9c35-2330-433d-bf33-c46f065d5d06", Value: "old"},
		{UserID: &user.ID, CustomFieldID: "8501e000-09bb-428c-8be3-b0d3b0c682fd", Value: "admin"},
	}).Error)

	appConfigService := NewTestAppConfigService(&model.AppConfig{
		CustomFields: model.AppConfigVariable{Value: `[
			{"id":"608a9c35-2330-433d-bf33-c46f065d5d06","key":"nickname","displayName":"Nickname","type":"string","target":"user","required":false,"userEditable":true},
			{"id":"8501e000-09bb-428c-8be3-b0d3b0c682fd","key":"cost_center","displayName":"Cost center","type":"string","target":"user","required":false,"userEditable":false}
		]`},
	})
	service := NewCustomFieldValueService(db, appConfigService)

	tx := db.Begin()
	updatedValues, err := service.updateSelfEditableCustomFieldValuesForUser(t.Context(), user.ID, []dto.CustomFieldValueCreateDto{
		{CustomFieldID: "608a9c35-2330-433d-bf33-c46f065d5d06", Value: "new"},
	}, tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)

	valuesByFieldID := map[string]string{}
	for _, value := range updatedValues {
		valuesByFieldID[value.CustomFieldID] = value.Value
	}
	assert.Equal(t, "new", valuesByFieldID["608a9c35-2330-433d-bf33-c46f065d5d06"])
	assert.Equal(t, "admin", valuesByFieldID["8501e000-09bb-428c-8be3-b0d3b0c682fd"])

	tx = db.Begin()
	_, err = service.updateSelfEditableCustomFieldValuesForUser(t.Context(), user.ID, []dto.CustomFieldValueCreateDto{
		{CustomFieldID: "invalid", Value: "user"},
	}, tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
	tx.Rollback()

	var costCenter model.CustomFieldValue
	require.NoError(t, db.Where("user_id = ? AND custom_field_id = ?", user.ID, "8501e000-09bb-428c-8be3-b0d3b0c682fd").First(&costCenter).Error)
	assert.Equal(t, "admin", costCenter.Value)
}
