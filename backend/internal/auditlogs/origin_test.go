package auditlogs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestRecordDescribesTheOrigin(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "owner"}, Username: "owner"}).Error)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "admin"}, Username: "administrator"}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "provisioner"}, Name: "Provisioner"}).Error)
	require.NoError(t, db.Exec(`INSERT INTO api_keys (id, created_at, name, key, expires_at, user_id) VALUES ('terraform', ?, 'Terraform', 'hash', ?, 'admin')`, "2026-01-01 00:00:00", "2099-01-01 00:00:00").Error)
	s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})

	record := func(t *testing.T, origin Origin, userID string) AuditLog {
		t.Helper()
		s.Record(t.Context(), EventPasskeyRemoved, origin, userID, nil, nil)
		var entry AuditLog
		require.NoError(t, db.Order("created_at DESC").First(&entry).Error)
		require.NoError(t, db.Delete(&entry).Error)
		return entry
	}

	t.Run("user acting on their own account", func(t *testing.T) {
		entry := record(t, Origin{Principal: authz.Principal{Kind: authz.KindSession, UserID: "owner"}}, "owner")
		assert.Equal(t, [3]string{"user", "owner", "owner"}, actor(entry))
		assert.Equal(t, Data{"via": "session"}, entry.Data)
	})

	t.Run("user acting on someone else's account with an API key", func(t *testing.T) {
		entry := record(t, Origin{Principal: authz.Principal{Kind: authz.KindAPIKey, UserID: "admin", APIKeyID: "terraform"}}, "owner")
		assert.Equal(t, [3]string{"user", "admin", "administrator"}, actor(entry))
		assert.Equal(t, Data{"via": "api_key", "viaName": "Terraform"}, entry.Data)
	})

	t.Run("user acting through an app over MCP", func(t *testing.T) {
		entry := record(t, Origin{Principal: authz.Principal{Kind: authz.KindOAuthUser, UserID: "owner", ClientID: "provisioner"}, MCP: true}, "owner")
		assert.Equal(t, Data{"via": "mcp", "viaName": "Provisioner"}, entry.Data)
	})

	t.Run("client acting as itself on no user", func(t *testing.T) {
		entry := record(t, Origin{Principal: authz.Principal{Kind: authz.KindOAuthClient, ClientID: "provisioner"}}, "")
		assert.Empty(t, entry.UserID)
		assert.Equal(t, [3]string{"client", "provisioner", "Provisioner"}, actor(entry))
		assert.Equal(t, Data{"via": "oauth"}, entry.Data)
	})

	t.Run("static API key", func(t *testing.T) {
		entry := record(t, Origin{Principal: authz.Principal{Kind: authz.KindStaticAPIKey}}, "owner")
		assert.Equal(t, [3]string{"static_api_key", "", ""}, actor(entry))
		assert.Equal(t, Data{"via": "api_key"}, entry.Data)
	})

	t.Run("LDAP sync", func(t *testing.T) {
		entry := record(t, LDAPSync, "owner")
		assert.Equal(t, [3]string{"ldap", "", ""}, actor(entry))
		assert.Empty(t, entry.Data)
	})

	t.Run("nobody signed in", func(t *testing.T) {
		entry := record(t, Origin{}, "owner")
		assert.Equal(t, [3]string{"", "", ""}, actor(entry))
		assert.Empty(t, entry.Data)
	})
}

func actor(entry AuditLog) [3]string {
	return [3]string{string(entry.ActorType), entry.ActorID, entry.ActorName}
}
