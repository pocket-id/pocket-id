package customclaims

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestCustomClaimUpdatesRejectMissingOwner(t *testing.T) {
	service := &Service{db: testutils.NewDatabaseForTest(t)}

	_, err := service.UpdateCustomClaimsForUser(t.Context(), "missing-user", nil)
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	_, err = service.UpdateCustomClaimsForUserGroup(t.Context(), "missing-group", nil)
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func TestIsReservedClaimIgnoresUnicodeCase(t *testing.T) {
	for _, key := range []string{"ſub", "groupſ"} {
		require.True(t, isReservedClaim(key), key)
	}
}
