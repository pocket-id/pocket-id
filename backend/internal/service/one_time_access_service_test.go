package service

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

func TestExchangeOneTimeAccessTokenRejectsDisabledUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	appConfig := appconfig.NewTestAppConfigService(nil)
	instanceID := newInstanceID(t, db)
	jwtService := initJwtService(t, db, instanceID, appConfig, newTestEnvConfig())
	auditLogService := NewAuditLogService(db, nil, &GeoLiteService{})
	oneTimeAccessService := NewOneTimeAccessService(db, nil, jwtService, auditLogService, nil)

	user := model.User{
		Base:     model.Base{ID: "disabled-user"},
		Username: "disabled-user",
		Disabled: true,
	}
	require.NoError(t, db.Create(&user).Error)

	loginCode := model.OneTimeAccessToken{
		Base:      model.Base{ID: "disabled-user-login-code"},
		Token:     "ABCDEF",
		ExpiresAt: datatype.DateTime(time.Now().Add(time.Minute)),
		UserID:    user.ID,
	}
	require.NoError(t, db.Create(&loginCode).Error)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := oneTimeAccessService.ExchangeOneTimeAccessToken(t.Context(), dbConfig, loginCode.Token, "", "", "")

	var userDisabledErr *common.UserDisabledError
	require.ErrorAs(t, err, &userDisabledErr)
	require.Empty(t, exchangedUser.ID)
	require.Empty(t, accessToken)

	var remainingLoginCode model.OneTimeAccessToken
	require.NoError(t, db.Where("token = ?", loginCode.Token).First(&remainingLoginCode).Error)

	var auditLogCount int64
	require.NoError(t, db.Model(&model.AuditLog{}).Where("user_id = ?", user.ID).Count(&auditLogCount).Error)
	require.Zero(t, auditLogCount)
}
