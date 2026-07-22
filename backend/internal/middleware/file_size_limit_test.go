package middleware

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

func TestHumaFileSizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := httpapi.New(router, router.Group("/"))
	operation := huma.Operation{OperationID: "multipart-overflow", Method: http.MethodPost, Path: "/api/upload"}
	operation.Middlewares = append(operation.Middlewares, NewFileSizeLimitMiddleware().Huma(api, 64))
	type uploadInput struct {
		RawBody huma.MultipartFormFiles[struct {
			File huma.FormFile `form:"file" required:"true"`
		}]
	}
	httpapi.Register(api, operation, func(context.Context, *uploadInput) (*struct{}, error) {
		t.Fatal("handler must not run after multipart overflow")
		return nil, nil
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "large.bin")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("x"), 256))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	require.JSONEq(t, `{"error":"The file can't be larger than 64 bytes"}`, response.Body.String())
}
