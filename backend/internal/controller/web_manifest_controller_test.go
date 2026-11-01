package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestWebManifestUsesConfiguredAppName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalEnvConfig := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = originalEnvConfig })
	common.EnvConfig.UiConfigDisabled = true

	cfg := appconfig.NewTestConfig(nil)
	cfg.AppName = "Acme Login"

	router := gin.New()
	NewWebManifestController(&router.RouterGroup, appconfig.NewTestAppConfigService(cfg))

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/app.webmanifest", nil))
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "application/manifest+json", res.Header().Get("Content-Type"))

	var manifest struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &manifest))
	assert.Equal(t, "Acme Login", manifest.Name)
}
