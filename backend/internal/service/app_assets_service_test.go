package service

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func TestAppAssetsService_GetImage(t *testing.T) {
	store := newFilesystemStorageForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "background.webp"), bytes.NewReader([]byte("data")))
	require.NoError(t, err)

	service := NewAppAssetsService(map[string]string{"background": "webp"}, store)

	reader, size, mimeType, err := service.GetAsset(t.Context(), "background")
	require.NoError(t, err)
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), payload)
	require.Equal(t, int64(len(payload)), size)
	require.Equal(t, "image/webp", mimeType)
}

func TestAppAssetsService_GetImageWithDefault(t *testing.T) {
	store := newFilesystemStorageForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "logoDark.png"), bytes.NewReader([]byte("custom")))
	require.NoError(t, err)

	service := NewAppAssetsService(map[string]string{"logoDark": "png"}, store)

	t.Run("returns the custom image if one is set", func(t *testing.T) {
		reader, size, mimeType, err := service.GetAssetWithDefault(t.Context(), "logoDark")
		require.NoError(t, err)
		defer reader.Close()
		payload, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("custom"), payload)
		assert.Equal(t, int64(len(payload)), size)
		assert.Equal(t, "image/png", mimeType)
	})

	t.Run("returns the embedded image if no custom image is set", func(t *testing.T) {
		reader, size, mimeType, err := service.GetAssetWithDefault(t.Context(), "logoLight")
		require.NoError(t, err)
		defer reader.Close()
		payload, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, int64(len(payload)), size)
		assert.Equal(t, "image/svg+xml", mimeType)
		assert.True(t, strings.HasPrefix(string(payload), "<svg"))
	})

	t.Run("returns not found if no embedded image exists", func(t *testing.T) {
		_, _, _, err := service.GetAssetWithDefault(t.Context(), "default-profile-picture")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	})

	t.Run("GetAsset doesn't return the embedded image", func(t *testing.T) {
		_, _, _, err := service.GetAsset(t.Context(), "logoLight")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	})
}

func TestAppAssetsService_UpdateImage(t *testing.T) {
	store := newFilesystemStorageForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "logoLight.svg"), bytes.NewReader([]byte("old")))
	require.NoError(t, err)

	service := NewAppAssetsService(map[string]string{"logoLight": "svg"}, store)

	fileHeader := newFileHeader(t, "logoLight.png", []byte("new"))

	err = service.UpdateAsset(t.Context(), fileHeader, "logoLight")
	require.NoError(t, err)

	reader, _, err := store.Open(t.Context(), path.Join("application-assets", "logoLight.png"))
	require.NoError(t, err)
	_ = reader.Close()

	_, _, err = store.Open(t.Context(), path.Join("application-assets", "logoLight.svg"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAppAssetsService_UpdateImageStripsMetadata(t *testing.T) {
	store := newFilesystemStorageForTest(t)

	service := NewAppAssetsService(map[string]string{}, store)

	fileHeader := newFileHeader(t, "logo.webp", webpFile(
		webpChunk("VP8 ", []byte{1, 2, 3, 4}),
		webpChunk("EXIF", []byte("secret")),
	))

	err := service.UpdateAsset(t.Context(), fileHeader, "logoLight")
	require.NoError(t, err)

	reader, _, err := store.Open(t.Context(), path.Join("application-assets", "logoLight.webp"))
	require.NoError(t, err)
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "secret")
	assert.Contains(t, string(payload), "VP8 ")
}

func TestAppAssetsService_ErrorsAndFlags(t *testing.T) {
	store := newFilesystemStorageForTest(t)

	service := NewAppAssetsService(map[string]string{}, store)

	t.Run("get missing image returns not found", func(t *testing.T) {
		_, _, _, err := service.GetAsset(t.Context(), "missing")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	})

	t.Run("reject unsupported file types", func(t *testing.T) {
		err := service.UpdateAsset(t.Context(), newFileHeader(t, "logo.txt", []byte("nope")), "logo")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeFileTypeNotSupported))
	})

	t.Run("accept fonts only for the title font", func(t *testing.T) {
		err := service.UpdateAsset(t.Context(), newFileHeader(t, "logo.woff2", []byte("font")), "logoLight")
		assert.True(t, apperror.IsCode(err, apperror.CodeFileTypeNotSupported))

		err = service.UpdateAsset(t.Context(), newFileHeader(t, "font.png", []byte("img")), "font")
		assert.True(t, apperror.IsCode(err, apperror.CodeFileTypeNotSupported))

		err = service.UpdateAsset(t.Context(), newFileHeader(t, "Inter-Variable.woff2", []byte("font")), "font")
		require.NoError(t, err)
		reader, _, mimeType, err := service.GetAsset(t.Context(), "font")
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, "font/woff2", mimeType)
	})

	t.Run("delete and extension tracking", func(t *testing.T) {
		err := store.Save(t.Context(), path.Join("application-assets", "default-profile-picture.png"), bytes.NewReader([]byte("img")))
		require.NoError(t, err)
		service.extensions["default-profile-picture"] = "png"

		err = service.DeleteAsset(t.Context(), "default-profile-picture")
		require.NoError(t, err)
		assert.False(t, service.IsDefaultProfilePictureSet())
		reader, size, err := store.Open(t.Context(), deletedApplicationAssetPath("default-profile-picture"))
		require.NoError(t, err)
		assert.Zero(t, size)
		require.NoError(t, reader.Close())

		err = service.DeleteAsset(t.Context(), "default-profile-picture")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))

		err = service.UpdateAsset(t.Context(), newFileHeader(t, "default-profile-picture.png", []byte("new")), "default-profile-picture")
		require.NoError(t, err)
		_, _, err = store.Open(t.Context(), deletedApplicationAssetPath("default-profile-picture"))
		require.ErrorIs(t, err, fs.ErrNotExist)
	})
}

func newFilesystemStorageForTest(t *testing.T) storage.FileStorage {
	t.Helper()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})
	return store
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	require.NoError(t, err)

	return fileHeader
}

func webpFile(chunks ...[]byte) []byte {
	var out bytes.Buffer
	out.WriteString("RIFF")
	out.Write([]byte{0, 0, 0, 0})
	out.WriteString("WEBP")
	for _, chunk := range chunks {
		out.Write(chunk)
	}
	binary.LittleEndian.PutUint32(out.Bytes()[4:8], uint32(out.Len()-8)) //nolint:gosec
	return out.Bytes()
}

func webpChunk(chunkType string, data []byte) []byte {
	var out bytes.Buffer
	out.WriteString(chunkType)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(data))) //nolint:gosec
	out.Write(data)
	if len(data)%2 == 1 {
		out.WriteByte(0)
	}
	return out.Bytes()
}
