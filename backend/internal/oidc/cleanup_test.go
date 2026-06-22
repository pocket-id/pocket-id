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

	past := datatype.DateTime(time.Now().Add(-time.Hour))
	future := datatype.DateTime(time.Now().Add(time.Hour))

	rows := []OAuth2Session{
		{Base: model.Base{ID: "expired"}, Kind: "access_token", Key: "k-expired", RequestID: "r1", Active: true, RequestData: "{}", ExpiresAt: &past},
		{Base: model.Base{ID: "rotated"}, Kind: "refresh_token", Key: "k-rotated", RequestID: "r2", Active: false, RequestData: "{}", ExpiresAt: &future},
		{Base: model.Base{ID: "active"}, Kind: "refresh_token", Key: "k-active", RequestID: "r3", Active: true, RequestData: "{}", ExpiresAt: &future},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	deleted, err := CleanupExpiredOAuth2Sessions(t.Context(), db)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	var remaining []string
	require.NoError(t, db.Model(&OAuth2Session{}).Pluck("id", &remaining).Error)
	require.ElementsMatch(t, []string{"active", "rotated"}, remaining)
}
