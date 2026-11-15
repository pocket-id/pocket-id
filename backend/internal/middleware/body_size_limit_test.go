package middleware

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBodySizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.Use(NewBodySizeLimitMiddleware().Add(64))
	router.POST("/json", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.POST("/upload", NewFileSizeLimitMiddleware().Add(1024), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	serve := func(path string, body []byte, contentType string) int {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, bytes.NewReader(body))
		request.Header.Set("Content-Type", contentType)
		router.ServeHTTP(recorder, request)
		return recorder.Code
	}

	require.Equal(t, http.StatusNoContent, serve("/json", bytes.Repeat([]byte("x"), 64), "application/json"))
	require.Equal(t, http.StatusRequestEntityTooLarge, serve("/json", bytes.Repeat([]byte("x"), 65), "application/json"))

	// A route with its own file size limit may accept more than the global limit
	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("file", "image.png")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("x"), 512))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.Equal(t, http.StatusNoContent, serve("/upload", upload.Bytes(), writer.FormDataContentType()))
}
