package controller

import (
	"encoding/json"
	"testing"

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

func TestClientIDMetadataDocumentDiscoveryFlag(t *testing.T) {
	origFlag := common.EnvConfig.CIMDEnabled
	origURL := common.EnvConfig.AppURL
	t.Cleanup(func() {
		common.EnvConfig.CIMDEnabled = origFlag
		common.EnvConfig.AppURL = origURL
	})

	common.EnvConfig.AppURL = "https://test.example.com"
	jwtSvc := newMinimalJwtService(t)

	parse := func(t *testing.T) map[string]any {
		t.Helper()
		wkc := &WellKnownController{jwtService: jwtSvc}
		raw, err := wkc.computeOIDCConfiguration()
		require.NoError(t, err)
		var cfg map[string]any
		require.NoError(t, json.Unmarshal(raw, &cfg))
		return cfg
	}

	common.EnvConfig.CIMDEnabled = true
	assert.Equal(t, true, parse(t)["client_id_metadata_document_supported"])

	common.EnvConfig.CIMDEnabled = false
	assert.Equal(t, false, parse(t)["client_id_metadata_document_supported"])
}
