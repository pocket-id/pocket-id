package oidc

import (
	"net/http"
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestBuildClientFromMetadata(t *testing.T) {
	const id = "https://app.example.com/oauth/client"

	t.Run("public client maps to PKCE", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{
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
		assert.True(t, c.IsMetadataDocument())
		assert.Equal(t, []string{"https://app.example.com/callback"}, []string(c.CallbackURLs))
		assert.Equal(t, []string{"https://app.example.com/logout"}, []string(c.LogoutCallbackURLs))
		assert.Empty(t, c.Credentials.FederatedIdentities)
	})

	t.Run("private_key_jwt maps to federated identity", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{ //nolint:gosec // G101 false positive: auth method name, not a credential
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

	t.Run("secret-based auth methods are rejected", func(t *testing.T) {
		for _, m := range []string{"client_secret_basic", "client_secret_post", "client_secret_jwt"} {
			doc := &fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: m}
			_, err := buildClientFromMetadata(doc, id)
			require.Errorf(t, err, "method %q", m)
		}
	})

	t.Run("private_key_jwt without jwks_uri is rejected", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: "private_key_jwt"} //nolint:gosec // G101 false positive: auth method name, not a credential
		_, err := buildClientFromMetadata(doc, id)
		require.Error(t, err)
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

func TestRefreshMetadataClient(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("feature disabled", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("non-URL id", func(t *testing.T) {
		s := newMetadataStore(t, nil, withCIMDEnabled(true))
		_, err := s.RefreshMetadataClient(t.Context(), "not-a-url")
		require.Error(t, err)
	})

	t.Run("unknown client yields not found", func(t *testing.T) {
		s := newMetadataStore(t, nil, withCIMDEnabled(true))
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("non-metadata client is rejected", func(t *testing.T) {
		s := newMetadataStore(t, nil, withCIMDEnabled(true))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Standard"}
		require.NoError(t, s.db.Create(&seed).Error)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
		require.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("forces re-fetch even when cache is fresh", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, withCIMDEnabled(true))

		fresh := datatype.DateTime(time.Now().Add(time.Hour))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Old", ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &fresh}
		require.NoError(t, s.db.Create(&seed).Error)

		// A normal lookup still returns the cached value.
		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "Old", fc.(Client).Name)

		// A forced refresh re-fetches and updates the cached client.
		c, err := s.RefreshMetadataClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "App", c.Name)
		assert.True(t, c.IsMetadataDocument())
	})
}

func newMetadataStore(t *testing.T, responses map[string]*http.Response, opts ...func(*Store)) *Store {
	t.Helper()
	s := NewStore(testutils.NewDatabaseForTest(t), nil).
		WithHTTPClient(&http.Client{Transport: &testutils.MockRoundTripper{Responses: responses}}).
		WithGetCIMDURLAllowlist(func() []string { return []string{"*"} })
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func withCIMDEnabled(enabled bool) func(*Store) {
	return func(s *Store) { s.WithCIMDEnabled(enabled) }
}

func withGetCIMDURLAllowlist(fn func() []string) func(*Store) {
	return func(s *Store) { s.WithGetCIMDURLAllowlist(fn) }
}

func TestGetClient_CIMDURLAllowlist(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("empty allowlist denies without fetching", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			withCIMDEnabled(true),
			withGetCIMDURLAllowlist(func() []string { return nil }),
		)
		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)

		var count int64
		require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})

	t.Run("non-matching allowlist denies", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			withCIMDEnabled(true),
			withGetCIMDURLAllowlist(func() []string { return []string{"https://other.example.com/**"} }),
		)
		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("matching allowlist allows", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			withCIMDEnabled(true),
			withGetCIMDURLAllowlist(func() []string { return []string{"https://8.8.8.8/**"} }),
		)
		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, fc.(Client).OidcClient.ID)
	})

	t.Run("refresh denied when not allowlisted", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			withCIMDEnabled(true),
			withGetCIMDURLAllowlist(func() []string { return nil }),
		)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
	})
}

func TestGetClient_MetadataDocument(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("non-URL id falls through to the database", func(t *testing.T) {
		s := newMetadataStore(t, nil, withCIMDEnabled(true))
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
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, withCIMDEnabled(true))

		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		c := fc.(Client).OidcClient
		assert.Equal(t, id, c.ID)
		assert.True(t, c.IsMetadataDocument())
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
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, withCIMDEnabled(true))

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Old", ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &stale}
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
