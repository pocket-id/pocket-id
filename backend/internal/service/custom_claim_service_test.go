package service

import (
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

func TestCustomClaimUpdatesRejectMissingOwner(t *testing.T) {
	service := NewCustomClaimService(testutils.NewDatabaseForTest(t))

	_, err := service.UpdateCustomClaimsForUser(t.Context(), "missing-user", nil)
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	_, err = service.UpdateCustomClaimsForUserGroup(t.Context(), "missing-group", nil)
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}
