package oidc

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestLooksLikeClientIDMetadataDocumentURL(t *testing.T) {
	cases := map[string]bool{
		"https://app.example.com/oauth/client": true,
		"https://app.example.com/c":            true,
		"my-app":                               false,
		"client-1234":                          false,
		"http://app.example.com/oauth/client":  false,
		"":                                     false,
	}
	for in, want := range cases {
		assert.Equalf(t, want, looksLikeClientIDMetadataDocumentURL(in), "input %q", in)
	}
}

func TestValidateClientIDMetadataDocumentURL(t *testing.T) {
	valid := []string{
		"https://app.example.com/oauth/client",
		"https://app.example.com:8443/oauth/client",
	}
	for _, in := range valid {
		u, err := url.Parse(in)
		require.NoError(t, err)
		assert.NoErrorf(t, validateClientIDMetadataDocumentURL(u), "input %q", in)
	}

	invalid := []string{
		"http://app.example.com/oauth/client",
		"https://app.example.com",
		"https://app.example.com/",
		"https://app.example.com/a/../b",
		"https://app.example.com/./a",
		"https://app.example.com/oauth#frag",
		"https://user:pass@app.example.com/oauth",
		"https://app.example.com/oauth?foo=bar",
	}
	for _, in := range invalid {
		u, err := url.Parse(in)
		require.NoError(t, err)
		assert.Errorf(t, validateClientIDMetadataDocumentURL(u), "input %q", in)
	}
}

func TestBuildClientFromMetadata(t *testing.T) {
	const id = "https://app.example.com/oauth/client"

	t.Run("public client maps to PKCE", func(t *testing.T) {
		doc := &clientMetadataDocument{
			ClientID:                id,
			ClientName:              "Example App",
			RedirectURIs:            []string{"https://app.example.com/callback"},
			PostLogoutRedirectURIs:  []string{"https://app.example.com/logout"},
			TokenEndpointAuthMethod: "none",
		}
		c, err := buildClientFromMetadata(doc, id)
		require.NoError(t, err)
		assert.Equal(t, id, c.ID)
		assert.Equal(t, "Example App", c.Name)
		assert.True(t, c.IsPublic)
		assert.True(t, c.PkceEnabled)
		assert.True(t, c.IsMetadataDocument)
		assert.Equal(t, []string{"https://app.example.com/callback"}, []string(c.CallbackURLs))
		assert.Equal(t, []string{"https://app.example.com/logout"}, []string(c.LogoutCallbackURLs))
		assert.Empty(t, c.Credentials.FederatedIdentities)
	})

	t.Run("private_key_jwt maps to federated identity", func(t *testing.T) {
		doc := &clientMetadataDocument{ //nolint:gosec // G101 false positive: auth method name, not a credential
			ClientID:                id,
			RedirectURIs:            []string{"https://app.example.com/callback"},
			TokenEndpointAuthMethod: "private_key_jwt",
			JwksURI:                 "https://app.example.com/jwks.json",
		}
		c, err := buildClientFromMetadata(doc, id)
		require.NoError(t, err)
		assert.False(t, c.IsPublic)
		assert.False(t, c.PkceEnabled)
		require.Len(t, c.Credentials.FederatedIdentities, 1)
		fi := c.Credentials.FederatedIdentities[0]
		assert.Equal(t, id, fi.Issuer)
		assert.Equal(t, id, fi.Subject)
		assert.Equal(t, "https://app.example.com/jwks.json", fi.JWKS)
		// Name falls back to the host when client_name is absent.
		assert.Equal(t, "app.example.com", c.Name)
	})

	t.Run("client_id mismatch is rejected", func(t *testing.T) {
		doc := &clientMetadataDocument{ClientID: "https://evil.example.com/c"}
		_, err := buildClientFromMetadata(doc, id)
		require.Error(t, err)
	})

	t.Run("client secret is rejected", func(t *testing.T) {
		secret := "shh"
		doc := &clientMetadataDocument{ClientID: id, ClientSecret: &secret}
		_, err := buildClientFromMetadata(doc, id)
		require.Error(t, err)

		exp := int64(0)
		doc2 := &clientMetadataDocument{ClientID: id, ClientSecretExpiresAt: &exp}
		_, err = buildClientFromMetadata(doc2, id)
		require.Error(t, err)
	})

	t.Run("secret-based auth methods are rejected", func(t *testing.T) {
		for _, m := range []string{"client_secret_basic", "client_secret_post", "client_secret_jwt"} {
			doc := &clientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: m}
			_, err := buildClientFromMetadata(doc, id)
			require.Errorf(t, err, "method %q", m)
		}
	})

	t.Run("private_key_jwt without jwks_uri is rejected", func(t *testing.T) {
		doc := &clientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: "private_key_jwt"} //nolint:gosec // G101 false positive: auth method name, not a credential
		_, err := buildClientFromMetadata(doc, id)
		require.Error(t, err)
	})
}

func newMetadataFetchStore(responses map[string]*http.Response) *Store {
	return &Store{
		httpClient: &http.Client{
			Transport: &testutils.MockRoundTripper{Responses: responses},
		},
	}
}

func TestParseClientMetadataCacheTTL(t *testing.T) {
	assert.Equal(t, clientMetadataDefaultTTL, parseClientMetadataCacheTTL(""))
	assert.Equal(t, clientMetadataDefaultTTL, parseClientMetadataCacheTTL("no-store"))
	assert.Equal(t, 10*time.Minute, parseClientMetadataCacheTTL("max-age=600"))
	assert.Equal(t, clientMetadataMinCacheTTL, parseClientMetadataCacheTTL("max-age=1"))
	assert.Equal(t, clientMetadataMaxCacheTTL, parseClientMetadataCacheTTL("max-age=999999"))
	assert.Equal(t, 10*time.Minute, parseClientMetadataCacheTTL("public, max-age=600"))
}

func TestFetchClientIDMetadataDocument(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("fetches and parses", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "max-age=600")
		s := newMetadataFetchStore(map[string]*http.Response{id: resp})
		doc, ttl, err := s.fetchClientIDMetadataDocument(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, doc.ClientID)
		assert.Equal(t, "App", doc.ClientName)
		assert.Equal(t, 10*time.Minute, ttl)
	})

	t.Run("non-200 is an error", func(t *testing.T) {
		//nolint:bodyclose
		s := newMetadataFetchStore(map[string]*http.Response{
			id: testutils.NewMockResponse(http.StatusNotFound, "nope"),
		})
		_, _, err := s.fetchClientIDMetadataDocument(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("redirects are not followed", func(t *testing.T) {
		redir := testutils.NewMockResponse(http.StatusFound, "") //nolint:bodyclose // mock response, no real body
		redir.Header.Set("Location", "https://127.0.0.1/elsewhere")
		s := newMetadataFetchStore(map[string]*http.Response{id: redir})
		_, _, err := s.fetchClientIDMetadataDocument(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("oversize body is rejected", func(t *testing.T) {
		big := `{"client_id":"x","padding":"` + strings.Repeat("a", 6*1024) + `"}`
		//nolint:bodyclose
		s := newMetadataFetchStore(map[string]*http.Response{
			id: testutils.NewMockResponse(http.StatusOK, big),
		})
		_, _, err := s.fetchClientIDMetadataDocument(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("non-JSON body is rejected", func(t *testing.T) {
		//nolint:bodyclose
		s := newMetadataFetchStore(map[string]*http.Response{
			id: testutils.NewMockResponse(http.StatusOK, "<html>not json</html>"),
		})
		_, _, err := s.fetchClientIDMetadataDocument(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("private url", func(t *testing.T) {
		s := &Store{}
		_, _, err := s.fetchClientIDMetadataDocument(t.Context(), "https://127.0.0.1/oauth/client")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private IP addresses are not allowed")
	})
}

func TestMetadataClientChanges(t *testing.T) {
	base := model.OidcClient{
		Name:               "App",
		CallbackURLs:       model.UrlList{"https://app/cb"},
		LogoutCallbackURLs: model.UrlList{"https://app/lo"},
		IsPublic:           true,
		Credentials: model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{
			{Issuer: "https://app", JWKS: "https://app/jwks"},
		}},
	}

	t.Run("no changes", func(t *testing.T) {
		assert.Empty(t, metadataClientChanges(base, base))
	})

	t.Run("redirect_uris change", func(t *testing.T) {
		next := base
		next.CallbackURLs = model.UrlList{"https://app/other"}
		assert.Equal(t, []string{"redirect_uris"}, metadataClientChanges(base, next))
	})

	t.Run("auth method and jwks change", func(t *testing.T) {
		next := base
		next.IsPublic = false
		next.Credentials = model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{
			{Issuer: "https://app", JWKS: "https://app/jwks2"},
		}}
		got := metadataClientChanges(base, next)
		assert.Contains(t, got, "token_endpoint_auth_method")
		assert.Contains(t, got, "jwks_uri")
	})
}

func newMetadataStore(t *testing.T, responses map[string]*http.Response, opts ...StoreOption) *Store {
	t.Helper()
	allOpts := append([]StoreOption{
		WithHTTPClient(&http.Client{Transport: &testutils.MockRoundTripper{Responses: responses}}),
	}, opts...)
	return NewStore(testutils.NewDatabaseForTest(t), allOpts...)
}

func TestGetClient_MetadataDocument(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("non-URL id falls through to the database", func(t *testing.T) {
		s := newMetadataStore(t, nil, WithCIMDEnabled(true))
		_, err := s.GetClient(t.Context(), "does-not-exist")
		require.ErrorIs(t, err, fosite.ErrNotFound)
	})

	t.Run("feature disabled yields not found without fetching", func(t *testing.T) {
		//nolint:bodyclose
		s := newMetadataStore(t, map[string]*http.Response{
			id: testutils.NewMockResponse(http.StatusOK, body),
		})
		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrNotFound)
	})

	t.Run("fetches, upserts, and reuses the cache", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "max-age=600")
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, WithCIMDEnabled(true))

		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		c := fc.(Client).OidcClient
		assert.Equal(t, id, c.ID)
		assert.True(t, c.IsMetadataDocument)
		assert.True(t, c.IsPublic)
		require.NotNil(t, c.MetadataExpiresAt)
		assert.Equal(t, "8.8.8.8", c.ClientIDHost())

		var count int64
		require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count)

		fc2, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "App", fc2.(Client).Name)
	})

	t.Run("refetch when stale preserves consent", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, WithCIMDEnabled(true))

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Old", IsMetadataDocument: true, MetadataExpiresAt: &stale}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-1", "openid", time.Now()).Error)

		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "App", fc.(Client).Name)

		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").
			Where("client_id = ?", id).Count(&consent).Error)
		assert.Equal(t, int64(1), consent)
	})
}
