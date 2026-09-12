package controller

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func TestAppImagesControllerGetLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	extensions := map[string]string{}
	appImagesController := &AppImagesController{
		appImagesService: service.NewAppImagesService(extensions, store),
	}

	t.Run("returns the bundled logo if no custom logo is set", func(t *testing.T) {
		res, err := getLogo(t, appImagesController, "/api/application-images/logo")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "image/svg+xml", res.Header().Get("Content-Type"))
		assert.Contains(t, res.Body.String(), `fill="#000"`)
	})

	t.Run("returns the bundled dark mode logo if no custom logo is set", func(t *testing.T) {
		res, err := getLogo(t, appImagesController, "/api/application-images/logo?light=false")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.Contains(t, res.Body.String(), `fill="#fff"`)
	})

	t.Run("returns not found if the bundled logo is skipped", func(t *testing.T) {
		_, err := getLogo(t, appImagesController, "/api/application-images/logo?default=false")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeImageNotFound))
	})

	t.Run("returns the custom logo if one is set", func(t *testing.T) {
		require.NoError(t, store.Save(context.Background(), path.Join("application-images", "logoLight.png"), bytes.NewReader([]byte("custom"))))
		extensions["logoLight"] = "png"
		t.Cleanup(func() {
			delete(extensions, "logoLight")
		})

		for _, target := range []string{"/api/application-images/logo", "/api/application-images/logo?default=false"} {
			res, err := getLogo(t, appImagesController, target)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, "image/png", res.Header().Get("Content-Type"))
			assert.Equal(t, "custom", res.Body.String())
		}
	})
}

func getLogo(t *testing.T, appImagesController *AppImagesController, target string) (*httptest.ResponseRecorder, error) {
	t.Helper()

	res := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(res)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)

	return res, appImagesController.getLogoHandler(ctx)
}
