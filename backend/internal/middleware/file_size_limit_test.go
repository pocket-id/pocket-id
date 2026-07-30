package middleware

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/stretchr/testify/require"
)

func TestFileSizeLimitMiddlewareClassifiesMultipartErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("oversized body", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "large.png")
		require.NoError(t, err)
		_, err = part.Write(bytes.Repeat([]byte("x"), 128))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		recorder := serveMultipartRequest(t, body.Bytes(), writer.FormDataContentType(), 64)

		require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
		require.Equal(t, apperror.CodeFileTooLarge, responseCode(t, recorder))
	})

	t.Run("malformed body", func(t *testing.T) {
		recorder := serveMultipartRequest(t, []byte("not multipart"), "multipart/form-data; boundary=missing", 1024)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, apperror.CodeInvalidRequestBody, responseCode(t, recorder))
	})
}

func serveMultipartRequest(t *testing.T, body []byte, contentType string, maxSize int64) *httptest.ResponseRecorder {
	t.Helper()

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.POST("/", NewFileSizeLimitMiddleware().Add(maxSize), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	router.ServeHTTP(recorder, request)
	return recorder
}

func responseCode(t *testing.T, recorder *httptest.ResponseRecorder) apperror.Code {
	t.Helper()

	var body struct {
		Code apperror.Code `json:"code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Code
}
