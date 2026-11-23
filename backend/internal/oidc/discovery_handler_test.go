package oidc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type testKeyPublisher struct{}

func (testKeyPublisher) GetKeyAlg() (jwa.KeyAlgorithm, error) { return jwa.RS256(), nil }

func (testKeyPublisher) PublicJWKS() []byte { return []byte(`{"keys":[]}`) }

func TestClientIDMetadataDocumentDiscoveryFollowsAllowlist(t *testing.T) {
	origURL := common.EnvConfig.AppURL
	t.Cleanup(func() {
		common.EnvConfig.AppURL = origURL
	})

	common.EnvConfig.AppURL = "https://test.example.com"
	cimdURLAllowlist := []string(nil)
	wkc := &discoveryHandler{
		db:   testutils.NewDatabaseForTest(t),
		keys: testKeyPublisher{},
		getCIMDURLAllowlist: func() []string {
			return cimdURLAllowlist
		},
	}

	parse := func(t *testing.T) map[string]any {
		t.Helper()
		raw, err := wkc.computeServerMetadata(t.Context())
		require.NoError(t, err)
		var cfg map[string]any
		require.NoError(t, json.Unmarshal(raw, &cfg))
		return cfg
	}

	cimdURLAllowlist = []string{"https://client.example.com/**"}
	assert.Equal(t, true, parse(t)["client_id_metadata_document_supported"])

	cimdURLAllowlist = nil
	assert.Equal(t, false, parse(t)["client_id_metadata_document_supported"])
}

func TestOAuthAuthorizationServerMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origAppURL := common.EnvConfig.AppURL
	origInternalAppURL := common.EnvConfig.InternalAppURL
	t.Cleanup(func() {
		common.EnvConfig.AppURL = origAppURL
		common.EnvConfig.InternalAppURL = origInternalAppURL
	})
	common.EnvConfig.AppURL = "https://test.example.com"
	common.EnvConfig.InternalAppURL = "https://test.example.com"

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&claimmappingpolicy.CustomScope{Key: "kuber", Name: "Kubernetes role", Description: "Your Kubernetes role"}).Error)

	h := &discoveryHandler{db: db, keys: testKeyPublisher{}}
	router := gin.New()
	router.GET("/.well-known/oauth-authorization-server", httpserver.Handle(h.oauthAuthorizationServerHandler).Handler)
	router.GET("/.well-known/openid-configuration", httpserver.Handle(h.openIDConfigurationHandler).Handler)

	get := func(t *testing.T, path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	w := get(t, "/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var doc map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))

	assert.Equal(t, common.EnvConfig.AppURL, doc["issuer"])
	assert.Equal(t, common.EnvConfig.AppURL+"/authorize", doc["authorization_endpoint"])
	assert.Equal(t, common.EnvConfig.InternalAppURL+"/api/oidc/token", doc["token_endpoint"])
	assert.Contains(t, doc["response_types_supported"], "code")
	assert.NotEmpty(t, doc["jwks_uri"])
	assert.Contains(t, doc["scopes_supported"], "openid")
	assert.Contains(t, doc["scopes_supported"], "kuber")
	assert.Contains(t, doc["grant_types_supported"], "authorization_code")
	assert.Contains(t, doc["code_challenge_methods_supported"], "S256")
	assert.Equal(t, "https://pocket-id.org/docs", doc["service_documentation"])
	assert.ElementsMatch(t, []any{"query", "fragment", "form_post"}, doc["response_modes_supported"])
	assert.NotContains(t, doc, "revocation_endpoint")
	assert.NotContains(t, doc, "registration_endpoint")

	for name, value := range doc {
		if arr, ok := value.([]any); ok {
			assert.NotEmpty(t, arr, "metadata member %q must be omitted when it has no values", name)
		}
	}

	assert.JSONEq(t, get(t, "/.well-known/openid-configuration").Body.String(), w.Body.String())
}
