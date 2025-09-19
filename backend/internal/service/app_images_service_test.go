package service

import (
	"bytes"
	"io/fs"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestAppImagesService_GetImage(t *testing.T) {
	tempDir := t.TempDir()
	originalUploadPath := common.EnvConfig.UploadPath
	common.EnvConfig.UploadPath = tempDir
	t.Cleanup(func() {
		common.EnvConfig.UploadPath = originalUploadPath
	})

	imagesDir := filepath.Join(tempDir, "application-images")
	require.NoError(t, os.MkdirAll(imagesDir, 0o755))

	filePath := filepath.Join(imagesDir, "background.webp")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), fs.FileMode(0o644)))

	service := NewAppImagesService(map[string]string{"background": "webp"})

	path, mimeType, err := service.GetImage("background")
	require.NoError(t, err)
	require.Equal(t, filePath, path)
	require.Equal(t, "image/webp", mimeType)
}

func TestAppImagesService_UpdateImage(t *testing.T) {
	tempDir := t.TempDir()
	originalUploadPath := common.EnvConfig.UploadPath
	common.EnvConfig.UploadPath = tempDir
	t.Cleanup(func() {
		common.EnvConfig.UploadPath = originalUploadPath
	})

	imagesDir := filepath.Join(tempDir, "application-images")
	require.NoError(t, os.MkdirAll(imagesDir, 0o755))

	oldPath := filepath.Join(imagesDir, "logoLight.svg")
	require.NoError(t, os.WriteFile(oldPath, []byte("old"), fs.FileMode(0o644)))

	service := NewAppImagesService(map[string]string{"logoLight": "svg"})

	fileHeader := newFileHeader(t, "logoLight.png", []byte("new"))

	require.NoError(t, service.UpdateImage(fileHeader, "logoLight"))

	_, err := os.Stat(filepath.Join(imagesDir, "logoLight.png"))
	require.NoError(t, err)

	_, err = os.Stat(oldPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func newFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	require.NoError(t, err)

	return fileHeader
}
