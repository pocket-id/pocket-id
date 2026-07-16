//go:build unit

package devicelogin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestCleanupExpiredRequests(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	requests := []Request{
		{
			Base:            model.Base{ID: "expired-device-login-request"},
			Code:            "PAAAAAAA",
			DeviceTokenHash: "expired-token-hash",
			Status:          RequestStatusPending,
			ExpiresAt:       datatype.DateTime(time.Now().Add(-time.Minute)),
			UserAgent:       "expired-agent",
		},
		{
			Base:            model.Base{ID: "active-device-login-request"},
			Code:            "PBBBBBBB",
			DeviceTokenHash: "active-token-hash",
			Status:          RequestStatusPending,
			ExpiresAt:       datatype.DateTime(time.Now().Add(time.Minute)),
			UserAgent:       "active-agent",
		},
	}
	require.NoError(t, db.Create(&requests).Error)

	deleted, err := CleanupExpiredRequests(t.Context(), db)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	var remaining []Request
	require.NoError(t, db.Order("id").Find(&remaining).Error)
	require.Len(t, remaining, 1)
	require.Equal(t, "active-device-login-request", remaining[0].ID)
}
