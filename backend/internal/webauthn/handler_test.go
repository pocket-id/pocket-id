package webauthn

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

func TestRequestWithBodyReconstructsUnderlyingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := httpapi.New(router, router.Group("/"))
	type input struct {
		Body map[string]string
	}
	type output struct {
		Body map[string]string
	}
	httpapi.Register(api, huma.Operation{
		OperationID: "reconstruct-request",
		Method:      http.MethodPost,
		Path:        "/api/reconstruct",
	}, func(ctx context.Context, _ *input) (*output, error) {
		request := requestWithBody(ctx, []byte(`{"credential":"value"}`))
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.True(t, bytes.Equal([]byte(`{"credential":"value"}`), body))
		require.Equal(t, int64(len(body)), request.ContentLength)
		return &output{Body: map[string]string{"status": "ok"}}, nil
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/reconstruct", http.NoBody)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)

	request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/reconstruct", strings.NewReader(`{"input":"present"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
}
