package appassets

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestAppAssetsService_GetImage(t *testing.T) {
	service, store := newServiceForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "background.webp"), bytes.NewReader([]byte("data")))
	require.NoError(t, err)
	require.NoError(t, recordAsset(t.Context(), service.db, "background", "background.webp"))

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
	service, store := newServiceForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "logoDark.png"), bytes.NewReader([]byte("custom")))
	require.NoError(t, err)
	require.NoError(t, recordAsset(t.Context(), service.db, "logoDark", "logoDark.png"))

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
	service, store := newServiceForTest(t)

	err := store.Save(t.Context(), path.Join("application-assets", "logoLight.svg"), bytes.NewReader([]byte("old")))
	require.NoError(t, err)
	require.NoError(t, recordAsset(t.Context(), service.db, "logoLight", "logoLight.svg"))

	fileHeader := newFileHeader(t, "logoLight.png", []byte("new"))

	err = service.UpdateAsset(t.Context(), fileHeader, "logoLight")
	require.NoError(t, err)

	payload, mimeType := readAsset(t, service, "logoLight")
	assert.Equal(t, "new", payload)
	assert.Equal(t, "image/png", mimeType)

	_, _, err = store.Open(t.Context(), path.Join("application-assets", "logoLight.svg"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAppAssetsService_UpdateImageStripsMetadata(t *testing.T) {
	service, _ := newServiceForTest(t)

	fileHeader := newFileHeader(t, "logo.webp", webpFile(
		webpChunk("VP8 ", []byte{1, 2, 3, 4}),
		webpChunk("EXIF", []byte("secret")),
	))

	err := service.UpdateAsset(t.Context(), fileHeader, "logoLight")
	require.NoError(t, err)

	payload, _ := readAsset(t, service, "logoLight")
	assert.NotContains(t, payload, "secret")
	assert.Contains(t, payload, "VP8 ")
}

func TestAppAssetsService_ErrorsAndFlags(t *testing.T) {
	service, _ := newServiceForTest(t)

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

	t.Run("deleting a missing image returns not found", func(t *testing.T) {
		err := service.DeleteAsset(t.Context(), "default-profile-picture")
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	})
}

func TestAppAssetsService_SharedAcrossReplicas(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		store := newFilesystemStorageForTest(t)
		replicaA := &Service{db: db, storage: store}
		replicaB := &Service{db: db, storage: store}

		// Replica B reads the asset once, before replica A replaces it with another file type
		err := replicaB.UpdateAsset(t.Context(), newFileHeader(t, "logo.svg", []byte("<svg/>")), "logoLight")
		require.NoError(t, err)
		_, mimeType := readAsset(t, replicaB, "logoLight")
		assert.Equal(t, "image/svg+xml", mimeType)

		err = replicaA.UpdateAsset(t.Context(), newFileHeader(t, "logo.png", []byte("png")), "logoLight")
		require.NoError(t, err)

		payload, mimeType := readAsset(t, replicaB, "logoLight")
		assert.Equal(t, "image/png", mimeType)
		assert.Equal(t, "png", payload)

		// Replica B can delete what replica A uploaded
		err = replicaB.DeleteAsset(t.Context(), "logoLight")
		require.NoError(t, err)
		_, _, _, err = replicaA.GetAsset(t.Context(), "logoLight")
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
		files, err := store.List(t.Context(), "application-assets/logoLight")
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestAppAssetsService_ConcurrentUploads(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		store := newFilesystemStorageForTest(t)
		replicas := []*Service{{db: db, storage: store}, {db: db, storage: store}}

		// Uploads of both file types race on both replicas, so some replace a file of the same type and some one of the other type
		var wg sync.WaitGroup
		errs := make(chan error, 20)
		for i := range 20 {
			wg.Go(func() {
				fileName := []string{"logo.svg", "logo.png"}[i%2]
				errs <- replicas[i%2].UpdateAsset(context.Background(), newFileHeader(t, fileName, []byte("<svg/>")), "logoLight")
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		// Exactly the file of the last upload is left, and it's the one being served
		readAsset(t, replicas[0], "logoLight")
		files, err := store.List(t.Context(), "application-assets/logoLight")
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})
}

func TestInit(t *testing.T) {
	service, store := newServiceForTest(t)

	// The bundled background is copied once, and stays removed after an admin deleted it
	require.NoError(t, Init(t.Context(), service.db, store))
	_, mimeType := readAsset(t, service, "background")
	assert.Equal(t, "image/avif", mimeType)
	require.NoError(t, service.DeleteAsset(t.Context(), "background"))

	require.NoError(t, Init(t.Context(), service.db, store))

	_, _, _, err := service.GetAsset(t.Context(), "background")
	assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	_, _, err = store.Open(t.Context(), path.Join("application-assets", "background.avif"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func readAsset(t *testing.T, service *Service, name string) (payload string, mimeType string) {
	t.Helper()

	reader, _, mimeType, err := service.GetAsset(t.Context(), name)
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(data), mimeType
}

func newServiceForTest(t *testing.T) (*Service, storage.FileStorage) {
	t.Helper()

	store := newFilesystemStorageForTest(t)
	return &Service{db: testutils.NewDatabaseForTest(t), storage: store}, store
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
