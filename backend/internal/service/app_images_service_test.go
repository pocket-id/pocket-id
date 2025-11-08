package service

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func TestAppImagesService_GetImage(t *testing.T) {
	store, err := storage.NewFileStorage(context.Background(), storage.Config{
		Backend: "fs",
		Root:    t.TempDir(),
	})
	require.NoError(t, err)

	require.NoError(t, store.Save(context.Background(), path.Join("application-images", "background.webp"), bytes.NewReader([]byte("data"))))

	service := NewAppImagesService(map[string]string{"background": "webp"}, store)

	reader, size, mimeType, err := service.GetImage(context.Background(), "background")
	require.NoError(t, err)
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), payload)
	require.Equal(t, int64(len(payload)), size)
	require.Equal(t, "image/webp", mimeType)
}

func TestAppImagesService_UpdateImage(t *testing.T) {
	store, err := storage.NewFileStorage(context.Background(), storage.Config{
		Backend: "fs",
		Root:    t.TempDir(),
	})
	require.NoError(t, err)

	require.NoError(t, store.Save(context.Background(), path.Join("application-images", "logoLight.svg"), bytes.NewReader([]byte("old"))))

	service := NewAppImagesService(map[string]string{"logoLight": "svg"}, store)

	fileHeader := newFileHeader(t, "logoLight.png", []byte("new"))

	require.NoError(t, service.UpdateImage(fileHeader, "logoLight"))

	reader, _, err := store.Open(context.Background(), path.Join("application-images", "logoLight.png"))
	require.NoError(t, err)
	_ = reader.Close()

	_, _, err = store.Open(context.Background(), path.Join("application-images", "logoLight.svg"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAppImagesService_ErrorsAndFlags(t *testing.T) {
	store, err := storage.NewFileStorage(context.Background(), storage.Config{
		Backend: "fs",
		Root:    t.TempDir(),
	})
	require.NoError(t, err)

	service := NewAppImagesService(map[string]string{}, store)

	t.Run("get missing image returns not found", func(t *testing.T) {
		_, _, _, err := service.GetImage(context.Background(), "missing")
		require.Error(t, err)
		var imageErr *common.ImageNotFoundError
		assert.ErrorAs(t, err, &imageErr)
	})

	t.Run("reject unsupported file types", func(t *testing.T) {
		err := service.UpdateImage(newFileHeader(t, "logo.txt", []byte("nope")), "logo")
		require.Error(t, err)
		var fileTypeErr *common.FileTypeNotSupportedError
		assert.ErrorAs(t, err, &fileTypeErr)
	})

	t.Run("delete and extension tracking", func(t *testing.T) {
		require.NoError(t, store.Save(context.Background(), path.Join("application-images", "default-profile-picture.png"), bytes.NewReader([]byte("img"))))
		service.extensions["default-profile-picture"] = "png"

		require.NoError(t, service.DeleteImage("default-profile-picture"))
		assert.False(t, service.IsDefaultProfilePictureSet())

		err := service.DeleteImage("default-profile-picture")
		require.Error(t, err)
		var imageErr *common.ImageNotFoundError
		assert.ErrorAs(t, err, &imageErr)
	})
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

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	require.NoError(t, err)

	return fileHeader
}
