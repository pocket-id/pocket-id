package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestCreateReauthenticationTokenWithAccessToken(t *testing.T) {
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"},
	})

	setupService := func(t *testing.T) (*WebAuthnService, model.User) {
		t.Helper()

		jwtService, db, _ := setupJwtService(t, mockConfig)
		user := model.User{
			Base:     model.Base{ID: "reauth-user"},
			Username: "reauth-user",
		}
		require.NoError(t, db.Create(&user).Error)

		return &WebAuthnService{
			db:         db,
			jwtService: jwtService,
		}, user
	}

	t.Run("accepts a fresh access token from WebAuthn login", func(t *testing.T) {
		service, user := setupService(t)
		accessToken, err := service.jwtService.GenerateAccessToken(user, AuthenticationMethodPhishingResistant)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		require.NoError(t, err)
		assert.NotEmpty(t, reauthenticationToken)
	})

	t.Run("rejects a fresh access token from one-time access login", func(t *testing.T) {
		service, user := setupService(t)
		accessToken, err := service.jwtService.GenerateAccessToken(user, AuthenticationMethodOneTimePassword)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.ReauthenticationRequiredError))
	})

	t.Run("rejects a fresh access token without an authentication method", func(t *testing.T) {
		service, user := setupService(t)
		accessToken, err := service.jwtService.GenerateAccessToken(user, "")
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.ReauthenticationRequiredError))
	})
}

func TestConsumeReauthenticationTokenReturnsTokenCreationTime(t *testing.T) {
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"},
	})
	jwtService, db, _ := setupJwtService(t, mockConfig)
	service := &WebAuthnService{
		db:         db,
		jwtService: jwtService,
	}

	const (
		userID = "reauth-user"
		token  = "reauthentication-token"
	)
	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&model.ReauthenticationToken{
		Token:     utils.CreateSha256Hash(token),
		ExpiresAt: datatype.DateTime(time.Now().Add(time.Minute)),
		UserID:    userID,
	}).Error)

	var storedToken model.ReauthenticationToken
	require.NoError(t, db.First(&storedToken, "user_id = ?", userID).Error)

	tx := db.Begin()
	reauthenticatedAt, err := service.ConsumeReauthenticationToken(t.Context(), tx, token, userID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)

	require.Equal(t, storedToken.CreatedAt.UTC(), reauthenticatedAt)
}
