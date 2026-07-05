package service

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

func TestAuditLogService_CreateSignInFailed(t *testing.T) {
	// Setup database using project testing utility
	db := testutils.NewDatabaseForTest(t)

	// Initialize service
	service := NewAuditLogService(db, nil, nil, nil)

	// Test data
	ipAddress := "127.0.0.1"
	userAgent := "test-agent"
	userID := "test-user"

	// When
	createdLog := service.CreateSignInFailure(t.Context(), ipAddress, userAgent, userID)

	// Then
	require.NotEmpty(t, createdLog.ID)
	require.Equal(t, model.AuditLogEventSignInFailed, createdLog.Event)
	require.Equal(t, userID, createdLog.UserID)
	require.Equal(t, ipAddress, *createdLog.IpAddress)

	// Verify in DB
	var dbLog model.AuditLog
	err := db.First(&dbLog, "id = ?", createdLog.ID).Error
	require.NoError(t, err)
	require.Equal(t, model.AuditLogEventSignInFailed, dbLog.Event)
}
