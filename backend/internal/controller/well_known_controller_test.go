package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

func newMinimalJwtService(t *testing.T) *service.JwtService {
	t.Helper()

	key, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	require.NoError(t, err, "failed to generate test JWK key")

	svc := &service.JwtService{}
	require.NoError(t, svc.SetKey(key), "failed to set JWK key on JwtService")
	return svc
}

func TestClientIDMetadataDocumentDiscoveryFollowsAllowlist(t *testing.T) {
	origURL := common.EnvConfig.AppURL
	t.Cleanup(func() {
		common.EnvConfig.AppURL = origURL
	})

	common.EnvConfig.AppURL = "https://test.example.com"
	jwtSvc := newMinimalJwtService(t)
	cimdURLAllowlist := []string(nil)
	wkc := &WellKnownController{
		jwtService: jwtSvc,
		getCIMDURLAllowlist: func() []string {
			return cimdURLAllowlist
		},
	}

	parse := func(t *testing.T) map[string]any {
		t.Helper()
		raw, err := wkc.computeOIDCConfiguration()
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

	jwtSvc := newMinimalJwtService(t)

	router := gin.New()
	group := router.Group("/")
	NewWellKnownController(group, jwtSvc,
		func() []string { return nil },
		func() []string { return []string{"https://app.example.com/**"} })

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/.well-known/oauth-authorization-server", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))

	assert.Equal(t, common.EnvConfig.AppURL, doc["issuer"])
	assert.Equal(t, common.EnvConfig.AppURL+"/authorize", doc["authorization_endpoint"])
	assert.NotEmpty(t, doc["jwks_uri"])
	assert.Contains(t, doc["code_challenge_methods_supported"], "S256")
	assert.Equal(t, true, doc["resource_indicators_supported"])
	// Advertised because this server was given a non-empty DCR redirect allowlist.
	assert.NotEmpty(t, doc["registration_endpoint"])
}

// With no DCR redirect URI allowlist configured, registration would reject every
// request, so the endpoint must not be advertised. The empty allowlist is the
// disabled-by-default state; there is no separate feature flag.
func TestRegistrationEndpointDiscoveryFollowsAllowlist(t *testing.T) {
	jwtSvc := newMinimalJwtService(t)

	dcrAllowlist := []string(nil)
	wkc := &WellKnownController{
		jwtService:                 jwtSvc,
		getCIMDURLAllowlist:        func() []string { return nil },
		getDCRRedirectURIAllowlist: func() []string { return dcrAllowlist },
	}

	parse := func(t *testing.T) map[string]any {
		t.Helper()
		raw, err := wkc.computeOAuthASMetadata()
		require.NoError(t, err)
		var cfg map[string]any
		require.NoError(t, json.Unmarshal(raw, &cfg))
		return cfg
	}

	assert.NotContains(t, parse(t), "registration_endpoint")

	dcrAllowlist = []string{"https://app.example.com/**"}
	assert.NotEmpty(t, parse(t)["registration_endpoint"])
}

// TestBackChannelEndpointsUseInternalURL pins which base URL each advertised endpoint is
// built on. A deployment can set INTERNAL_APP_URL to a separate address that clients reach
// directly, so an endpoint the client calls back-channel must be advertised there rather
// than on the browser-facing URL.
//
// registration is such an endpoint, exactly like token, introspection and PAR, so it is
// grouped with them here; the endpoints a browser is redirected to must stay on the public
// URL. Asserting the whole grouping means a future endpoint added to the wrong base is
// caught rather than only the one that was previously wrong.
func TestBackChannelEndpointsUseInternalURL(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.AppURL = "https://public.example.com"
	common.EnvConfig.InternalAppURL = "https://internal.example.com"

	wkc := &WellKnownController{
		jwtService:                 newMinimalJwtService(t),
		getCIMDURLAllowlist:        func() []string { return nil },
		getDCRRedirectURIAllowlist: func() []string { return []string{"https://app.example.com/**"} },
	}

	raw, err := wkc.computeOAuthASMetadata()
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	for _, endpoint := range []string{
		"token_endpoint",
		"introspection_endpoint",
		"pushed_authorization_request_endpoint",
		"registration_endpoint",
	} {
		value, _ := doc[endpoint].(string)
		assert.True(t, strings.HasPrefix(value, common.EnvConfig.InternalAppURL),
			"%s is called back-channel and must be advertised on the internal URL, got %q", endpoint, value)
	}

	for _, endpoint := range []string{
		"authorization_endpoint",
		"device_authorization_endpoint",
	} {
		value, _ := doc[endpoint].(string)
		assert.True(t, strings.HasPrefix(value, common.EnvConfig.AppURL),
			"%s is reached by a browser and must stay on the public URL, got %q", endpoint, value)
	}
}
