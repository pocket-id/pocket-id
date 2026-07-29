package oidc

import (
	"context"
	"errors"
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

	t.Run("authenticated clients are rejected", func(t *testing.T) {
		for _, m := range []string{"private_key_jwt", "client_secret_basic", "client_secret_post", "client_secret_jwt"} { //nolint:gosec // G101 false positive: authentication method names, not credentials
			doc := &fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: m}
			_, err := buildClientFromMetadata(doc, id)
			require.Errorf(t, err, "method %q", m)
		}
	})

	t.Run("name falls back to the client ID host", func(t *testing.T) {
		c, err := buildClientFromMetadata(&fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: "none"}, id)
		require.NoError(t, err)
		assert.Equal(t, "app.example.com", c.Name)
	})
}

func TestMetadataClientChanges(t *testing.T) {
	base := model.OidcClient{
		Name:               "App",
		CallbackURLs:       model.UrlList{"https://app/cb"},
		LogoutCallbackURLs: model.UrlList{"https://app/lo"},
		IsPublic:           true,
	}

	t.Run("no changes", func(t *testing.T) {
		assert.Empty(t, metadataClientChanges(base, base))
	})

	t.Run("redirect_uris change", func(t *testing.T) {
		next := base
		next.CallbackURLs = model.UrlList{"https://app/other"}
		assert.Equal(t, []string{"redirect_uris"}, metadataClientChanges(base, next))
	})

	t.Run("auth method change", func(t *testing.T) {
		next := base
		next.IsPublic = false
		got := metadataClientChanges(base, next)
		assert.Contains(t, got, "token_endpoint_auth_method")
	})
}

func TestRefreshMetadataClient(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("empty allowlist", func(t *testing.T) {
		s := newMetadataStore(t, nil, func() []string { return nil })
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("non-URL id", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		_, err := s.RefreshMetadataClient(t.Context(), "not-a-url")
		require.Error(t, err)
	})

	t.Run("unknown client yields not found", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("non-metadata client is rejected", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Standard"}
		require.NoError(t, s.db.Create(&seed).Error)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
		require.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("forces re-fetch even when cache is fresh", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		fresh := datatype.DateTime(time.Now().Add(time.Hour))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Old", IsPublic: true, PkceEnabled: true, ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &fresh}
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

type metadataStore struct {
	*Store
	resolver *cimdClientResolver
}

func (s *metadataStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	client, err := s.resolver.ResolveClient(ctx, id, s.Store.GetClient)
	if err == nil {
		return client, nil
	}
	if errors.Is(err, fosite.ErrNotFound) {
		return nil, fosite.ErrNotFound
	}
	if fosite.LooksLikeCIMDURL(id) {
		return nil, fosite.ErrInvalidClient.WithHint("The client metadata document could not be resolved.").WithWrap(err).WithDebug(err.Error())
	}
	return nil, err
}

func (s *metadataStore) RefreshMetadataClient(ctx context.Context, id string) (model.OidcClient, error) {
	return s.resolver.RefreshMetadataClient(ctx, id)
}

func newMetadataStore(t *testing.T, responses map[string]*http.Response, allowlists ...func() []string) *metadataStore {
	t.Helper()
	getAllowlist := func() []string { return []string{"*"} }
	if len(allowlists) > 0 {
		getAllowlist = allowlists[0]
	}
	store := NewStore(testutils.NewDatabaseForTest(t), nil)
	return &metadataStore{
		Store: store,
		resolver: newCIMDClientResolver(store, cimdResolverConfig{
			httpClient:      &http.Client{Transport: &testutils.MockRoundTripper{Responses: responses}},
			getURLAllowlist: getAllowlist,
		}),
	}
}

func TestGetClient_CIMDURLAllowlist(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("empty allowlist denies without fetching", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			func() []string { return nil },
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
			func() []string { return []string{"https://other.example.com/**"} },
		)
		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("matching allowlist allows", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			func() []string { return []string{"https://8.8.8.8/**"} },
		)
		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, fc.(Client).ID)
	})

	t.Run("refresh denied when not allowlisted", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			func() []string { return nil },
		)
		_, err := s.RefreshMetadataClient(t.Context(), id)
		require.Error(t, err)
	})
}

func TestGetClient_MetadataDocument(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	t.Run("non-URL id falls through to the database", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		_, err := s.GetClient(t.Context(), "does-not-exist")
		require.ErrorIs(t, err, fosite.ErrNotFound)
	})

	t.Run("allowlist changes apply without rebuilding the store", func(t *testing.T) {
		allowlist := []string(nil)
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			func() []string { return allowlist },
		)

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)

		allowlist = []string{"https://8.8.8.8/**"}
		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, fc.(Client).ID)

		allowlist = nil
		_, err = s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("pre-registered URL client takes precedence", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp},
			func() []string { return nil },
		)
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Standard"}
		require.NoError(t, s.db.Create(&seed).Error)

		resolved, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "Standard", resolved.(Client).Name)

		stored, err := s.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "Standard", stored.Name)
		assert.False(t, stored.IsMetadataDocument())
	})

	t.Run("no-store metadata is not persisted", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "no-store")
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		resolved, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, resolved.GetID())

		var count int64
		require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
		assert.Zero(t, count)
	})

	t.Run("private_key_jwt metadata is rejected and not persisted", func(t *testing.T) {
		privateKeyBody := `{"client_id":"https://8.8.8.8/oauth/client","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"private_key_jwt","jwks_uri":"https://8.8.4.4/jwks"}` //nolint:gosec // G101 false positive: authentication method name, not a credential
		resp := testutils.NewMockResponse(http.StatusOK, privateKeyBody)                                                                                                                     //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)

		var count int64
		require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
		assert.Zero(t, count)
	})

	t.Run("cached authenticated metadata client is rejected", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		fresh := datatype.DateTime(time.Now().Add(time.Hour))
		seed := model.OidcClient{
			Base:              model.Base{ID: id},
			ClientType:        model.OidcClientTypeCIMD,
			MetadataExpiresAt: &fresh,
			Credentials: model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{{
				Issuer: id,
				JWKS:   "https://8.8.4.4/jwks",
			}}},
		}
		require.NoError(t, s.db.Create(&seed).Error)

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("cached authenticated metadata client is replaced when the document becomes public", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})
		fresh := datatype.DateTime(time.Now().Add(time.Hour))
		seed := model.OidcClient{
			Base:              model.Base{ID: id},
			ClientType:        model.OidcClientTypeCIMD,
			MetadataExpiresAt: &fresh,
			Credentials: model.OidcClientCredentials{FederatedIdentities: []model.OidcClientFederatedIdentity{{
				Issuer: id,
				JWKS:   "https://8.8.4.4/jwks",
			}}},
		}
		require.NoError(t, s.db.Create(&seed).Error)

		resolved, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		client := resolved.(Client)
		assert.True(t, client.OidcClient.IsPublic)
		assert.True(t, client.PkceEnabled)
		assert.Empty(t, client.Credentials.FederatedIdentities)
	})

	t.Run("fetches, upserts, and reuses the cache", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "max-age=600")
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		fc, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		c := fc.(Client).OidcClient
		assert.Equal(t, id, c.ID)
		assert.True(t, c.IsMetadataDocument())
		assert.True(t, c.IsPublic)
		require.NotNil(t, c.MetadataExpiresAt)

		var count int64
		require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count)

		fc2, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "App", fc2.(Client).Name)
	})

	t.Run("refetch when stale preserves consent", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{Base: model.Base{ID: id}, Name: "Old", IsPublic: true, PkceEnabled: true, ClientType: model.OidcClientTypeCIMD, MetadataExpiresAt: &stale}
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
