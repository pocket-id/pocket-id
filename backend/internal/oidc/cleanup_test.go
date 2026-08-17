package oidc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestCleanupExpiredOAuth2SessionsKeepsInvalidatedButUnexpiredSessions verifies that
// rotated/consumed (active=false) sessions are kept until their original expiry, so fosite
// can still detect refresh-token reuse and revoke the affected token family. Only rows
// past their expiry are removed.
func TestCleanupExpiredOAuth2SessionsKeepsInvalidatedButUnexpiredSessions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	err := db.Create(&model.OidcClient{Base: model.Base{ID: "cleanup-client"}, Name: "Cleanup Client"}).Error
	require.NoError(t, err)

	future := datatype.DateTime(time.Now().Add(time.Hour))

	rows := []OAuth2Session{
		{Base: model.Base{ID: "expired"}, Kind: "access_token", Key: "k-expired", RequestID: "r1", ClientID: "cleanup-client", Active: true, RequestData: `{"client_id":"cleanup-client"}`, ExpiresAt: new(datatype.DateTime(time.Now().Add(-time.Hour)))},
		{Base: model.Base{ID: "rotated"}, Kind: "refresh_token", Key: "k-rotated", RequestID: "r2", ClientID: "cleanup-client", Active: false, RequestData: `{"client_id":"cleanup-client"}`, ExpiresAt: &future},
		{Base: model.Base{ID: "active"}, Kind: "refresh_token", Key: "k-active", RequestID: "r3", ClientID: "cleanup-client", Active: true, RequestData: `{"client_id":"cleanup-client"}`, ExpiresAt: &future},
	}
	for i := range rows {
		err = db.Create(&rows[i]).Error
		require.NoError(t, err)
	}

	deleted, err := cleanupExpiredOAuth2Sessions(t.Context(), db)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	var remaining []string
	err = db.Model(&OAuth2Session{}).Pluck("id", &remaining).Error
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"active", "rotated"}, remaining)
}
