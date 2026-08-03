package apikey

import (
	"testing"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

func TestMissingManagedAPIKeyReturnsNotFound(t *testing.T) {
	service, err := newService(t.Context(), testutils.NewDatabaseForTest(t), "")
	require.NoError(t, err)

	err = service.RevokeApiKey(t.Context(), "user-id", "missing-key")
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))

	_, _, err = service.RenewApiKey(t.Context(), "user-id", "missing-key", time.Now().Add(time.Hour))
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyNotFound))
}
