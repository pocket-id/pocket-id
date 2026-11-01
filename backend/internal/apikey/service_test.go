package apikey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestMissingManagedAPIKeyReturnsNotFound(t *testing.T) {
	service, err := newService(t.Context(), testutils.NewDatabaseForTest(t), "")
	require.NoError(t, err)

	err = service.RevokeApiKey(t.Context(), "user-id", "missing-key")
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))

	_, _, err = service.RenewApiKey(t.Context(), "user-id", "missing-key", time.Now().Add(time.Hour))
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))

	_, err = service.UpdateApiKey(t.Context(), "user-id", "missing-key", authz.NewScopeSet(authz.AccountRead), apiKeyUpdateDto{Name: "key", Scopes: datatype.StringList{"account:read"}})
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))
}

func TestAPIKeyScopes(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service, err := newService(t.Context(), db, "")
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
			})
			require.True(t, apperror.IsCode(err, apperror.CodeInvalidAPIKeyScope), scope)
		}
	})

	t.Run("update changes scopes and keeps the token", func(t *testing.T) {
		created, _, err := service.CreateApiKey(t.Context(), user.ID, grantable, apiKeyCreateDto{
			Name:      "scoped",
			ExpiresAt: datatype.DateTime(time.Now().Add(time.Hour)),
			Scopes:    datatype.StringList{string(authz.AccountWrite), string(authz.AccountRead), string(authz.AccountRead)},
		})
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
