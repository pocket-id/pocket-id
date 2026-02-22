// We use model_test here to avoid an import cycle
package model_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func TestAppConfigVariable_AsMinutesDuration(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		expected        time.Duration
		expectedSeconds int
	}{
		{
			name:            "valid positive integer",
			value:           "60",
			expected:        60 * time.Minute,
			expectedSeconds: 3600,
		},
		{
			name:            "valid zero integer",
			value:           "0",
			expected:        0,
			expectedSeconds: 0,
		},
		{
			name:            "negative integer",
			value:           "-30",
			expected:        -30 * time.Minute,
			expectedSeconds: -1800,
		},
		{
			name:            "invalid non-integer",
			value:           "not-a-number",
			expected:        0,
			expectedSeconds: 0,
		},
		{
			name:            "empty string",
			value:           "",
			expected:        0,
			expectedSeconds: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configVar := model.AppConfigVariable{
				Value: tt.value,
			}

			result := configVar.AsDurationMinutes()
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedSeconds, int(result.Seconds()))
		})
	}
}

// This test ensures that the model.AppConfig and dto.AppConfigUpdateDto structs match:
// - They should have the same properties, where the "json" tag of dto.AppConfigUpdateDto should match the "key" tag in model.AppConfig
// - dto.AppConfigDto should not include "internal" fields from model.AppConfig
// This test is primarily meant to catch discrepancies between the two structs as fields are added or removed over time
func TestAppConfigStructMatchesUpdateDto(t *testing.T) {
	appConfigType := reflect.TypeFor[model.AppConfig]()
	updateDtoType := reflect.TypeFor[dto.AppConfigUpdateDto]()

	// Process AppConfig fields
	appConfigFields := make(map[string]string)
	for field := range appConfigType.Fields() {
		if field.Tag.Get("key") == "" {
			// Skip internal fields
			continue
		}

		// Extract the key name from the tag (takes the part before any comma)
		keyTag := field.Tag.Get("key")
		keyName, _, _ := strings.Cut(keyTag, ",")

		appConfigFields[field.Name] = keyName
	}

	// Process AppConfigUpdateDto fields
	dtoFields := make(map[string]string)
	for field := range updateDtoType.Fields() {
		// Extract the json name from the tag (takes the part before any binding constraints)
		jsonTag := field.Tag.Get("json")
		jsonName, _, _ := strings.Cut(jsonTag, ",")

		dtoFields[jsonName] = field.Name
	}

	// Verify every AppConfig field has a matching DTO field with the same name
	for fieldName, keyName := range appConfigFields {
		if strings.HasSuffix(fieldName, "ImageType") || keyName == "instanceId" {
			// Skip internal fields that shouldn't be in the DTO
			continue
		}

		// Check if there's a DTO field with a matching JSON tag
		_, exists := dtoFields[keyName]
		assert.True(t, exists, "Field %s with key '%s' in AppConfig has no matching field in AppConfigUpdateDto", fieldName, keyName)
	}

	// Verify every DTO field has a matching AppConfig field
	for jsonName, fieldName := range dtoFields {
		// Find a matching field in AppConfig by key tag
		found := false
		for _, keyName := range appConfigFields {
			if keyName == jsonName {
				found = true
				break
			}
		}

		assert.True(t, found, "Field %s with json tag '%s' in AppConfigUpdateDto has no matching field in AppConfig", fieldName, jsonName)
	}
}
