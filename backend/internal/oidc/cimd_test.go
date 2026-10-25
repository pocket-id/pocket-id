package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/pocket-id/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/outbound"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestCIMDClientLimitsResponseHeaders(t *testing.T) {
	// Exercise both transport paths so hostname exceptions cannot bypass the header limit
	for _, allowlist := range []string{"loopback", "localhost"} {
		t.Run(allowlist, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				size := 1024
				if r.URL.Path == "/oversized" {
					size = 40 * 1024
				}
				w.Header().Set("X-Padding", strings.Repeat("a", size))
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)

			// Use the production outbound transport and CIMD client construction
			clients, err := outbound.New(&common.EnvConfigSchema{OutboundAllowedHostsClientMetadata: allowlist})
			require.NoError(t, err)
			resolver := newCIMDClientResolver(nil, cimdResolverConfig{transport: clients.Transport(outbound.PurposeClientMetadata)})
			provider, ok := resolver.resolver.Fetcher.(fosite.CIMDSecureHTTPClientProvider)
			require.True(t, ok)
			client := provider.CIMDHTTPClient()
			serverURL := server.URL
			if allowlist == "localhost" {
				serverURL = strings.Replace(serverURL, "127.0.0.1", "localhost", 1)
			}

			// Normal headers must still succeed while oversized headers fail before the body is read
			for _, path := range []string{"/normal", "/oversized"} {
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+path, nil)
				require.NoError(t, err)
				res, err := client.Do(req)
				if res != nil {
					_ = res.Body.Close()
				}
				if path == "/oversized" {
					require.ErrorContains(t, err, "response headers exceeded")
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

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
		assert.Equal(t, []string{"authorization_code"}, []string(c.MetadataGrantTypes))
		assert.Empty(t, c.Credentials.FederatedIdentities)
		assert.Equal(t, model.DefaultAccessTokenDurationMinutes, c.AccessTokenDurationMinutes)
		assert.Equal(t, model.DefaultRefreshTokenDurationMinutes, c.RefreshTokenDurationMinutes)
	})

	t.Run("authenticated clients are rejected", func(t *testing.T) {
		for _, m := range []string{"private_key_jwt", "client_secret_basic", "client_secret_post", "client_secret_jwt"} { //nolint:gosec // G101 false positive: authentication method names, not credentials
			doc := &fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: m}
			_, err := buildClientFromMetadata(doc, id)
			require.Errorf(t, err, "method %q", m)
		}
	})

	t.Run("unsupported grant types are dropped", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{
			ClientID:                id,
			RedirectURIs:            []string{"https://app.example.com/callback"},
			TokenEndpointAuthMethod: "none",
			GrantTypes:              []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:jwt-bearer"},
		}
		c, err := buildClientFromMetadata(doc, id)
		require.NoError(t, err)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, []string(c.MetadataGrantTypes))
	})

	t.Run("documents without a supported initiating grant are rejected", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{
			ClientID:                id,
			RedirectURIs:            []string{"https://app.example.com/callback"},
			TokenEndpointAuthMethod: "none",
			GrantTypes:              []string{"refresh_token", "urn:ietf:params:oauth:grant-type:jwt-bearer"},
		}
		_, err := buildClientFromMetadata(doc, id)
		require.ErrorIs(t, err, errInitiatingGrantRequired)
	})

	t.Run("name falls back to the client ID host", func(t *testing.T) {
		c, err := buildClientFromMetadata(&fosite.ClientMetadataDocument{ClientID: id, TokenEndpointAuthMethod: "none"}, id)
		require.NoError(t, err)
		assert.Equal(t, "app.example.com", c.Name)
	})
}

func TestCIMDPolicyValidate(t *testing.T) {
	policy := cimdPolicy{}

	for _, test := range []struct {
		name          string
		grantTypes    []string
		responseTypes []string
		wantError     string
	}{
		{name: "defaults are supported"},
		{name: "authorization code and refresh token are supported", grantTypes: []string{"authorization_code", "refresh_token"}},
		{name: "device code is supported", grantTypes: []string{string(fosite.GrantTypeDeviceCode)}},
		{name: "unsupported grant types are ignored", grantTypes: []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:jwt-bearer"}},
		{name: "client credentials cannot initiate authorization", grantTypes: []string{"client_credentials"}, wantError: "must enable"},
		{name: "refresh token cannot initiate authorization", grantTypes: []string{"refresh_token"}, wantError: "must enable"},
		{name: "implicit response is rejected", grantTypes: []string{"authorization_code"}, responseTypes: []string{"token"}, wantError: "unsupported response_type"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := policy.ValidateCIMDClient(t.Context(), &fosite.ClientMetadataDocument{
				TokenEndpointAuthMethod: "none",
				GrantTypes:              test.grantTypes,
				ResponseTypes:           test.responseTypes,
			})
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantError)
		})
	}

	t.Run("omitted authentication method is rejected", func(t *testing.T) {
		err := policy.ValidateCIMDClient(t.Context(), &fosite.ClientMetadataDocument{})
		require.ErrorContains(t, err, "token_endpoint_auth_method")
	})
}

func TestMetadataClientChanges(t *testing.T) {
	base := model.OidcClient{
		Name:               "App",
		CallbackURLs:       datatype.StringList{"https://app/cb"},
		LogoutCallbackURLs: datatype.StringList{"https://app/lo"},
		IsPublic:           true,
	}

	t.Run("no changes", func(t *testing.T) {
		assert.Empty(t, metadataClientChanges(base, base))
	})

	t.Run("redirect_uris change", func(t *testing.T) {
		next := base
		next.CallbackURLs = datatype.StringList{"https://app/other"}
		assert.Equal(t, []string{"redirect_uris"}, metadataClientChanges(base, next))
	})

	t.Run("auth method change", func(t *testing.T) {
		next := base
		next.IsPublic = false
		got := metadataClientChanges(base, next)
		assert.Contains(t, got, "token_endpoint_auth_method")
	})

	t.Run("grant types change", func(t *testing.T) {
		next := base
		next.MetadataGrantTypes = datatype.StringList{"authorization_code", "refresh_token"}
		assert.Contains(t, metadataClientChanges(base, next), "grant_types")
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
		seed := model.OidcClient{
			Base:                        model.Base{ID: id},
			Name:                        "Old",
			IsPublic:                    true,
			PkceEnabled:                 true,
			ClientType:                  model.OidcClientTypeCIMD,
			MetadataExpiresAt:           &fresh,
			AccessTokenDurationMinutes:  2 * 60,
			RefreshTokenDurationMinutes: 7 * 24 * 60,
		}
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
		assert.Equal(t, int64(2*60), c.AccessTokenDurationMinutes)
		assert.Equal(t, int64(7*24*60), c.RefreshTokenDurationMinutes)
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
			getURLAllowlist: getAllowlist,
			transport:       &testutils.MockRoundTripper{Responses: responses},
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

	t.Run("no-store metadata is rejected and not persisted", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "no-store")
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)

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

	// A display-only change must not cost the user their consent
	t.Run("refetch when stale preserves consent", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base: model.Base{ID: id}, Name: "Old", IsPublic: true, PkceEnabled: true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{"https://app/cb"},
			MetadataExpiresAt: &stale,
		}
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

	// Whoever controls the document could otherwise repoint an already-consented user's authorization code at a URL of their choosing, with no prompt
	t.Run("changed redirect_uris revoke consent", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base: model.Base{ID: id}, Name: "App", IsPublic: true, PkceEnabled: true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{"https://app/previous-cb"},
			MetadataExpiresAt: &stale,
		}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-1", "openid", time.Now()).Error)

		_, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)

		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").
			Where("client_id = ?", id).Count(&consent).Error)
		assert.Zero(t, consent, "consent must not survive a redirect_uris change")
	})

	t.Run("failed consent revocation rolls back refreshed metadata", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base: model.Base{ID: id}, Name: "App", IsPublic: true, PkceEnabled: true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{"https://app/previous-cb"},
			MetadataExpiresAt: &stale,
		}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-1", "openid", time.Now()).Error)
		require.NoError(t, s.db.Exec(`
			CREATE TRIGGER reject_cimd_consent_delete
			BEFORE DELETE ON user_authorized_oidc_clients
			BEGIN
				SELECT RAISE(ABORT, 'consent deletion blocked');
			END;
		`).Error)

		_, err := s.GetClient(t.Context(), id)
		require.Error(t, err)

		stored, err := s.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, []string{"https://app/previous-cb"}, []string(stored.CallbackURLs))
		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").Where("client_id = ?", id).Count(&consent).Error)
		assert.Equal(t, int64(1), consent)
	})
}

// Pocket ID matches administrator-registered callback URLs as wildcard patterns, so a self-asserted "*" would otherwise match every redirect URI in existence
func TestBuildClientFromMetadata_RejectsPatternRedirectURIs(t *testing.T) {
	const id = "https://app.example.com/oauth/client"

	unsupportedByPocketID := []string{
		"*",
		"https://*.example.com/cb",
		"javascript:alert(1)",
	}
	for _, uri := range unsupportedByPocketID {
		t.Run("redirect_uris "+uri, func(t *testing.T) {
			doc := &fosite.ClientMetadataDocument{
				ClientID:                id,
				RedirectURIs:            []string{uri},
				TokenEndpointAuthMethod: "none",
			}
			_, err := buildClientFromMetadata(doc, id)
			require.Error(t, err)
		})
	}

	// post_logout_redirect_uris go through the same validation
	doc := &fosite.ClientMetadataDocument{
		ClientID:                id,
		RedirectURIs:            []string{"https://app.example.com/cb"},
		PostLogoutRedirectURIs:  []string{"*"},
		TokenEndpointAuthMethod: "none",
	}
	_, err := buildClientFromMetadata(doc, id)
	require.Error(t, err)
}

func TestMatchRedirectURI_MetadataClientExactMatchSucceeds(t *testing.T) {
	metadataClient := Client{OidcClient: model.OidcClient{
		Base:         model.Base{ID: "https://app.example.com/oauth/client"},
		ClientType:   model.OidcClientTypeCIMD,
		CallbackURLs: datatype.StringList{"https://app.example.com/callback"},
	}}

	matched, err := matchRedirectURI("https://app.example.com/callback", metadataClient)
	require.NoError(t, err)
	require.NotNil(t, matched)
	assert.Equal(t, "https://app.example.com/callback", matched.String())
}

// A document declaring only authorization_code must not silently receive refresh_token and device_code
func TestClient_DeclaredCapabilitiesAreEnforced(t *testing.T) {
	t.Run("grant types are restricted to the declaration", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{
			ClientType:         model.OidcClientTypeCIMD,
			IsPublic:           true,
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
		}}
		assert.Equal(t, fosite.Arguments{"authorization_code"}, client.GetGrantTypes())
	})

	t.Run("declared refresh_token is honoured", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{
			ClientType:         model.OidcClientTypeCIMD,
			IsPublic:           true,
			MetadataGrantTypes: datatype.StringList{"authorization_code", "refresh_token"},
		}}
		assert.Equal(t, fosite.Arguments{"authorization_code", "refresh_token"}, client.GetGrantTypes())
	})

	t.Run("an empty declaration uses the RFC default", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{ClientType: model.OidcClientTypeCIMD, IsPublic: true}}
		assert.Equal(t, fosite.Arguments{"authorization_code"}, client.GetGrantTypes())
	})

	t.Run("registered clients are unaffected", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{
			ClientType:         model.OidcClientTypeStandard,
			IsPublic:           true,
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
		}}
		assert.Contains(t, client.GetGrantTypes(), "refresh_token")
	})
}

func TestBuildClientFromMetadata_RecordsDeclaredCapabilities(t *testing.T) {
	const id = "https://app.example.com/oauth/client"

	t.Run("declared values are recorded", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{
			ClientID:                id,
			RedirectURIs:            []string{"https://app.example.com/cb"},
			TokenEndpointAuthMethod: "none",
			GrantTypes:              []string{"authorization_code", "refresh_token"},
		}
		client, err := buildClientFromMetadata(doc, id)
		require.NoError(t, err)
		assert.Equal(t, datatype.StringList{"authorization_code", "refresh_token"}, client.MetadataGrantTypes)
	})

	t.Run("omitted grant_types defaults to authorization_code", func(t *testing.T) {
		doc := &fosite.ClientMetadataDocument{
			ClientID:                id,
			RedirectURIs:            []string{"https://app.example.com/cb"},
			TokenEndpointAuthMethod: "none",
		}
		client, err := buildClientFromMetadata(doc, id)
		require.NoError(t, err)
		assert.Equal(t, datatype.StringList{"authorization_code"}, client.MetadataGrantTypes)
	})
}

// The allowlist is the operator's only gate on which URLs may become clients, and it is matched with the same wildcard syntax as callback URLs
func TestCIMDURLAllowlist_HostilePatterns(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","client_name":"App","redirect_uris":["https://app/cb"],"token_endpoint_auth_method":"none"}`

	denied := []struct {
		name      string
		allowlist []string
	}{
		{"empty list denies", nil},
		{"different host denies", []string{"https://other.example.com/**"}},
		{"different scheme denies", []string{"http://8.8.8.8/**"}},
		{"host as a path segment denies", []string{"https://evil.example/8.8.8.8/**"}},
		{"prefix of the host denies", []string{"https://8.8.8.8.evil.example/**"}},
	}
	for _, tc := range denied {
		t.Run(tc.name, func(t *testing.T) {
			resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
			s := newMetadataStore(t, map[string]*http.Response{id: resp}, func() []string { return tc.allowlist })
			_, err := s.GetClient(t.Context(), id)
			require.ErrorIs(t, err, fosite.ErrInvalidClient)

			var count int64
			require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
			assert.Zero(t, count, "a denied client must never be persisted")
		})
	}

	// "*" matches everything, which is the fully open configuration and the operator's choice rather than a bug
	t.Run("bare wildcard allows everything", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp}, func() []string { return []string{"*"} })
		_, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)
	})
}

// Section 3 constrains the Client Identifier URL, and none of these vectors may reach a fetch
func TestGetClient_HostileClientIDURLs(t *testing.T) {
	hostile := []string{
		"http://8.8.8.8/oauth/client",              // not https
		"https://8.8.8.8",                          // no path component
		"https://user:pass@8.8.8.8/oauth/client",   // userinfo
		"https://8.8.8.8/oauth/client#frag",        // fragment
		"https://8.8.8.8/oauth/client?x=1",         // query component
		"https://8.8.8.8/oauth/../client",          // dot segments
		"https://127.0.0.1/oauth/client",           // loopback
		"https://169.254.169.254/latest/meta-data", // cloud metadata
	}

	for _, id := range hostile {
		t.Run(id, func(t *testing.T) {
			s := newMetadataStore(t, nil)
			_, err := s.GetClient(t.Context(), id)
			require.Error(t, err, "must not be accepted as a Client Identifier URL")

			var count int64
			require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

// Section 5.2 forbids caching error responses or invalid documents, and section 5 requires every non-200 status to be treated as an error
func TestGetClient_MetadataFailuresAreNotPersisted(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"

	cases := []struct {
		name     string
		response *http.Response
	}{
		{"404", testutils.NewMockResponse(http.StatusNotFound, `{}`)},                                                                                               //nolint:bodyclose // mock
		{"500", testutils.NewMockResponse(http.StatusInternalServerError, `{}`)},                                                                                    //nolint:bodyclose // mock
		{"204", testutils.NewMockResponse(http.StatusNoContent, ``)},                                                                                                //nolint:bodyclose // mock
		{"invalid JSON", testutils.NewMockResponse(http.StatusOK, `{not json`)},                                                                                     //nolint:bodyclose // mock
		{"truncated JSON", testutils.NewMockResponse(http.StatusOK, `{"client_id":`)},                                                                               //nolint:bodyclose // mock
		{"empty body", testutils.NewMockResponse(http.StatusOK, ``)},                                                                                                //nolint:bodyclose // mock
		{"null body", testutils.NewMockResponse(http.StatusOK, `null`)},                                                                                             //nolint:bodyclose // mock
		{"client_id mismatch", testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://evil/x"}`)},                                                          //nolint:bodyclose // mock
		{"wrong client_id type", testutils.NewMockResponse(http.StatusOK, `{"client_id":123}`)},                                                                     //nolint:bodyclose // mock
		{"oversize document", testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/client","padding":"`+strings.Repeat("a", 6*1024)+`"}`)}, //nolint:bodyclose // mock
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newMetadataStore(t, map[string]*http.Response{id: tc.response})
			_, err := s.GetClient(t.Context(), id)
			require.Error(t, err)

			var count int64
			require.NoError(t, s.db.Model(&model.OidcClient{}).Where("id = ?", id).Count(&count).Error)
			assert.Zero(t, count, "a failed or invalid document must never be cached")
		})
	}

	t.Run("no response at all", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		_, err := s.GetClient(t.Context(), id)
		require.Error(t, err)
	})
}

func TestCIMDAuthMethodNegotiation(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		primary   string
		supported string
		allowed   bool
	}{
		{"legacy public", "none", "", true},
		{"legacy private", "private_key_jwt", "", false},
		{"explicit public alternative", "private_key_jwt", `["none","private_key_jwt"]`, true},
		{"alternative order is irrelevant", "private_key_jwt", `["private_key_jwt","none"]`, true},
		{"no mutual method", "private_key_jwt", `["private_key_jwt"]`, false},
		{"plural list excludes legacy public", "none", `["private_key_jwt"]`, false},
		{"empty supported methods", "none", `[]`, false},
		{"null supported methods", "none", `null`, false},
		{"supported methods must be array", "none", `"none"`, false},
		{"supported methods must be strings", "none", `["none",1]`, false},
		{"empty method is invalid", "none", `["none",""]`, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			doc := &fosite.ClientMetadataDocument{ClientID: "https://client.example.com/metadata", TokenEndpointAuthMethod: scenario.primary}
			if scenario.supported != "" {
				doc.AdditionalMetadata = map[string]json.RawMessage{"token_endpoint_auth_methods_supported": json.RawMessage(scenario.supported)}
			}
			policyError := (cimdPolicy{}).ValidateCIMDClient(t.Context(), doc)
			client, buildError := buildClientFromMetadata(doc, doc.ClientID)
			if !scenario.allowed {
				require.Error(t, policyError)
				require.Error(t, buildError)
				return
			}
			require.NoError(t, policyError)
			require.NoError(t, buildError)
			require.True(t, client.IsPublic)
			require.True(t, client.PkceEnabled)
			require.Empty(t, client.Credentials.FederatedIdentities)
			require.Equal(t, scenario.primary, doc.TokenEndpointAuthMethod)
		})
	}
}

func TestCIMDNegotiatedPublicClientPersistsAndRevalidates(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"
	body := `{"client_id":"https://8.8.8.8/oauth/client","redirect_uris":["https://app.example.com/callback"],"token_endpoint_auth_method":"private_key_jwt","token_endpoint_auth_methods_supported":["none","private_key_jwt"],"jwks_uri":"https://8.8.4.4/jwks"}`
	response := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response has no network connection
	response.Header.Set("Cache-Control", "max-age=600")
	store := newMetadataStore(t, map[string]*http.Response{id: response})
	resolved, err := store.GetClient(t.Context(), id)
	require.NoError(t, err)
	client := resolved.(Client)
	require.True(t, client.IsPublic())
	require.True(t, client.PkceEnabled)
	require.Empty(t, client.Credentials.FederatedIdentities)
	cached, err := store.GetClient(t.Context(), id)
	require.NoError(t, err)
	require.True(t, cached.(Client).IsPublic())
	require.True(t, cached.(Client).PkceEnabled)
	require.Empty(t, cached.(Client).Credentials.FederatedIdentities)
	// Losing the mutual public method must fail refresh rather than silently retaining the cached authentication policy
	response = testutils.NewMockResponse(http.StatusOK, strings.Replace(body, `["none","private_key_jwt"]`, `["private_key_jwt"]`, 1)) //nolint:bodyclose // mock response has no network connection
	store.resolver.resolver.Fetcher = fosite.NewDefaultCIMDFetcher(fosite.WithCIMDTransport(&testutils.MockRoundTripper{Responses: map[string]*http.Response{id: response}}))
	_, err = store.RefreshMetadataClient(t.Context(), id)
	require.Error(t, err)
}

func TestCIMDNegotiatedPublicClientPKCE(t *testing.T) {
	const clientID = "https://8.8.8.8/oauth/client"
	const callback = "https://app.example.com/callback"
	body := `{"client_id":"https://8.8.8.8/oauth/client","redirect_uris":["https://app.example.com/callback"],"token_endpoint_auth_method":"private_key_jwt","token_endpoint_auth_methods_supported":["none","private_key_jwt"],"jwks_uri":"https://8.8.4.4/jwks"}`
	for _, scenario := range []string{"valid verifier", "missing challenge", "wrong verifier", "missing verifier", "private assertion", "client credentials"} {
		t.Run(scenario, func(t *testing.T) {
			response := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response has no network connection
			store := newMetadataStore(t, map[string]*http.Response{clientID: response})
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)
			provider, err := newProvider(store.Store, &federatedClientAuthenticator{clients: store}, testTokenSigner{key: key}, Config{BaseURL: "https://issuer.example.com", TokenBaseURL: "https://issuer.example.com", Secret: []byte("test-secret")}, store.resolver)
			require.NoError(t, err)
			verifier := strings.Repeat("v", 64)
			challenge := sha256.Sum256([]byte(verifier))
			params := url.Values{"client_id": {clientID}, "redirect_uri": {callback}, "response_type": {"code"}, "scope": {"openid"}, "state": {"state-with-enough-entropy"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}}
			if scenario == "missing challenge" {
				params.Del("code_challenge")
				params.Del("code_challenge_method")
			}
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
			authorize, err := provider.NewAuthorizeRequest(t.Context(), request)
			require.NoError(t, err)
			authorize.GrantScope("openid")
			authorize.GrantAudience(clientID)
			session := NewAuthenticatedSession("synthetic-user", "passkey", time.Now(), time.Now())
			code, err := provider.NewAuthorizeResponse(t.Context(), authorize, session)
			if scenario == "missing challenge" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			form := url.Values{"grant_type": {"authorization_code"}, "code": {code.GetCode()}, "client_id": {clientID}, "redirect_uri": {callback}, "code_verifier": {verifier}}
			switch scenario {
			case "wrong verifier":
				form.Set("code_verifier", strings.Repeat("w", 64))
			case "missing verifier":
				form.Del("code_verifier")
			case "private assertion":
				form.Set("client_assertion_type", clientAssertionTypeJWTBearer)
				assertion, err := jwt.NewBuilder().Issuer(clientID).Subject(clientID).Audience([]string{"https://issuer.example.com/api/oidc/token"}).IssuedAt(time.Now()).Expiration(time.Now().Add(time.Minute)).JwtID("assertion-1").Build()
				require.NoError(t, err)
				signed, err := jwt.Sign(assertion, jwt.WithKey(jwa.ES256(), key))
				require.NoError(t, err)
				form.Set("client_assertion", string(signed))
			case "client credentials":
				form.Set("grant_type", "client_credentials")
			}
			request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://issuer.example.com/api/oidc/token", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			access, err := provider.NewAccessRequest(t.Context(), request, NewEmptySession())
			if scenario != "valid verifier" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			token, err := provider.NewAccessResponse(t.Context(), access)
			require.NoError(t, err)
			require.NotEmpty(t, token.GetAccessToken())
		})
	}
}
