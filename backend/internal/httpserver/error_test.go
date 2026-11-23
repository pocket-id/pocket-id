package httpserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

func TestClassifyUnmappedPersistenceErrorAsInternal(t *testing.T) {
	classified := ClassifyError(gorm.ErrRecordNotFound)

	require.Equal(t, apperror.CodeInternal, classified.Code)
	require.Equal(t, http.StatusInternalServerError, classified.Status)
}

func TestClassifyDeadlineAsRequestTimeout(t *testing.T) {
	classified := ClassifyError(context.DeadlineExceeded)

	require.Equal(t, apperror.CodeRequestTimeout, classified.Code)
	require.Equal(t, http.StatusGatewayTimeout, classified.Status)
	require.Equal(t, "Request timed out", classified.Message)
}
