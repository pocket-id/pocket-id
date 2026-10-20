//go:build unit

package claimmappingpolicy

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := testutils.NewDatabaseForTest(t)
	return newService(db)
}

// getDefaultClaimMappingPolicy loads the policy currently flagged as default
func getDefaultClaimMappingPolicy(t *testing.T, service *Service) OidcClaimMappingPolicy {
	t.Helper()
	var policy OidcClaimMappingPolicy
	err := service.db.WithContext(t.Context()).Where("is_default = ?", true).First(&policy).Error
	require.NoError(t, err)
	return policy
}

// TestDeleteClaimMappingPolicyResetsClients checks that clients pointing at a deleted policy are
// reset to NULL, which makes them fall back to the default policy.
func TestDeleteClaimMappingPolicyResetsClients(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	service := newService(db)

	policy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Attached"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	client := model.OidcClient{Base: model.Base{ID: "attached-client"}, Name: "Attached Client", ClaimMappingPolicyId: new(policy.ID)}
	require.NoError(t, db.Create(&client).Error)

	require.NoError(t, service.DeleteClaimMappingPolicy(context.Background(), policy.ID))

	var updated model.OidcClient
	require.NoError(t, db.First(&updated, "id = ?", client.ID).Error)
	require.Nil(t, updated.ClaimMappingPolicyId)
}

func TestListClaimMappingPolicyNotEmpty(t *testing.T) {
	service := newTestService(t)

	policies, pagination, err := service.ListClaimMappingPolicy(context.Background(), "", utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, policies)
	assert.Equal(t, int64(1), pagination.TotalItems)
}

func TestCreateClaimMappingPolicy(t *testing.T) {
	service := newTestService(t)

	dto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Test Policy",
			IsDefault: false,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
				Scope:       []string{"openid"},
				AccessToken: true,
				IDToken:     true,
			},
			{
				ClaimName:   "name",
				SourceType:  "user_field",
				SourceValue: string(UserFieldDisplayName),
				Scope:       []string{"profile"},
				AccessToken: true,
				IDToken:     true,
			},
		},
	}

	result, err := service.CreateClaimMappingPolicy(context.Background(), dto)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "Test Policy", result.Name)
	assert.False(t, result.IsDefault)
	assert.Len(t, result.ClaimMappings, 2)

	// Verify it was persisted
	policy, err := service.GetClaimMappingPolicy(context.Background(), result.ID)
	require.NoError(t, err)
	assert.Equal(t, result.ID, policy.ID)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.Len(t, policy.ClaimMappings, 2)
}

func TestCreateClaimMappingPolicyWithDefault(t *testing.T) {
	service := newTestService(t)

	// Create a default policy
	dto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Default Policy",
			IsDefault: true,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}

	result, err := service.CreateClaimMappingPolicy(context.Background(), dto)
	require.NoError(t, err)
	assert.True(t, result.IsDefault)

	// Verify it's the default
	defaultPolicy := getDefaultClaimMappingPolicy(t, service)
	assert.Equal(t, result.ID, defaultPolicy.ID)
}

func TestCreateClaimMappingPolicyDefaultUnsetsOthers(t *testing.T) {
	service := newTestService(t)

	// Create first default policy
	dto1 := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "First Default",
			IsDefault: true,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result1, err := service.CreateClaimMappingPolicy(context.Background(), dto1)
	require.NoError(t, err)

	// Create second default policy - should unset the first one
	dto2 := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Second Default",
			IsDefault: true,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result2, err := service.CreateClaimMappingPolicy(context.Background(), dto2)
	require.NoError(t, err)

	// Check that the first one is no longer default
	policy1, err := service.GetClaimMappingPolicy(context.Background(), result1.ID)
	require.NoError(t, err)
	assert.False(t, policy1.IsDefault)

	// Check that the second one is default
	policy2, err := service.GetClaimMappingPolicy(context.Background(), result2.ID)
	require.NoError(t, err)
	assert.True(t, policy2.IsDefault)

	// Check the default policy is the second one
	defaultPolicy := getDefaultClaimMappingPolicy(t, service)
	assert.Equal(t, result2.ID, defaultPolicy.ID)
}

func TestGetClaimMappingPolicyNotFound(t *testing.T) {
	service := newTestService(t)

	_, err := service.GetClaimMappingPolicy(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestUpdateClaimMappingPolicy(t *testing.T) {
	service := newTestService(t)

	mappings := []OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "email",
			SourceType:  "user_field",
			SourceValue: string(UserFieldEmail),
		},
	}
	updateDto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Updated Policy",
			IsDefault: false,
		},
		ClaimMappings: mappings,
	}
	// Create initial policy
	dto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Initial Policy",
			IsDefault: false,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result, err := service.CreateClaimMappingPolicy(context.Background(), dto)
	require.NoError(t, err)

	updated, err := service.UpdateClaimMappingPolicy(context.Background(), result.ID, updateDto)
	require.NoError(t, err)
	assert.Equal(t, "Updated Policy", updated.Name)
	assert.Len(t, updated.ClaimMappings, 2)

	// Verify persistence
	policy, err := service.GetClaimMappingPolicy(context.Background(), result.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Policy", policy.Name)
	assert.Len(t, policy.ClaimMappings, 2)
}

func TestUpdateClaimMappingPolicySetAsDefault(t *testing.T) {
	service := newTestService(t)

	// Create two policies, neither default
	dto1 := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Policy 1",
			IsDefault: false,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result1, err := service.CreateClaimMappingPolicy(context.Background(), dto1)
	require.NoError(t, err)

	dto2 := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Policy 2",
			IsDefault: false,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result2, err := service.CreateClaimMappingPolicy(context.Background(), dto2)
	require.NoError(t, err)

	// Update policy 1 to be default
	updateDto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Policy 1",
			IsDefault: true,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}

	_, err = service.UpdateClaimMappingPolicy(context.Background(), result1.ID, updateDto)
	require.NoError(t, err)

	// Verify policy 1 is now default
	policy1, err := service.GetClaimMappingPolicy(context.Background(), result1.ID)
	require.NoError(t, err)
	assert.True(t, policy1.IsDefault)

	// Verify policy 2 is NOT default
	policy2, err := service.GetClaimMappingPolicy(context.Background(), result2.ID)
	require.NoError(t, err)
	assert.False(t, policy2.IsDefault)
}

func TestDeleteClaimMappingPolicy(t *testing.T) {
	service := newTestService(t)

	// Create a non-default policy
	dto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "To Delete",
			IsDefault: false,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result, err := service.CreateClaimMappingPolicy(context.Background(), dto)
	require.NoError(t, err)

	// Delete it
	err = service.DeleteClaimMappingPolicy(context.Background(), result.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = service.GetClaimMappingPolicy(context.Background(), result.ID)
	require.Error(t, err)
}

// TestDeleteDefaultClaimMappingPolicyFallsBackToBuiltIn checks that deleting the default policy makes the built-in policy the default again
func TestDeleteDefaultClaimMappingPolicyFallsBackToBuiltIn(t *testing.T) {
	service := newTestService(t)

	// Create a default policy, which takes the default flag over from the built-in one
	dto := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
			Name:      "Default Policy",
			IsDefault: true,
		},
		ClaimMappings: []OidcClaimMappingDto{
			{
				ClaimName:   "user_id",
				SourceType:  "user_field",
				SourceValue: string(UserFieldID),
			},
		},
	}
	result, err := service.CreateClaimMappingPolicy(context.Background(), dto)
	require.NoError(t, err)

	// Delete it
	err = service.DeleteClaimMappingPolicy(context.Background(), result.ID)
	require.NoError(t, err)

	_, err = service.GetClaimMappingPolicy(context.Background(), result.ID)
	require.Error(t, err)

	// The built-in policy is the default again
	builtIn, err := service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
	require.NoError(t, err)
	assert.True(t, builtIn.IsDefault)
}

// TestUpdateClaimMappingPolicyUnsetDefaultFallsBackToBuiltIn checks that unsetting the default flag on a policy makes the built-in policy the default again
func TestUpdateClaimMappingPolicyUnsetDefaultFallsBackToBuiltIn(t *testing.T) {
	service := newTestService(t)

	input := OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Custom", IsDefault: true},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	}
	created, err := service.CreateClaimMappingPolicy(context.Background(), input)
	require.NoError(t, err)

	// The custom policy took over the default flag from the built-in one
	builtIn, err := service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
	require.NoError(t, err)
	require.False(t, builtIn.IsDefault)

	// Unset the default flag on the custom policy
	input.IsDefault = false
	_, err = service.UpdateClaimMappingPolicy(context.Background(), created.ID, input)
	require.NoError(t, err)

	custom, err := service.GetClaimMappingPolicy(context.Background(), created.ID)
	require.NoError(t, err)
	assert.False(t, custom.IsDefault)

	builtIn, err = service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
	require.NoError(t, err)
	assert.True(t, builtIn.IsDefault)
}

// builtInPolicyDto returns the built-in policy as the frontend would send it back on update
func builtInPolicyDto(t *testing.T, service *Service) OidcClaimMappingPolicyDto {
	t.Helper()
	policy, err := service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
	require.NoError(t, err)

	var policyDto OidcClaimMappingPolicyDto
	require.NoError(t, dto.MapStruct(policy, &policyDto))
	return policyDto
}

// TestUpdateBuiltInClaimMappingPolicyOnlyAllowsDefaultFlag checks that the built-in policy inserted by the migrations is read-only except for its default flag
func TestUpdateBuiltInClaimMappingPolicyOnlyAllowsDefaultFlag(t *testing.T) {
	service := newTestService(t)

	t.Run("renaming is rejected", func(t *testing.T) {
		input := builtInPolicyDto(t, service)
		input.Name = "Renamed"

		_, err := service.UpdateClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID, input)
		var appErr *apperror.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperror.CodeBuiltInClaimMappingPolicyUpdate, appErr.Code())
	})

	t.Run("changing the mappings is rejected", func(t *testing.T) {
		input := builtInPolicyDto(t, service)
		input.ClaimMappings = input.ClaimMappings[:1]

		_, err := service.UpdateClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID, input)
		var appErr *apperror.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperror.CodeBuiltInClaimMappingPolicyUpdate, appErr.Code())

		policy, err := service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
		require.NoError(t, err)
		assert.Greater(t, len(policy.ClaimMappings), 1)
	})

	t.Run("unsetting the default flag is rejected", func(t *testing.T) {
		input := builtInPolicyDto(t, service)
		input.IsDefault = false

		_, err := service.UpdateClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID, input)
		var appErr *apperror.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperror.CodeValidationFailed, appErr.Code())

		policy, err := service.GetClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
		require.NoError(t, err)
		assert.True(t, policy.IsDefault)
	})

	t.Run("deleting is rejected even when not default", func(t *testing.T) {
		// Make another policy the default so the built-in one is no longer protected by the default check
		_, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
			OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "New Default", IsDefault: true},
			ClaimMappings: []OidcClaimMappingDto{
				{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
			},
		})
		require.NoError(t, err)

		err = service.DeleteClaimMappingPolicy(context.Background(), BuiltInClaimMappingPolicyID)
		var appErr *apperror.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperror.CodeBuiltInClaimMappingPolicyUpdate, appErr.Code())
	})
}

// Tests for validateClaimMappings

func TestValidateClaimMappingsDuplicateClaims(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldEmail),
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate claim mapping")
}

func TestValidateClaimMappingsReservedClaims(t *testing.T) {
	reservedClaims := []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "azp", "nbf", "jti"}

	for _, claim := range reservedClaims {
		t.Run(claim, func(t *testing.T) {
			err := validateClaimMappings([]OidcClaimMappingDto{
				{
					ClaimName:   "user_id",
					SourceType:  "user_field",
					SourceValue: string(UserFieldID),
				},
				{
					ClaimName:   claim,
					SourceType:  "user_field",
					SourceValue: string(UserFieldEmail),
				},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot remap reserved claim")
			assert.Contains(t, err.Error(), claim)
		})
	}
}

func TestValidateClaimMappingsEmptyFields(t *testing.T) {
	tests := []struct {
		name    string
		mapping OidcClaimMappingDto
	}{
		{
			name:    "empty claim name",
			mapping: OidcClaimMappingDto{SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
		{
			name:    "empty source type",
			mapping: OidcClaimMappingDto{ClaimName: "user_id", SourceValue: string(UserFieldID)},
		},
		{
			name:    "empty source value",
			mapping: OidcClaimMappingDto{ClaimName: "user_id", SourceType: "user_field"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateClaimMappings([]OidcClaimMappingDto{
				{
					ClaimName:   "user_id",
					SourceType:  "user_field",
					SourceValue: string(UserFieldID),
				},
				tc.mapping,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "claim name, source type and source value are required")
		})
	}
}

func TestValidateClaimMappingsInvalidUserField(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: "invalid_field",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user field")
}

func TestValidateClaimMappingsValidUserFields(t *testing.T) {
	validFields := []OidcUserField{
		UserFieldID,
		UserFieldEmail,
		UserFieldEmailVerified,
		UserFieldFirstName,
		UserFieldLastName,
		UserFieldDisplayName,
		UserFieldUsername,
		UserFieldLocale,
		UserFieldPicture,
		UserFieldGroups,
	}

	for _, field := range validFields {
		t.Run(string(field), func(t *testing.T) {
			err := validateClaimMappings([]OidcClaimMappingDto{
				{
					ClaimName:   "user_id",
					SourceType:  "user_field",
					SourceValue: string(UserFieldID),
				},
				{
					ClaimName:   string(field),
					SourceType:  "user_field",
					SourceValue: string(field),
				},
			})
			require.NoError(t, err)
		})
	}
}

func TestValidateClaimMappingsCustomClaim(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "custom_claim",
			SourceType:  "custom_claim",
			SourceValue: "my_custom_key",
		},
	})
	require.NoError(t, err)
}

func TestValidateClaimMappingsCustomClaimInvalidLength(t *testing.T) {
	// Test empty custom claim key
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "custom",
			SourceType:  "custom_claim",
			SourceValue: "",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claim name, source type and source value are required")

	// Test too long custom claim key (256 chars)
	longKey := strings.Repeat("a", 256)
	err = validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "custom",
			SourceType:  "custom_claim",
			SourceValue: longKey,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid custom claim key length")
}

func TestValidateClaimMappingsStaticSource(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "static_claim",
			SourceType:  "static",
			SourceValue: `{"role": "admin"}`,
		},
	})
	require.NoError(t, err)
}

func TestValidateClaimMappingsInvalidSourceType(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
		},
		{
			ClaimName:   "invalid",
			SourceType:  "invalid_type",
			SourceValue: "some_value",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source type")
}

func TestValidateClaimMappingsValidComplexPolicy(t *testing.T) {
	err := validateClaimMappings([]OidcClaimMappingDto{
		{
			ClaimName:   "user_id",
			SourceType:  "user_field",
			SourceValue: string(UserFieldID),
			Scope:       []string{"openid"},
			AccessToken: true,
			IDToken:     true,
		},
		{
			ClaimName:   "name",
			SourceType:  "user_field",
			SourceValue: string(UserFieldDisplayName),
			Scope:       []string{"profile"},
			AccessToken: true,
			IDToken:     true,
		},
		{
			ClaimName:   "email",
			SourceType:  "user_field",
			SourceValue: string(UserFieldEmail),
			Scope:       []string{"email"},
			AccessToken: false,
			IDToken:     true,
		},
		{
			ClaimName:   "custom_department",
			SourceType:  "custom_claim",
			SourceValue: "department",
			Scope:       []string{"profile"},
			AccessToken: true,
			IDToken:     true,
		},
		{
			ClaimName:   "app_version",
			SourceType:  "static",
			SourceValue: "1.0.0",
		},
	})
	require.NoError(t, err)
}

func TestListClaimMappingPolicyWithData(t *testing.T) {
	service := newTestService(t)

	// Create multiple policies
	for i := 0; i < 5; i++ {
		dto := OidcClaimMappingPolicyDto{
			OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{
				Name:      "Policy " + string(rune('A'+i)),
				IsDefault: i == 0,
			},
			ClaimMappings: []OidcClaimMappingDto{
				{
					ClaimName:   "user_id",
					SourceType:  "user_field",
					SourceValue: string(UserFieldID),
				},
			},
		}
		_, err := service.CreateClaimMappingPolicy(context.Background(), dto)
		require.NoError(t, err)
	}

	// List all policies
	policies, pagination, err := service.ListClaimMappingPolicy(context.Background(), "", utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.Len(t, policies, 6)
	assert.Equal(t, int64(6), pagination.TotalItems)
}

// Tests for ListClientsByClaimMappingPolicy

func TestListClientsByClaimMappingPolicy(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Attached"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	otherPolicy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Other"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "Grafana", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-b"}, Name: "Nextcloud", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	// Assigned to another policy, must not show up
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-c"}, Name: "Gitea", ClaimMappingPolicyId: &otherPolicy.ID,
	}).Error)
	// Assigned to no policy at all, must not show up either
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-d"}, Name: "Immich",
	}).Error)

	clients, pagination, err := service.ListClientsByClaimMappingPolicy(
		context.Background(), policy.ID, "", utils.ListRequestOptions{},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, pagination.TotalItems)
	require.Len(t, clients, 2)

	names := []string{clients[0].Name, clients[1].Name}
	assert.ElementsMatch(t, []string{"Grafana", "Nextcloud"}, names)
}

func TestListClientsByClaimMappingPolicyFiltersBySearchTerm(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Attached"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "Grafana", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-b"}, Name: "Nextcloud", ClaimMappingPolicyId: &policy.ID,
	}).Error)

	clients, pagination, err := service.ListClientsByClaimMappingPolicy(
		context.Background(), policy.ID, "next", utils.ListRequestOptions{},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, pagination.TotalItems)
	require.Len(t, clients, 1)
	assert.Equal(t, "Nextcloud", clients[0].Name)
}

// TestListClientsByClaimMappingPolicyEscapesWildcards checks that LIKE wildcards in the search term are matched literally
func TestListClientsByClaimMappingPolicyEscapesWildcards(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Attached"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "100% Grafana", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-b"}, Name: "Next_cloud", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-c"}, Name: "Nextcloud", ClaimMappingPolicyId: &policy.ID,
	}).Error)

	// A bare % would match every client if it were not escaped
	clients, _, err := service.ListClientsByClaimMappingPolicy(context.Background(), policy.ID, "%", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "100% Grafana", clients[0].Name)

	// An unescaped _ would also match "Nextcloud" since it stands for any single character
	clients, _, err = service.ListClientsByClaimMappingPolicy(context.Background(), policy.ID, "t_c", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "Next_cloud", clients[0].Name)
}

func TestListClientsByClaimMappingPolicyPaginates(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy, err := service.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: "Attached"},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)

	for _, name := range []string{"Client A", "Client B", "Client C"} {
		require.NoError(t, db.Create(&model.OidcClient{
			Base: model.Base{ID: name}, Name: name, ClaimMappingPolicyId: &policy.ID,
		}).Error)
	}

	options := utils.ListRequestOptions{}
	options.Pagination.Page = 2
	options.Pagination.Limit = 2

	clients, pagination, err := service.ListClientsByClaimMappingPolicy(
		context.Background(), policy.ID, "", options,
	)
	require.NoError(t, err)
	assert.EqualValues(t, 3, pagination.TotalItems)
	assert.EqualValues(t, 2, pagination.TotalPages)
	assert.Len(t, clients, 1)
}

func TestListClientsByClaimMappingPolicyNotFound(t *testing.T) {
	service := newTestService(t)

	_, _, err := service.ListClientsByClaimMappingPolicy(
		context.Background(), "nonexistent", "", utils.ListRequestOptions{},
	)
	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

// Tests for assigning and detaching clients

// newPolicyWithClients creates a policy plus the given clients, each attached to the policy passed
// as its value (nil for a client on no policy at all).
func newPolicyWithClients(t *testing.T, svc *Service, name string) *OidcClaimMappingPolicyDto {
	t.Helper()
	policy, err := svc.CreateClaimMappingPolicy(context.Background(), OidcClaimMappingPolicyDto{
		OidcClaimMappingPolicyMetadataDto: OidcClaimMappingPolicyMetadataDto{Name: name},
		ClaimMappings: []OidcClaimMappingDto{
			{ClaimName: "user_id", SourceType: "user_field", SourceValue: string(UserFieldID)},
		},
	})
	require.NoError(t, err)
	return policy
}

func TestListAssignableClientsExcludesClientsAlreadyOnThePolicy(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy := newPolicyWithClients(t, service, "Target")
	otherPolicy := newPolicyWithClients(t, service, "Other")

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-on-policy"}, Name: "Grafana", ClaimMappingPolicyId: &policy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-on-other"}, Name: "Gitea", ClaimMappingPolicyId: &otherPolicy.ID,
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-unassigned"}, Name: "Immich",
	}).Error)

	clients, pagination, err := service.ListAssignableClients(
		context.Background(), policy.ID, "", utils.ListRequestOptions{},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, pagination.TotalItems)

	names := make([]string, len(clients))
	for i, client := range clients {
		names[i] = client.Name
	}
	// A client on another policy can still be moved onto this one; only the ones already here are hidden
	assert.ElementsMatch(t, []string{"Gitea", "Immich"}, names)
}

func TestAssignClientToClaimMappingPolicy(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy := newPolicyWithClients(t, service, "Target")
	otherPolicy := newPolicyWithClients(t, service, "Other")

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "Grafana", ClaimMappingPolicyId: &otherPolicy.ID,
	}).Error)

	err := service.AssignClientToClaimMappingPolicy(context.Background(), policy.ID, "client-a")
	require.NoError(t, err)

	var client model.OidcClient
	require.NoError(t, db.First(&client, "id = ?", "client-a").Error)
	require.NotNil(t, client.ClaimMappingPolicyId)
	assert.Equal(t, policy.ID, *client.ClaimMappingPolicyId)
}

func TestAssignClientToClaimMappingPolicyUnknownClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy := newPolicyWithClients(t, service, "Target")

	err := service.AssignClientToClaimMappingPolicy(context.Background(), policy.ID, "nonexistent")
	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func TestAssignClientToClaimMappingPolicyUnknownPolicy(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-a"}, Name: "Grafana"}).Error)

	err := service.AssignClientToClaimMappingPolicy(context.Background(), "nonexistent", "client-a")
	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func TestRemoveClientFromClaimMappingPolicy(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy := newPolicyWithClients(t, service, "Target")

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "Grafana", ClaimMappingPolicyId: &policy.ID,
	}).Error)

	err := service.RemoveClientFromClaimMappingPolicy(context.Background(), policy.ID, "client-a")
	require.NoError(t, err)

	var client model.OidcClient
	require.NoError(t, db.First(&client, "id = ?", "client-a").Error)
	assert.Nil(t, client.ClaimMappingPolicyId)
}

// A client that has since been moved to another policy must not be detached by a stale request
func TestRemoveClientFromClaimMappingPolicyLeavesClientsOfOtherPolicies(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	policy := newPolicyWithClients(t, service, "Target")
	otherPolicy := newPolicyWithClients(t, service, "Other")

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-a"}, Name: "Grafana", ClaimMappingPolicyId: &otherPolicy.ID,
	}).Error)

	err := service.RemoveClientFromClaimMappingPolicy(context.Background(), policy.ID, "client-a")
	require.Error(t, err)
	assert.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	var client model.OidcClient
	require.NoError(t, db.First(&client, "id = ?", "client-a").Error)
	require.NotNil(t, client.ClaimMappingPolicyId)
	assert.Equal(t, otherPolicy.ID, *client.ClaimMappingPolicyId)
}
