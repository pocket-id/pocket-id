package feature

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeAppConfig struct {
	cfg *appconfig.AppConfigModel
}

func (f *fakeAppConfig) GetConfig(context.Context) (*appconfig.AppConfigModel, error) {
	return f.cfg, nil
}

func (f *fakeAppConfig) UpdateAppConfigValues(_ context.Context, keysAndValues ...string) error {
	f.cfg.EnabledFeatures = appconfig.AppConfigValue(keysAndValues[1])
	return nil
}

func TestListCombinesEnabledFeaturesWithUsage(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.EnabledFeatures = `["apiKeys","unknown"]`
	cfg.SignupDefaultCustomClaims = `[{"key":"team","value":"photo"}]`
	cfg.LdapEnabled = "true"
	svc := New(Dependencies{DB: db, AppConfig: &fakeAppConfig{cfg: cfg}}).service

	require.NoError(t, db.Create(&api.API{Name: "Orders", Audience: "https://orders.example.com"}).Error)
	userID := "user-1"
	require.NoError(t, db.Create(&model.CustomClaim{Key: "level", Value: "1", UserID: &userID}).Error)

	require.NoError(t, claimmappingpolicy.Bootstrap(t.Context(), db))
	require.NoError(t, db.Create(&claimmappingpolicy.OidcClaimMappingPolicy{Name: "Roles"}).Error)

	require.NoError(t, db.Create(&model.OidcClient{
		Base:        model.Base{ID: "client-1"},
		Name:        "Grafana",
		Credentials: model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{{Issuer: "https://ci.example.com"}}},
	}).Error)

	statuses, err := svc.List(t.Context())
	require.NoError(t, err)
	assert.Equal(t, map[Feature]Status{
		LDAP:                 {Enabled: true, InUse: true},
		SCIM:                 {Enabled: false, InUse: false},
		CustomClaims:         {Enabled: true, InUse: true},
		APIs:                 {Enabled: true, InUse: true},
		APIKeys:              {Enabled: true, InUse: false},
		CIMD:                 {Enabled: false, InUse: false},
		FederatedCredentials: {Enabled: true, InUse: true},
		ClaimMappings:        {Enabled: true, InUse: true},
	}, statuses)

	// A metadata document client alone puts CIMD in use, and its own federated identities do not count
	require.NoError(t, db.Create(&model.OidcClient{
		Base:        model.Base{ID: "https://app.example.com/oauth-client.json"},
		Name:        "Example app",
		ClientType:  model.OidcClientTypeCIMD,
		Credentials: model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{{Issuer: "https://app.example.com"}}},
	}).Error)
	// Clients created before the credentials column existed still have NULL there
	require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", "client-1").Update("credentials", gorm.Expr("NULL")).Error)
	statuses, err = svc.List(t.Context())
	require.NoError(t, err)
	assert.Equal(t, Status{Enabled: true, InUse: true}, statuses[CIMD])
	assert.Equal(t, Status{Enabled: false, InUse: false}, statuses[FederatedCredentials])
}

func TestListSavesFeaturesInUseOnce(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	svc := New(Dependencies{DB: db, AppConfig: &fakeAppConfig{cfg: cfg}}).service

	orders := api.API{Name: "Orders", Audience: "https://orders.example.com"}
	require.NoError(t, db.Create(&orders).Error)
	require.NoError(t, claimmappingpolicy.Bootstrap(t.Context(), db))
	_, err := svc.List(t.Context())
	require.NoError(t, err)
	assert.JSONEq(t, `["apis"]`, cfg.EnabledFeatures.String())

	// Deleting the last API leaves the feature on
	require.NoError(t, db.Delete(&orders).Error)
	statuses, err := svc.List(t.Context())
	require.NoError(t, err)
	assert.Equal(t, Status{Enabled: true, InUse: false}, statuses[APIs])
}
