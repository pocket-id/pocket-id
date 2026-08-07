//go:build unit

package service

import (
	"net/http"
	"sync"
	"testing"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestRegisterDynamicClient(t *testing.T) {
	// GetDynamicClientRedirectUriAllowlist reads from the
	// in-memory env config, which GetConfig only returns when the UI config is disabled.
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)

	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	appConfigService := appconfig.NewTestAppConfigService(cfg)

	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appConfigService

	t.Run("confidential client gets a secret and a registration token", func(t *testing.T) {
		client, secret, regToken, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			ClientName:              "MCP Client",
			TokenEndpointAuthMethod: "client_secret_basic",
		})
		require.NoError(t, err)
		assert.Equal(t, model.OidcClientTypeDynamic, client.ClientType)
		assert.NotEmpty(t, secret)
		assert.NotEmpty(t, regToken)
		assert.NotNil(t, client.RegistrationAccessTokenHash)
	})

	t.Run("public client (auth method none) gets no secret", func(t *testing.T) {
		client, secret, _, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			TokenEndpointAuthMethod: "none",
		})
		require.NoError(t, err)
		assert.True(t, client.IsPublic)
		assert.Empty(t, secret)
	})

	t.Run("redirect URI outside the allowlist is rejected", func(t *testing.T) {
		_, _, _, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs: []string{"https://evil.example.com/cb"},
		})
		require.Error(t, err)
	})
}

// TestRegisterDynamicClient_LogoURI exercises the logo_uri wiring added to
// RegisterDynamicClient and UpdateDynamicClient: a served logo_uri is
// downloaded (via the same SSRF-guarded downloadAndSaveLogoFromURL used by
// CreateClient) and stored against the new client, best-effort. The SSRF
// guard itself, redirect handling, size limits, etc. are already covered
// by TestOidcService_downloadAndSaveLogoFromURL; this test only asserts the
// new call site wires things together correctly.
func TestRegisterDynamicClient_LogoURI(t *testing.T) {
	const publicLogoHost = "https://8.8.8.8"

	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)

	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	appConfigService := appconfig.NewTestAppConfigService(cfg)

	dbStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	t.Run("a served logo_uri is downloaded and stored on the client", func(t *testing.T) {
		pngContent := []byte("fake-png-content")
		//nolint:bodyclose
		pngResponse := testutils.NewMockResponse(http.StatusOK, string(pngContent))
		pngResponse.Header.Set("Content-Type", "image/png")

		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: map[string]*http.Response{
					//nolint:bodyclose
					publicLogoHost + "/logo.png": pngResponse,
				},
			},
		}

		svc, err := NewOidcService(db, nil, appConfigService, nil, nil, nil, httpClient, dbStorage)
		require.NoError(t, err)

		client, _, _, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			ClientName:              "Client With Logo",
			TokenEndpointAuthMethod: "client_secret_basic",
			LogoURI:                 publicLogoHost + "/logo.png",
		})
		require.NoError(t, err)

		var stored model.OidcClient
		require.NoError(t, db.First(&stored, "id = ?", client.ID).Error)
		assert.True(t, stored.HasLogo())
		require.NotNil(t, stored.ImageType)
		assert.Equal(t, "png", *stored.ImageType)
	})

	t.Run("a logo_uri that fails to download does not fail registration", func(t *testing.T) {
		svc, err := NewOidcService(db, nil, appConfigService, nil, nil, nil, http.DefaultClient, dbStorage)
		require.NoError(t, err)

		client, _, _, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			ClientName:              "Client With Bad Logo",
			TokenEndpointAuthMethod: "client_secret_basic",
			LogoURI:                 "http://127.0.0.1/logo.png", // rejected by the SSRF guard
		})
		require.NoError(t, err)
		assert.NotEmpty(t, client.ID)

		var stored model.OidcClient
		require.NoError(t, db.First(&stored, "id = ?", client.ID).Error)
		assert.False(t, stored.HasLogo())
	})
}

func TestDynamicClientConfiguration(t *testing.T) {
	// GetDynamicClientRedirectUriAllowlist reads from the
	// in-memory env config, which GetConfig only returns when the UI config is disabled.
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)

	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	appConfigService := appconfig.NewTestAppConfigService(cfg)

	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appConfigService

	// Register a confidential dynamic client, capturing clientID + regToken.
	registered, _, regToken, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "MCP Client",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	clientID := registered.ID

	// Create a standard (non-dynamic) client directly in the DB.
	standardClient := model.OidcClient{
		Name:         "Standard Client",
		CallbackURLs: datatype.StringList{"https://app.example.com/cb"},
		ClientType:   model.OidcClientTypeStandard,
	}
	require.NoError(t, db.Create(&standardClient).Error)
	standardID := standardClient.ID

	t.Run("GET returns the client with a valid token and rotates it", func(t *testing.T) {
		got, rotated, err := svc.GetDynamicClient(t.Context(), clientID, regToken)
		require.NoError(t, err)
		assert.Equal(t, clientID, got.ID)

		// RFC 7592 section 2.1 and appendix A.1: a read rotates the registration
		// access token and returns the new one, so the old one stops working.
		require.NotEmpty(t, rotated)
		assert.NotEqual(t, regToken, rotated)

		_, _, err = svc.GetDynamicClient(t.Context(), clientID, regToken)
		require.Error(t, err, "the superseded token must no longer authenticate")

		regToken = rotated
	})

	t.Run("GET with a wrong token is rejected", func(t *testing.T) {
		_, _, err := svc.GetDynamicClient(t.Context(), clientID, "wrong-token")
		require.Error(t, err)
	})

	t.Run("GET on a non-dynamic client is rejected", func(t *testing.T) {
		_, _, err := svc.GetDynamicClient(t.Context(), standardID, regToken)
		require.Error(t, err)
	})

	t.Run("PUT updates and re-validates redirect URIs", func(t *testing.T) {
		updated, secret, rotated, err := svc.UpdateDynamicClient(t.Context(), clientID, regToken, fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/new"},
			ClientName:              "Renamed",
			TokenEndpointAuthMethod: "client_secret_basic",
		})
		require.NoError(t, err)
		assert.Equal(t, "Renamed", updated.Name)
		// Already confidential with an existing secret, so no new secret is issued.
		assert.Empty(t, secret)
		require.NotEmpty(t, rotated)
		regToken = rotated
	})

	t.Run("PUT rejects a redirect URI outside the allowlist", func(t *testing.T) {
		_, _, _, err := svc.UpdateDynamicClient(t.Context(), clientID, regToken, fosite.ClientRegistrationRequest{
			RedirectURIs: []string{"https://evil.example.com/cb"},
		})
		require.Error(t, err)
	})

	t.Run("PUT issues a new secret on a public-to-confidential transition", func(t *testing.T) {
		publicClient, _, publicRegToken, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			ClientName:              "Public Client",
			TokenEndpointAuthMethod: "none",
		})
		require.NoError(t, err)
		require.Empty(t, publicClient.Secret)

		updated, secret, _, err := svc.UpdateDynamicClient(t.Context(), publicClient.ID, publicRegToken, fosite.ClientRegistrationRequest{
			RedirectURIs:            []string{"https://app.example.com/cb"},
			ClientName:              "Public Client",
			TokenEndpointAuthMethod: "client_secret_basic",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.NotEmpty(t, updated.Secret)
		assert.False(t, updated.IsPublic)
	})

	t.Run("DELETE removes the client", func(t *testing.T) {
		require.NoError(t, svc.DeleteDynamicClient(t.Context(), clientID, regToken))
		_, _, err := svc.GetDynamicClient(t.Context(), clientID, regToken)
		require.Error(t, err)
	})
}

// TestDynamicClientRegistrationAccessTokenRotationChain covers the full RFC 7592
// credential lifecycle across register, read and update.
//
// RFC 7592 section 3 makes registration_access_token REQUIRED on every client
// information response, but only a bcrypt hash is stored, so the token issued at
// registration cannot be reproduced later. Section 2.1 and appendix A.1 resolve this
// by rotating on read or update and returning the new value. This asserts both halves:
// each response carries a token, and the superseded one stops authenticating.
func TestDynamicClientRegistrationAccessTokenRotationChain(t *testing.T) {
	// The allowlist is read from the in-memory env config, which GetConfig only
	// returns when the UI config is disabled.
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appconfig.NewTestAppConfigService(cfg)

	client, _, tok0, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Rot",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tok0)

	_, tok1, err := svc.GetDynamicClient(t.Context(), client.ID, tok0)
	require.NoError(t, err)
	require.NotEmpty(t, tok1, "RFC 7592 s3: read response must carry a registration_access_token")
	assert.NotEqual(t, tok0, tok1)

	_, _, err = svc.GetDynamicClient(t.Context(), client.ID, tok0)
	require.Error(t, err, "superseded token must be rejected")

	_, _, tok2, err := svc.UpdateDynamicClient(t.Context(), client.ID, tok1, fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Rot2",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tok2, "RFC 7592 s3: update response must carry a registration_access_token")
	assert.NotEqual(t, tok1, tok2)

	_, _, err = svc.GetDynamicClient(t.Context(), client.ID, tok1)
	require.Error(t, err, "superseded token must be rejected after update")

	_, tok3, err := svc.GetDynamicClient(t.Context(), client.ID, tok2)
	require.NoError(t, err, "the newest token must still work")
	require.NotEmpty(t, tok3)
	t.Log("ok  rotation chain: register -> read -> update -> read, each token supersedes the last")
}

// TestDynamicClientRotationIsAtomicUnderConcurrency guards against handing a client a
// registration access token that does not work.
//
// Authentication and rotation have to happen together. If they do not, two requests
// carrying the same valid token can both authenticate against the same stored hash and
// then both rotate: each caller receives a different new token, but only the last write
// survives, so one caller is left holding a dead credential and locked out of managing
// its registration, which RFC 7592 section 5 explicitly warns against.
//
// The guarantee asserted here is that a caller either gets a token that authenticates or
// gets an error, never a token that silently does not work.
func TestDynamicClientRotationIsAtomicUnderConcurrency(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appconfig.NewTestAppConfigService(cfg)

	client, _, token, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Concurrent",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)

	const readers = 2
	issued := make([]string, readers)
	failures := make([]error, readers)

	var wg sync.WaitGroup
	for i := range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, rotated, err := svc.GetDynamicClient(t.Context(), client.ID, token)
			issued[i], failures[i] = rotated, err
		}()
	}
	wg.Wait()

	handedOut := 0
	for i := range readers {
		if failures[i] != nil {
			continue
		}
		handedOut++
		require.NotEmpty(t, issued[i], "a successful read must return a token")

		_, _, err := svc.GetDynamicClient(t.Context(), client.ID, issued[i])
		require.NoError(t, err,
			"every token handed to a caller must authenticate; a dead one locks the client out")
	}
	assert.Positive(t, handedOut, "at least one concurrent read must succeed")
}

// TestRejectedUpdateDoesNotBurnRegistrationToken guards a self-lockout: a client that
// sends an update the server rejects must keep the token it authenticated with.
//
// Rotation is deliberately sequenced after validation for this reason. Rotating up
// front would mean any rejected payload consumed the caller's registration access
// token, so a client could lock itself out of managing its own registration simply by
// sending a bad request, and would have no way to recover.
func TestRejectedUpdateDoesNotBurnRegistrationToken(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)
	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appconfig.NewTestAppConfigService(cfg)

	client, _, token, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Keeps Token",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)

	// Rejected: the redirect URI is outside the allowlist.
	_, _, _, err = svc.UpdateDynamicClient(t.Context(), client.ID, token, fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://evil.example.com/cb"},
		ClientName:              "Keeps Token",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.Error(t, err)

	// The token must still work, otherwise one bad request locked the client out.
	_, rotated, err := svc.GetDynamicClient(t.Context(), client.ID, token)
	require.NoError(t, err, "a rejected update must not consume the registration access token")
	require.NotEmpty(t, rotated)
}
