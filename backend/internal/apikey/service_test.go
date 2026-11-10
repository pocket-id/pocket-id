package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type nopAuditLogger struct{}

func (nopAuditLogger) Record(context.Context, auditlogs.Event, auditlogs.Origin, string, auditlogs.Data, *gorm.DB) {
}

func TestMissingManagedAPIKeyReturnsNotFound(t *testing.T) {
	service, err := newService(t.Context(), testutils.NewDatabaseForTest(t), "", nopAuditLogger{})
	require.NoError(t, err)

	err = service.RevokeApiKey(t.Context(), "user-id", "missing-key", auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))

	_, _, err = service.RenewApiKey(t.Context(), "user-id", "missing-key", time.Now().Add(time.Hour), 0, auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))

	_, err = service.UpdateApiKey(t.Context(), "user-id", "missing-key", authz.NewScopeSet(authz.AccountRead), apiKeyUpdateDto{Name: "key", Scopes: datatype.StringList{"account:read"}})
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))
}

func TestAPIKeyScopes(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service, err := newService(t.Context(), db, "", nopAuditLogger{})
	require.NoError(t, err)

	user := model.User{Username: "scoped-key-owner", FirstName: "Scoped", DisplayName: "Scoped"}
	require.NoError(t, db.Create(&user).Error)
	grantable := authz.NewScopeSet(authz.AccountRead, authz.AccountWrite)

	t.Run("create rejects scopes outside grantable", func(t *testing.T) {
		for _, scope := range []authz.Scope{authz.UsersRead, authz.AccountAPIKeysCreate} {
			_, _, err := service.CreateApiKey(t.Context(), user.ID, grantable, apiKeyCreateDto{
				Name:      "rejected",
				ExpiresAt: datatype.DateTime(time.Now().Add(time.Hour)),
				Scopes:    datatype.StringList{string(authz.AccountRead), string(scope)},
			}, auditlogs.Origin{})
			require.True(t, apperror.IsCode(err, apperror.CodeInvalidAPIKeyScope), scope)
		}
	})

	t.Run("update changes scopes and keeps the token", func(t *testing.T) {
		created, _, err := service.CreateApiKey(t.Context(), user.ID, grantable, apiKeyCreateDto{
			Name:      "scoped",
			ExpiresAt: datatype.DateTime(time.Now().Add(time.Hour)),
			Scopes:    datatype.StringList{string(authz.AccountWrite), string(authz.AccountRead), string(authz.AccountRead)},
		}, auditlogs.Origin{})
		require.NoError(t, err)
		require.Equal(t, datatype.StringList{"account:read", "account:write"}, created.Scopes)

		_, err = service.UpdateApiKey(t.Context(), user.ID, created.ID, grantable, apiKeyUpdateDto{Name: "scoped", Scopes: datatype.StringList{string(authz.UsersWrite)}})
		require.True(t, apperror.IsCode(err, apperror.CodeInvalidAPIKeyScope))

		updated, err := service.UpdateApiKey(t.Context(), user.ID, created.ID, grantable, apiKeyUpdateDto{Name: "renamed", Scopes: datatype.StringList{string(authz.AccountRead)}})
		require.NoError(t, err)

		var stored ApiKey
		require.NoError(t, db.First(&stored, "id = ?", created.ID).Error)
		require.Equal(t, "renamed", stored.Name)
		require.Equal(t, datatype.StringList{"account:read"}, stored.Scopes)
		require.Equal(t, created.Key, stored.Key)
		require.Equal(t, updated.Scopes, stored.Scopes)
	})
}

func TestRenewApiKeyRotation(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service, err := newService(t.Context(), db, "", nopAuditLogger{})
	require.NoError(t, err)

	user := model.User{Username: "rotating-key-owner", FirstName: "Rotating", DisplayName: "Rotating"}
	require.NoError(t, db.Create(&user).Error)

	createKey := func(t *testing.T, expiresAt time.Time) (ApiKey, string) {
		t.Helper()
		key, token, err := service.CreateApiKey(t.Context(), user.ID, authz.NewScopeSet(authz.AccountRead), apiKeyCreateDto{
			Name:      "rotating",
			ExpiresAt: datatype.DateTime(expiresAt),
			Scopes:    datatype.StringList{string(authz.AccountRead)},
		}, auditlogs.Origin{})
		require.NoError(t, err)
		return key, token
	}
	requireValid := func(t *testing.T, token string, valid bool) {
		t.Helper()
		_, err := service.ValidateApiKey(t.Context(), token)
		if valid {
			require.NoError(t, err)
		} else {
			require.True(t, apperror.IsCode(err, apperror.CodeInvalidAPIKey))
		}
	}
	newExpiry := time.Now().Add(90 * 24 * time.Hour)

	t.Run("without a grace period the old token stops working", func(t *testing.T) {
		key, oldToken := createKey(t, time.Now().Add(time.Hour))

		_, newToken, err := service.RenewApiKey(t.Context(), user.ID, key.ID, newExpiry, 0, auditlogs.Origin{})
		require.NoError(t, err)

		requireValid(t, oldToken, false)
		requireValid(t, newToken, true)
	})

	t.Run("the old token works during the grace period, capped at its expiration", func(t *testing.T) {
		oldExpiry := time.Now().Add(time.Hour)
		key, oldToken := createKey(t, oldExpiry)

		renewed, newToken, err := service.RenewApiKey(t.Context(), user.ID, key.ID, newExpiry, 24*time.Hour, auditlogs.Origin{})
		require.NoError(t, err)
		require.WithinDuration(t, oldExpiry, renewed.PreviousKeyExpiresAt.ToTime(), time.Second)

		requireValid(t, oldToken, true)
		requireValid(t, newToken, true)

		// Rotating again replaces the token that is still in its grace period
		_, newestToken, err := service.RenewApiKey(t.Context(), user.ID, key.ID, newExpiry, 24*time.Hour, auditlogs.Origin{})
		require.NoError(t, err)
		requireValid(t, oldToken, false)
		requireValid(t, newToken, true)
		requireValid(t, newestToken, true)
	})

	t.Run("the old token stops working once the grace period ends", func(t *testing.T) {
		key, oldToken := createKey(t, time.Now().Add(time.Hour))

		_, _, err := service.RenewApiKey(t.Context(), user.ID, key.ID, newExpiry, time.Hour, auditlogs.Origin{})
		require.NoError(t, err)
		require.NoError(t, db.Model(&ApiKey{}).Where("id = ?", key.ID).Update("previous_key_expires_at", datatype.DateTime(time.Now().Add(-time.Minute))).Error)

		requireValid(t, oldToken, false)
	})

	t.Run("renewing an expired key ignores the grace period", func(t *testing.T) {
		key, oldToken := createKey(t, time.Now().Add(time.Hour))
		require.NoError(t, db.Model(&ApiKey{}).Where("id = ?", key.ID).Update("expires_at", datatype.DateTime(time.Now().Add(-time.Hour))).Error)

		renewed, newToken, err := service.RenewApiKey(t.Context(), user.ID, key.ID, newExpiry, 24*time.Hour, auditlogs.Origin{})
		require.NoError(t, err)
		require.Nil(t, renewed.PreviousKey)

		requireValid(t, oldToken, false)
		requireValid(t, newToken, true)
	})
}
