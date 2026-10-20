//go:build unit

package claimmappingpolicy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// forEachDatabase runs the test on SQLite and, when a test server is configured, on Postgres, since the upsert relies on engine specific SQL
func forEachDatabase(t *testing.T, test func(t *testing.T, db *gorm.DB)) {
	t.Run("sqlite", func(t *testing.T) {
		test(t, testutils.NewDatabaseForTest(t))
	})
	t.Run("postgres", func(t *testing.T) {
		test(t, testutils.NewPostgresDatabaseForTest(t))
	})
}

func TestBootstrapInsertsBuiltInPolicyAsDefault(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db))

		var policy OidcClaimMappingPolicy
		require.NoError(t, db.First(&policy, "id = ?", BuiltInClaimMappingPolicyID).Error)
		require.Equal(t, builtInClaimMappingPolicyName, policy.Name)
		require.True(t, policy.IsDefault)
		require.Equal(t, builtInClaimMappings(), policy.ClaimMappings)
	})
}

func TestBootstrapIsIdempotent(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db))
		require.NoError(t, Bootstrap(t.Context(), db))

		var count int64
		require.NoError(t, db.Model(&OidcClaimMappingPolicy{}).Count(&count).Error)
		require.EqualValues(t, 1, count)
	})
}

func TestBootstrapRefreshesBuiltInPolicyMappings(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db))

		// Simulate a row written by an older version
		require.NoError(t, db.Model(&OidcClaimMappingPolicy{}).Where("id = ?", BuiltInClaimMappingPolicyID).Updates(map[string]any{
			"name":           "Old name",
			"claim_mappings": OidcClaimMappings{{ClaimName: "legacy", SourceType: MappingSourceStatic, SourceValue: "x"}},
		}).Error)

		require.NoError(t, Bootstrap(t.Context(), db))

		var policy OidcClaimMappingPolicy
		require.NoError(t, db.First(&policy, "id = ?", BuiltInClaimMappingPolicyID).Error)
		require.Equal(t, builtInClaimMappingPolicyName, policy.Name)
		require.Equal(t, builtInClaimMappings(), policy.ClaimMappings)
	})
}

func TestBootstrapKeepsAnotherDefaultPolicy(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db))

		// The admin promotes a custom policy to default
		custom := OidcClaimMappingPolicy{Base: model.Base{ID: "11111111-1111-1111-1111-111111111111"}, Name: "Custom", IsDefault: true, ClaimMappings: OidcClaimMappings{}}
		require.NoError(t, db.Model(&OidcClaimMappingPolicy{}).Where("id = ?", BuiltInClaimMappingPolicyID).Update("is_default", false).Error)
		require.NoError(t, db.Create(&custom).Error)

		require.NoError(t, Bootstrap(t.Context(), db))

		var builtIn OidcClaimMappingPolicy
		require.NoError(t, db.First(&builtIn, "id = ?", BuiltInClaimMappingPolicyID).Error)
		require.False(t, builtIn.IsDefault)

		var reloaded OidcClaimMappingPolicy
		require.NoError(t, db.First(&reloaded, "id = ?", custom.ID).Error)
		require.True(t, reloaded.IsDefault)
	})
}

func TestBootstrapRestoresDefaultWhenNoneExists(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db))

		// Lose the default flag, so clients without a policy would get no claims
		require.NoError(t, db.Model(&OidcClaimMappingPolicy{}).Where("id = ?", BuiltInClaimMappingPolicyID).Update("is_default", false).Error)

		require.NoError(t, Bootstrap(t.Context(), db))

		var builtIn OidcClaimMappingPolicy
		require.NoError(t, db.First(&builtIn, "id = ?", BuiltInClaimMappingPolicyID).Error)
		require.True(t, builtIn.IsDefault)
	})
}
