package service

import (
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// newOneTimeAccessServiceForTest sets up a OneTimeAccessService backed by an in-memory test actor host and returns both.
func newOneTimeAccessServiceForTest(t *testing.T, db *gorm.DB) (*OneTimeAccessService, *local.Host) {
	t.Helper()

	appConfig := appconfig.NewTestAppConfigService(nil)
	instanceID := newInstanceID(t, db)
	jwtService := initJwtService(t, db, instanceID, appConfig, newTestEnvConfig())
	auditLogService := NewAuditLogService(db, nil, &GeoLiteService{}, appConfig)

	var svc *OneTimeAccessService
	host := testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		svc, err = NewOneTimeAccessService(h, db, nil, jwtService, auditLogService, nil)
		require.NoError(t, err)
	})
	require.NotNil(t, svc)

	return svc, host
}

func TestExchangeOneTimeAccessTokenSuccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	oneTimeAccessService, host := newOneTimeAccessServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "enabled-user"},
		Username: "enabled-user",
	}
	require.NoError(t, db.Create(&user).Error)

	token, _, err := StoreOneTimeAccessToken(t.Context(), oneTimeAccessService.actorService, user.ID, time.Minute, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := oneTimeAccessService.ExchangeOneTimeAccessToken(t.Context(), dbConfig, token, "", "1.2.3.4", "test-agent")
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.NotEmpty(t, accessToken)

	// The token must have been consumed
	var state oneTimeAccessTokenState
	err = host.GetState(t.Context(), OneTimeAccessTokenActorType, token, &state)
	require.ErrorIs(t, err, actor.ErrStateNotFound)

	// A sign-in audit log must have been created
	var auditLogCount int64
	require.NoError(t, db.Model(&model.AuditLog{}).
		Where("user_id = ? AND event = ?", user.ID, model.AuditLogEventOneTimeAccessTokenSignIn).
		Count(&auditLogCount).Error)
	require.Equal(t, int64(1), auditLogCount)
}

func TestExchangeOneTimeAccessTokenInvalidToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	oneTimeAccessService, _ := newOneTimeAccessServiceForTest(t, db)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err := oneTimeAccessService.ExchangeOneTimeAccessToken(t.Context(), dbConfig, "does-not-exist", "", "", "")

	var invalidErr *common.TokenInvalidOrExpiredError
	require.ErrorAs(t, err, &invalidErr)
}

func TestExchangeOneTimeAccessTokenDeviceMismatch(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	oneTimeAccessService, host := newOneTimeAccessServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "device-user"},
		Username: "device-user",
	}
	require.NoError(t, db.Create(&user).Error)

	// Store a token that requires a device token
	token, deviceToken, err := StoreOneTimeAccessToken(t.Context(), oneTimeAccessService.actorService, user.ID, time.Minute, true)
	require.NoError(t, err)
	require.NotNil(t, deviceToken)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err = oneTimeAccessService.ExchangeOneTimeAccessToken(t.Context(), dbConfig, token, "wrong-device-token", "", "")

	var deviceErr *common.DeviceCodeInvalid
	require.ErrorAs(t, err, &deviceErr)

	// The token must not have been consumed on a device-token mismatch
	var state oneTimeAccessTokenState
	err = host.GetState(t.Context(), OneTimeAccessTokenActorType, token, &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)
}

func TestExchangeOneTimeAccessTokenRejectsDisabledUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	oneTimeAccessService, host := newOneTimeAccessServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "disabled-user"},
		Username: "disabled-user",
		Disabled: true,
	}
	require.NoError(t, db.Create(&user).Error)

	// Store a one-time access token for the disabled user in the actor state store
	token, _, err := StoreOneTimeAccessToken(t.Context(), oneTimeAccessService.actorService, user.ID, time.Minute, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := oneTimeAccessService.ExchangeOneTimeAccessToken(t.Context(), dbConfig, token, "", "", "")

	var userDisabledErr *common.UserDisabledError
	require.ErrorAs(t, err, &userDisabledErr)
	require.Empty(t, exchangedUser.ID)
	require.Empty(t, accessToken)

	// The token must have been restored (not consumed), since the exchange failed because the user is disabled
	var state oneTimeAccessTokenState
	err = host.GetState(t.Context(), OneTimeAccessTokenActorType, token, &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)

	var auditLogCount int64
	require.NoError(t, db.Model(&model.AuditLog{}).Where("user_id = ?", user.ID).Count(&auditLogCount).Error)
	require.Zero(t, auditLogCount)
}
