package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestClearInactiveDynamicClients(t *testing.T) {
	staleID := "https://stale.example/cimd"
	freshID := "https://fresh.example/cimd"
	standardID := "standard-client"

	// seed inserts one dynamic client whose metadata expired 7 months ago, one that
	// expired last month, and one standard client, then returns a fresh job handle.
	seed := func(t *testing.T, retention string) *DbCleanupJobs {
		t.Helper()

		// GetDynamicClientRetention reads from the in-memory env config, which
		// GetConfig only returns when the UI config is disabled.
		original := common.EnvConfig
		t.Cleanup(func() { common.EnvConfig = original })
		common.EnvConfig.UiConfigDisabled = true

		db := testutils.NewDatabaseForTest(t)

		now := time.Now()
		staleExpiry := datatype.DateTime(now.AddDate(0, -7, 0))
		freshExpiry := datatype.DateTime(now.AddDate(0, -1, 0))
		clients := []model.OidcClient{
			{Base: model.Base{ID: staleID}, Name: "stale", ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &staleExpiry},
			{Base: model.Base{ID: freshID}, Name: "fresh", ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &freshExpiry},
			{Base: model.Base{ID: standardID}, Name: "standard", ClientType: model.OidcClientTypeStandard},
		}
		require.NoError(t, db.Create(&clients).Error)

		cfg := appconfig.NewTestConfig(nil)
		cfg.DynamicClientRetentionDays = appconfig.AppConfigValue(retention)
		appConfigService := appconfig.NewTestAppConfigService(cfg)

		return &DbCleanupJobs{db: db, appConfigService: appConfigService}
	}

	remainingIDs := func(t *testing.T, j *DbCleanupJobs) map[string]bool {
		t.Helper()
		var remaining []model.OidcClient
		require.NoError(t, j.db.Find(&remaining).Error)
		ids := make(map[string]bool, len(remaining))
		for _, c := range remaining {
			ids[c.ID] = true
		}
		return ids
	}

	t.Run("deletes only inactive dynamic clients", func(t *testing.T) {
		j := seed(t, "180")
		require.NoError(t, j.clearInactiveDynamicClients(t.Context()))

		ids := remainingIDs(t, j)
		require.False(t, ids[staleID], "stale dynamic client should be deleted")
		require.True(t, ids[freshID], "fresh dynamic client should be kept")
		require.True(t, ids[standardID], "standard client should never be deleted")
	})

	t.Run("retention of 0 disables the cleanup", func(t *testing.T) {
		j := seed(t, "0")
		require.NoError(t, j.clearInactiveDynamicClients(t.Context()))

		ids := remainingIDs(t, j)
		require.True(t, ids[staleID])
		require.True(t, ids[freshID])
		require.True(t, ids[standardID])
	})

	t.Run("malformed retention disables the cleanup", func(t *testing.T) {
		j := seed(t, "not-a-duration")
		require.NoError(t, j.clearInactiveDynamicClients(t.Context()))

		require.True(t, remainingIDs(t, j)[staleID], "cleanup must not run for an unparseable retention")
	})
}
