package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/pocket-id/fosite"
	"github.com/pocket-id/fosite/compose"
	fositejwt "github.com/pocket-id/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
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
			resolver := newCIMDClientResolver(&Store{}, cimdResolverConfig{transport: clients.Transport(outbound.PurposeClientMetadata)})
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

// A client ID the outbound policy blocks will never become reachable, so it must not be answered as a temporary failure that invites a retry
func TestCIMDClientBlockedDestinationIsNotTemporary(t *testing.T) {
	clients, err := outbound.New(&common.EnvConfigSchema{})
	require.NoError(t, err)
	store := NewStore(testutils.NewDatabaseForTest(t), nil)
	resolver := newCIMDClientResolver(store, cimdResolverConfig{
		getURLAllowlist: func() []string { return []string{"*"} },
		transport:       clients.Transport(outbound.PurposeClientMetadata),
	})

	_, err = resolver.ResolveClient(t.Context(), "https://10.0.0.1/oauth/client", store.GetClient)
	require.ErrorIs(t, err, fosite.ErrCIMDFetchRefused)
	require.NotErrorIs(t, err, fosite.ErrCIMDTemporarilyUnavailable)
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
		{name: "unsupported response types are ignored", responseTypes: []string{"code", "token"}},
		{name: "response types without code are rejected", responseTypes: []string{"token"}, wantError: "response type"},
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

	for _, test := range []struct {
		name      string
		method    string
		supported []string
		wantError bool
	}{
		{name: "omitted authentication methods declare a public client"},
		{name: "preferred private_key_jwt without a supported list is rejected", method: "private_key_jwt", wantError: true},
		{name: "preferred private_key_jwt with none in the supported list is accepted", method: "private_key_jwt", supported: []string{"none", "private_key_jwt"}},
		{name: "supported list without none is rejected", method: "none", supported: []string{"private_key_jwt"}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := policy.ValidateCIMDClient(t.Context(), &fosite.ClientMetadataDocument{
				TokenEndpointAuthMethod:           test.method,
				TokenEndpointAuthMethodsSupported: test.supported,
			})
			if test.wantError {
				require.ErrorContains(t, err, "token endpoint authentication method")
				return
			}
			require.NoError(t, err)
		})
	}
}

// A client ID is the audience of the client's login tokens, so it must not be one an API accepts
func TestCIMDPolicyRejectsAPIAudienceAsClientID(t *testing.T) {
	const audience = "https://api.example.com/orders"
	policy := cimdPolicy{
		getURLAllowlist: func() []string { return []string{"*"} },
		apiAccess:       userAccess(map[string][]string{audience: nil}),
	}

	require.ErrorContains(t, policy.AllowCIMDClient(t.Context(), audience), "audience of an API")
	require.ErrorContains(t, policy.AllowCIMDClient(t.Context(), audience+"/"), "audience of an API")
	require.NoError(t, policy.AllowCIMDClient(t.Context(), "https://app.example.com/oauth/client"))
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

	// Clients are persisted, so a document that forbids caching is kept for the minimum lifetime instead of being refused
	t.Run("no-store metadata is cached for the minimum lifetime", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		resp.Header.Set("Cache-Control", "no-store")
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		_, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)

		stored, err := s.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		require.NotNil(t, stored.MetadataExpiresAt)
		assert.WithinDuration(t, time.Now().Add(time.Minute), stored.MetadataExpiresAt.ToTime(), 10*time.Second)
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

	t.Run("ChatGPT's metadata resolves to a public client", func(t *testing.T) {
		const chatGPTID = "https://chatgpt.com/oauth/client.json"
		chatGPTBody := `{"client_id":"https://chatgpt.com/oauth/client.json","client_uri":"https://chatgpt.com/","redirect_uris":["https://chatgpt.com/connector_platform_oauth_redirect"],"token_endpoint_auth_method":"private_key_jwt","token_endpoint_auth_methods_supported":["none","private_key_jwt"],"grant_types":["authorization_code","refresh_token","urn:ietf:params:oauth:grant-type:jwt-bearer"],"response_types":["code"],"client_name":"ChatGPT","logo_uri":"https://persistent.oaistatic.com/sonic/misc/openai-logo.png","token_endpoint_auth_signing_alg":"RS256","jwks_uri":"https://chatgpt.com/oauth/jwks.json"}` //nolint:gosec // G101 false positive: authentication method names, not credentials
		resp := testutils.NewMockResponse(http.StatusOK, chatGPTBody)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{chatGPTID: resp})

		_, err := s.GetClient(t.Context(), chatGPTID)
		require.NoError(t, err)

		stored, err := s.firstClientByID(t.Context(), chatGPTID)
		require.NoError(t, err)
		assert.Equal(t, "ChatGPT", stored.Name)
		assert.True(t, stored.IsPublic)
		assert.True(t, stored.PkceEnabled)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, []string(stored.MetadataGrantTypes))
	})

	t.Run("disabled metadata client is refused without fetching but can be refreshed", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, body) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})
		expired := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base:              model.Base{ID: id},
			Name:              "Old",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			MetadataExpiresAt: &expired,
			Disabled:          true,
		}
		require.NoError(t, s.db.Create(&seed).Error)

		_, err := s.GetClient(t.Context(), id)
		require.True(t, apperror.IsCode(err, apperror.CodeOidcClientDisabled), "expected a disabled client error, got %v", err)
		stored, err := s.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "Old", stored.Name, "the expired metadata document must not be fetched for a disabled client")

		refreshed, err := s.RefreshMetadataClient(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "App", refreshed.Name)
		assert.True(t, refreshed.Disabled)
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
		require.NoError(t, s.CreateRefreshTokenSession(t.Context(), "refresh", "", newTestRequester("request", id, "user-1", "")))

		_, err := s.GetClient(t.Context(), id)
		require.NoError(t, err)

		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").
			Where("client_id = ?", id).Count(&consent).Error)
		assert.Zero(t, consent, "consent must not survive a redirect_uris change")
		_, err = s.GetRefreshTokenSession(t.Context(), "refresh", nil)
		require.ErrorIs(t, err, fosite.ErrNotFound, "tokens must not survive the consent they were issued under")
	})

	// A document that names another client ID means the URL no longer belongs to the client users consented to
	t.Run("client_id mismatch revokes consent and tokens", func(t *testing.T) {
		resp := testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/other"}`) //nolint:bodyclose // mock response, no real body
		s := newMetadataStore(t, map[string]*http.Response{id: resp})

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base: model.Base{ID: id}, Name: "App", IsPublic: true, PkceEnabled: true,
			ClientType:        model.OidcClientTypeCIMD,
			MetadataExpiresAt: &stale,
		}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-1", "openid", time.Now()).Error)
		require.NoError(t, s.CreateRefreshTokenSession(t.Context(), "refresh", "", newTestRequester("request", id, "user-1", "")))

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrCIMDClientIDMismatch)

		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").
			Where("client_id = ?", id).Count(&consent).Error)
		assert.Zero(t, consent)
		_, err = s.GetRefreshTokenSession(t.Context(), "refresh", nil)
		require.ErrorIs(t, err, fosite.ErrNotFound)
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
		{"404", testutils.NewMockResponse(http.StatusNotFound, `{}`)},                                                                                                       //nolint:bodyclose // mock
		{"500", testutils.NewMockResponse(http.StatusInternalServerError, `{}`)},                                                                                            //nolint:bodyclose // mock
		{"204", testutils.NewMockResponse(http.StatusNoContent, ``)},                                                                                                        //nolint:bodyclose // mock
		{"invalid JSON", testutils.NewMockResponse(http.StatusOK, `{not json`)},                                                                                             //nolint:bodyclose // mock
		{"truncated JSON", testutils.NewMockResponse(http.StatusOK, `{"client_id":`)},                                                                                       //nolint:bodyclose // mock
		{"empty body", testutils.NewMockResponse(http.StatusOK, ``)},                                                                                                        //nolint:bodyclose // mock
		{"null body", testutils.NewMockResponse(http.StatusOK, `null`)},                                                                                                     //nolint:bodyclose // mock
		{"client_id mismatch", testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://evil/x"}`)},                                                                  //nolint:bodyclose // mock
		{"empty grant_types", testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/client","redirect_uris":["https://app/cb"],"grant_types":[]}`)}, //nolint:bodyclose // mock
		{"wrong client_id type", testutils.NewMockResponse(http.StatusOK, `{"client_id":123}`)},                                                                             //nolint:bodyclose // mock
		{"oversize document", testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/client","padding":"`+strings.Repeat("a", 6*1024)+`"}`)},         //nolint:bodyclose // mock
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

func TestCIMDClientIDMismatch_InvalidatesCacheAndRevokesGrants(t *testing.T) {
	const id = "https://8.8.8.8/oauth/client"

	t.Run("invalidates cache, revokes pending interactions, and blocks stale-on-outage fallback", func(t *testing.T) {
		responses := map[string]*http.Response{
			id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/other"}`), //nolint:bodyclose // mock
		}
		s := newMetadataStore(t, responses)

		stale := datatype.DateTime(time.Now().Add(-time.Hour))
		seed := model.OidcClient{
			Base:              model.Base{ID: id},
			Name:              "App",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{"https://app.example.com/cb"},
			MetadataExpiresAt: &stale,
		}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-1", "openid", time.Now()).Error)
		require.NoError(t, s.CreateRefreshTokenSession(t.Context(), "refresh", "", newTestRequester("request", id, "user-1", "")))
		require.NoError(t, s.db.Create(&InteractionSession{
			Base:        model.Base{ID: "interaction-1"},
			ClientID:    id,
			RequestedAt: datatype.DateTime(time.Now()),
		}).Error)

		_, err := s.GetClient(t.Context(), id)
		require.ErrorIs(t, err, fosite.ErrCIMDClientIDMismatch)

		stored, err := s.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		assert.Nil(t, stored.MetadataExpiresAt, "mismatch must clear metadata_expires_at so stale fallback cannot reuse it")

		var interactions int64
		require.NoError(t, s.db.Model(&InteractionSession{}).Where("client_id = ?", id).Count(&interactions).Error)
		assert.Zero(t, interactions, "pending interaction sessions must be revoked on mismatch")

		// A fresh resolver sharing the same DB (bypassing Fosite's in-memory failure cache) must not serve stale metadata during an outage
		freshStore := &metadataStore{
			Store: s.Store,
			resolver: newCIMDClientResolver(s.Store, cimdResolverConfig{
				getURLAllowlist: func() []string { return []string{"*"} },
				transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
					id: testutils.NewMockResponse(http.StatusServiceUnavailable, ``), //nolint:bodyclose // mock
				}},
			}),
		}
		_, err = freshStore.GetClient(t.Context(), id)
		require.Error(t, err)
	})

	t.Run("repairing an invalidated row with identical metadata still revokes surviving grants even after restart", func(t *testing.T) {
		s := newMetadataStore(t, nil)
		future := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
		seed := model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
			MetadataExpiresAt:  nil, // previously invalidated
		}
		require.NoError(t, s.db.Create(&seed).Error)
		require.NoError(t, s.db.Exec(
			"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
			id, "user-pre-restart", "openid", time.Now()).Error)
		require.NoError(t, s.CreateRefreshTokenSession(t.Context(), "refresh-pre-restart", "", newTestRequester("req-pre-restart", id, "user-pre-restart", "")))

		// A fresh Store with no in-memory state still revokes surviving grants when repairing an invalidated (MetadataExpiresAt == nil) row with identical metadata
		restartedStore := NewStore(s.db, nil)
		_, err := restartedStore.StoreCIMDClient(t.Context(), Client{OidcClient: model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
		}}, nil, future)
		require.NoError(t, err)

		var consent int64
		require.NoError(t, s.db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, "user-pre-restart").Count(&consent).Error)
		assert.Zero(t, consent, "repairing a persisted MetadataExpiresAt==nil row after restart must revoke surviving consent")
		_, err = restartedStore.GetRefreshTokenSession(t.Context(), "refresh-pre-restart", nil)
		require.ErrorIs(t, err, fosite.ErrNotFound, "repairing a persisted MetadataExpiresAt==nil row after restart must revoke surviving tokens")
	})

	t.Run("stale-on-outage fallback does not survive concurrent invalidation", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)
		store := NewStore(db, nil)
		stale := datatype.DateTime(time.Now().Add(-time.Hour).UTC().Truncate(time.Second))
		seed := model.OidcClient{
			Base:              model.Base{ID: id},
			Name:              "App",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{"https://app.example.com/cb"},
			MetadataExpiresAt: &stale,
		}
		require.NoError(t, db.Create(&seed).Error)

		var invalidated atomic.Bool
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Simulate a concurrent mismatch invalidation committing while the outage fetch is in flight
				if invErr := store.invalidateCIMDClientCache(req.Context(), id); invErr != nil {
					return nil, invErr
				}
				invalidated.Store(true)
				return testutils.NewMockResponse(http.StatusServiceUnavailable, ``), nil //nolint:bodyclose // mock
			}),
		})
		_, err := resolver.ResolveClient(t.Context(), id, store.GetClient)
		require.True(t, invalidated.Load())
		require.ErrorIs(t, err, fosite.ErrInvalidClient, "stale-on-outage fallback must be rejected once metadata_expires_at is invalidated")
	})

	t.Run("stale-on-outage fallback returns newer metadata committed during discovery and rejects incompatible replacement rows", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)
		store := NewStore(db, nil)
		stale := datatype.DateTime(time.Now().Add(-time.Hour).UTC().Truncate(time.Second))
		seed := model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "Old App Name",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
			MetadataExpiresAt:  &stale,
		}
		require.NoError(t, db.Create(&seed).Error)

		var refreshed atomic.Bool
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Simulate another refresh committing a newer document while this request hits a 503 outage
				updated := Client{OidcClient: model.OidcClient{
					Base:               model.Base{ID: id},
					Name:               "New App Name",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
				}}
				if _, storeErr := store.StoreCIMDClient(req.Context(), updated, nil, time.Now().Add(time.Hour).UTC().Truncate(time.Second)); storeErr != nil {
					return nil, storeErr
				}
				refreshed.Store(true)
				return testutils.NewMockResponse(http.StatusServiceUnavailable, ``), nil //nolint:bodyclose // mock
			}),
		})
		resolved, err := resolver.ResolveClient(t.Context(), id, store.GetClient)
		require.True(t, refreshed.Load())
		require.NoError(t, err)
		pocketClient, ok := resolved.(Client)
		require.True(t, ok)
		assert.Equal(t, "New App Name", pocketClient.Name, "ResolveClient must return the newly committed metadata rather than the superseded fallback snapshot")

		// Make the row stale again and simulate a concurrent edit that leaves it incompatible (pkce_enabled = false) during an outage
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &stale).Error)
		incompatResolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if err := db.WithContext(req.Context()).Model(&model.OidcClient{}).Where("id = ?", id).Update("pkce_enabled", false).Error; err != nil {
					return nil, err
				}
				return testutils.NewMockResponse(http.StatusServiceUnavailable, ``), nil //nolint:bodyclose // mock
			}),
		})
		_, err = incompatResolver.ResolveClient(t.Context(), id, store.GetClient)
		require.ErrorIs(t, err, fosite.ErrInvalidClient, "ResolveClient must reject an incompatible replacement row during stale-on-outage fallback")
	})

	t.Run("expired beyond stale window returns fresh metadata committed during a 503 outage without masking permanent failures", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)
		store := NewStore(db, nil)
		// 48h expired is outside the 24h StaleIfUnavailable window, so Fosite returns ErrCIMDTemporarilyUnavailable
		ancient := datatype.DateTime(time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second))
		require.NoError(t, db.Create(&model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "Ancient App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
			MetadataExpiresAt:  &ancient,
		}).Error)

		var refreshed atomic.Bool
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				updated := Client{OidcClient: model.OidcClient{
					Base:               model.Base{ID: id},
					Name:               "Fresh App",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
				}}
				if _, storeErr := store.StoreCIMDClient(req.Context(), updated, nil, time.Now().Add(time.Hour).UTC().Truncate(time.Second)); storeErr != nil {
					return nil, storeErr
				}
				refreshed.Store(true)
				return testutils.NewMockResponse(http.StatusServiceUnavailable, ``), nil //nolint:bodyclose // mock
			}),
		})

		resolved, err := resolver.ResolveClient(t.Context(), id, store.GetClient)
		require.True(t, refreshed.Load())
		require.NoError(t, err, "ResolveClient must return fresh metadata committed during a 503 even when the prior entry was outside StaleIfUnavailable")
		pocketClient, ok := resolved.(Client)
		require.True(t, ok)
		assert.Equal(t, "Fresh App", pocketClient.Name)

		// A permanent refusal (e.g. HTTP 404) must NEVER be masked by a fresh DB row committed during discovery
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &ancient).Error)
		permanentResolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				updated := Client{OidcClient: model.OidcClient{
					Base:               model.Base{ID: id},
					Name:               "Fresh App 2",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
				}}
				if _, storeErr := store.StoreCIMDClient(req.Context(), updated, nil, time.Now().Add(time.Hour).UTC().Truncate(time.Second)); storeErr != nil {
					return nil, storeErr
				}
				return testutils.NewMockResponse(http.StatusNotFound, ``), nil //nolint:bodyclose // mock
			}),
		})
		_, err = permanentResolver.ResolveClient(t.Context(), id, store.GetClient)
		require.Error(t, err, "permanent 404 discovery failure must not be masked by a concurrently stored row")
		require.NotErrorIs(t, err, fosite.ErrCIMDTemporarilyUnavailable)
	})

	t.Run("session lookup rejects access token deleted during CIMD metadata repair in the same lookup", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)
		store := NewStore(db, nil)
		seed := model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
			MetadataExpiresAt:  nil, // previously invalidated
		}
		require.NoError(t, db.Create(&seed).Error)
		require.NoError(t, store.CreateAccessTokenSession(t.Context(), "access-sig-repair", newTestRequester("req-repair", id, "user-1", "")))

		docJSON := `{"client_id":"` + id + `","client_name":"App","redirect_uris":["https://app.example.com/cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
				id: testutils.NewMockResponse(http.StatusOK, docJSON), //nolint:bodyclose // mock
			}},
		})
		store.clientResolver = resolver

		_, err := store.GetAccessTokenSession(t.Context(), "access-sig-repair", nil)
		require.ErrorIs(t, err, fosite.ErrNotFound, "session deleted by CIMD repair during lookup must not be returned as active")
	})

	t.Run("failed grant deletion persists NULL expiry to file-backed SQLite and blocks surviving browser and device consent", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)
		require.NoError(t, claimmappingpolicy.Bootstrap(t.Context(), db))
		store := NewStore(db, nil)
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
		})

		future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
		seed := model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			LogoutCallbackURLs: datatype.StringList{"https://app.example.com/logout-cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code", string(fosite.GrantTypeDeviceCode)},
			MetadataExpiresAt:  &future,
		}
		require.NoError(t, db.Create(&seed).Error)
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "user-interaction"}, Username: "interactor"}).Error)
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
			ClientID:   id,
			UserID:     "user-surviving-logout",
			Scope:      datatype.StringList{"openid"},
			LastUsedAt: datatype.DateTime(time.Now()),
		}).Error)
		require.NoError(t, db.Create(&InteractionSession{
			Base:            model.Base{ID: "interaction-surviving"},
			ClientID:        id,
			Scopes:          datatype.StringList{"openid"},
			ConsentRequired: true,
			RequestedAt:     datatype.DateTime(time.Now()),
			Parameters:      InteractionSessionParameters{"scope": "openid"},
		}).Error)

		const issuer = "https://issuer.example.com"
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		provider, err := newProvider(store, nil, testTokenSigner{key: key}, Config{
			BaseURL: issuer, TokenBaseURL: issuer, Secret: []byte("test-secret"),
		}, nil)
		require.NoError(t, err)
		claimsSvc := newClaimsService(db, nil, issuer, nil, appconfig.StaticResolver{})
		authSvc := newAuthorizationService(db, newInteractionSessionService(db), claimsSvc, nil, &fakeAuditLogger{}, nil)
		authSvc.validateCIMDClient = resolver.policy.AllowCIMDClient
		devSvc := newDeviceService(provider, store, provider.deviceStrategy, authSvc, claimsSvc, &fakeAuditLogger{}, db)

		// Mint a pending device code session before the mismatch occurs
		userCode, userCodeSig, err := provider.deviceStrategy.GenerateUserCode(t.Context())
		require.NoError(t, err)
		_, deviceCodeSig, err := provider.deviceStrategy.GenerateDeviceCode(t.Context())
		require.NoError(t, err)
		devReq := fosite.NewDeviceRequest()
		devReq.ID = "device-req-surviving"
		devReq.RequestedAt = time.Now().UTC()
		devReq.Client = Client{OidcClient: seed}
		devReq.RequestedScope = fosite.Arguments{"openid"}
		devReq.GrantedScope = fosite.Arguments{"openid"}
		devReq.Session = NewEmptySession()
		devReq.Session.SetExpiresAt(fosite.DeviceCode, time.Now().UTC().Add(15*time.Minute))
		devReq.Session.SetExpiresAt(fosite.UserCode, time.Now().UTC().Add(15*time.Minute))
		require.NoError(t, store.CreateDeviceAuthSession(t.Context(), deviceCodeSig, userCodeSig, devReq))

		// Inject a one-shot failure on oauth2_sessions deletion so revokeClientGrants fails while fallback cache invalidation succeeds
		var injectedFail atomic.Bool
		injectedFail.Store(true)
		cbName := "test:fail_oauth2_sessions_delete_once"
		require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(cbName, func(tx *gorm.DB) {
			if tx.Statement != nil && tx.Statement.Table == "oauth2_sessions" && injectedFail.CompareAndSwap(true, false) {
				_ = tx.AddError(errors.New("simulated oauth2_sessions delete failure"))
			}
		}))
		t.Cleanup(func() {
			_ = db.Callback().Delete().Remove(cbName)
		})

		revErr := resolver.revokeOnClientIDMismatch(t.Context(), id, fosite.ErrCIMDClientIDMismatch)
		require.ErrorIs(t, revErr, fosite.ErrCIMDClientIDMismatch)

		// A fresh Store reading the file-backed SQLite database must see NULL metadata_expires_at on disk
		freshStore := NewStore(db, nil)
		stored, err := freshStore.firstClientByID(t.Context(), id)
		require.NoError(t, err)
		assert.Nil(t, stored.MetadataExpiresAt, "fallback must persist NULL metadata_expires_at to disk when grant deletion fails")

		// A surviving browser interaction must NOT record consent for the invalidated client
		_, stepErr := authSvc.completeInteractionStep(t.Context(), "interaction-surviving", "user-interaction", interactionStepConsent, []string{"openid"}, "", time.Now().UTC(), requestMeta{})
		require.ErrorIs(t, stepErr, fosite.ErrInvalidClient)

		// A surviving device authorization approval must NOT record consent for the invalidated client
		devErr := devSvc.acceptDeviceCode(t.Context(), userCode, "user-interaction", "phr", time.Now().UTC(), nil, "", requestMeta{})
		require.ErrorIs(t, devErr, fosite.ErrInvalidClient)

		var consent int64
		require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, "user-interaction").Count(&consent).Error)
		assert.Zero(t, consent, "invalidated CIMD client must not gain new consent via browser or device approval")

		// Surviving consent for an invalidated CIMD client must not redirect RP-initiated logout to its untrusted post_logout_redirect_uri
		endSessionSvc := newEndSessionService(db, store, testTokenSigner{key: key}, "https://issuer.example.com")
		idTok, err := jwt.NewBuilder().
			Issuer("https://issuer.example.com").
			Subject("user-surviving-logout").
			Audience([]string{id}).
			JwtID("jti-logout-1").
			IssuedAt(time.Now().UTC()).
			Claim(common.TokenTypeClaim, string(IDTokenType)).
			Build()
		require.NoError(t, err)
		signedHint, err := jwt.Sign(idTok, jwt.WithKey(jwa.ES256(), key))
		require.NoError(t, err)
		endReq, err := endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "https://app.example.com/logout-cb",
		}, "user-surviving-logout")
		require.NoError(t, err)
		assert.Empty(t, endReq.callbackURL, "logout must suppress post_logout_redirect_uri for an invalidated CIMD client")

		// Repairing the invalidated row via a fresh Store with identical metadata must revoke surviving pre-mismatch consent
		_, err = freshStore.StoreCIMDClient(t.Context(), Client{OidcClient: model.OidcClient{
			Base:               model.Base{ID: id},
			Name:               "App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
			LogoutCallbackURLs: datatype.StringList{"https://app.example.com/logout-cb"},
			MetadataGrantTypes: datatype.StringList{"authorization_code"},
		}}, nil, future.ToTime())
		require.NoError(t, err)
		var survivingConsent int64
		require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, "user-surviving-logout").Count(&survivingConsent).Error)
		assert.Zero(t, survivingConsent, "repairing the invalidated row must revoke surviving pre-mismatch consent")

		// Once the CIMD client is active again with fresh consent, post_logout_redirect_uri must enforce strict exact matching (rejecting fragments, non-exact URIs, and unregistered loopback ports)
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("logout_callback_urls", datatype.StringList{
			"https://app.example.com/logout-cb",
			"http://127.0.0.1:8080/logout-cb",
		}).Error)
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
			ClientID:   id,
			UserID:     "user-surviving-logout",
			Scope:      datatype.StringList{"openid"},
			LastUsedAt: datatype.DateTime(time.Now()),
		}).Error)

		activeEndReq, err := endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "https://app.example.com/logout-cb",
		}, "user-surviving-logout")
		require.NoError(t, err)
		assert.Equal(t, "https://app.example.com/logout-cb", activeEndReq.callbackURL)

		exactLoopbackReq, err := endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "http://127.0.0.1:8080/logout-cb",
		}, "user-surviving-logout")
		require.NoError(t, err)
		assert.Equal(t, "http://127.0.0.1:8080/logout-cb", exactLoopbackReq.callbackURL)

		for _, badLogoutURI := range []string{
			"https://app.example.com/logout-cb#frag",
			"HTTPS://app.example.com/logout-cb",
			"https://app.example.com/logout-cb?extra=1",
			"http://127.0.0.1:49152/logout-cb",
		} {
			_, badErr := endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
				IdTokenHint:           string(signedHint),
				PostLogoutRedirectUri: badLogoutURI,
			}, "user-surviving-logout")
			require.Error(t, badErr, "active CIMD logout must reject non-exact post_logout_redirect_uri %q", badLogoutURI)
		}

		// Policy allowlist removal must suppress post_logout_redirect_uri even with surviving consent
		endSessionSvc.validateCIMDClient = func(context.Context, string) error {
			return errors.New("removed from allowlist")
		}
		disallowedEndReq, err := endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "https://app.example.com/logout-cb",
		}, "user-surviving-logout")
		require.NoError(t, err)
		assert.Empty(t, disallowedEndReq.callbackURL, "logout must suppress post_logout_redirect_uri when CIMD policy rejects the client")

		// Expired metadata (1h old, within the 24h outage window) must trigger discovery on logout rather than blindly extending the old callback for 24h
		staleOneHour := datatype.DateTime(time.Now().Add(-time.Hour).UTC().Truncate(time.Second))
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &staleOneHour).Error)
		logoutResolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
				id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"`+id+`","client_name":"App","redirect_uris":["https://app.example.com/cb"],"post_logout_redirect_uris":["https://app.example.com/new-logout"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`), //nolint:bodyclose // mock
			}},
		})
		store.clientResolver = logoutResolver
		endSessionSvc.validateCIMDClient = logoutResolver.policy.AllowCIMDClient
		_, err = endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "https://app.example.com/logout-cb",
		}, "user-surviving-logout")
		require.ErrorIs(t, err, apperror.OidcMissingAuthorization(), "logout must reject automatic session end once discovery revokes consent")

		// A client_id mismatch during expired-metadata logout discovery also revokes consent and returns an error; logout must still return OidcMissingAuthorization() rather than ending the browser session
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &staleOneHour).Error)
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
			ClientID:   id,
			UserID:     "user-surviving-logout",
			Scope:      datatype.StringList{"openid"},
			LastUsedAt: datatype.DateTime(time.Now()),
		}).Error)
		mismatchLogoutResolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
				id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/other"}`), //nolint:bodyclose // mock
			}},
		})
		store.clientResolver = mismatchLogoutResolver
		endSessionSvc.validateCIMDClient = mismatchLogoutResolver.policy.AllowCIMDClient
		_, err = endSessionSvc.resolve(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           string(signedHint),
			PostLogoutRedirectUri: "https://app.example.com/logout-cb",
		}, "user-surviving-logout")
		require.ErrorIs(t, err, apperror.OidcMissingAuthorization(), "logout must return OidcMissingAuthorization when discovery mismatch revokes consent")

		// Browser consent must also resolve expired metadata (within 24h stale window) before recording consent rather than blindly accepting it
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &staleOneHour).Error)
		consentResolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string { return []string{"*"} },
			transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
				id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"`+id+`","client_name":"App","redirect_uris":["https://app.example.com/changed-cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`), //nolint:bodyclose // mock
			}},
		})
		store.clientResolver = consentResolver
		authSvc.resolveCIMDClient = store.resolvePersistedClient
		require.NoError(t, db.Create(&InteractionSession{
			Base:            model.Base{ID: "interaction-expired-refresh"},
			ClientID:        id,
			Scopes:          datatype.StringList{"openid"},
			ConsentRequired: true,
			RequestedAt:     datatype.DateTime(time.Now()),
			Parameters:      InteractionSessionParameters{"scope": "openid"},
		}).Error)
		_, expiredConsentErr := authSvc.completeInteractionStep(t.Context(), "interaction-expired-refresh", "user-interaction", interactionStepConsent, []string{"openid"}, "", time.Now().UTC(), requestMeta{})
		require.ErrorIs(t, expiredConsentErr, apperror.OidcInteractionNotFound(), "browser consent must resolve expired metadata and reject once discovery revokes the pending interaction")

		// Browser consent must also reject an interaction once CIMD metadata crosses the 24h stale deadline
		authSvc.resolveCIMDClient = nil
		expiredBeyondStale := datatype.DateTime(time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second))
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &expiredBeyondStale).Error)
		require.NoError(t, db.Create(&InteractionSession{
			Base:            model.Base{ID: "interaction-stale-deadline"},
			ClientID:        id,
			Scopes:          datatype.StringList{"openid"},
			ConsentRequired: true,
			RequestedAt:     datatype.DateTime(time.Now()),
			Parameters:      InteractionSessionParameters{"scope": "openid"},
		}).Error)
		_, deadlineErr := authSvc.completeInteractionStep(t.Context(), "interaction-stale-deadline", "user-interaction", interactionStepConsent, []string{"openid"}, "", time.Now().UTC(), requestMeta{})
		require.ErrorIs(t, deadlineErr, fosite.ErrInvalidClient, "browser consent must be rejected once CIMD metadata crosses the stale deadline")
	})

	t.Run("mismatch cleanup inside a rolled-back caller transaction persists invalidation across restarts and fresh stores", func(t *testing.T) {
		testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
			const rollbackUserID = "33333333-3333-3333-3333-333333333333"
			store := NewStore(db, nil)
			future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
			require.NoError(t, db.Create(&model.User{Base: model.Base{ID: rollbackUserID}, Username: "rollback-user"}).Error)
			require.NoError(t, db.Create(&model.OidcClient{
				Base:               model.Base{ID: id},
				Name:               "App",
				IsPublic:           true,
				PkceEnabled:        true,
				ClientType:         model.OidcClientTypeCIMD,
				CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
				MetadataGrantTypes: datatype.StringList{"authorization_code"},
				MetadataExpiresAt:  &future,
			}).Error)
			require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
				ClientID:   id,
				UserID:     rollbackUserID,
				Scope:      datatype.StringList{"openid"},
				LastUsedAt: datatype.DateTime(time.Now()),
			}).Error)
			resolver := newCIMDClientResolver(store, cimdResolverConfig{
				getURLAllowlist: func() []string { return []string{"*"} },
				transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
					id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/other"}`), //nolint:bodyclose // mock
				}},
			})

			sentinel := errors.New("rollback outer tx")
			err := withTx(t.Context(), db, func(txCtx context.Context) error {
				_ = resolver.revokeOnClientIDMismatch(txCtx, id, fosite.ErrCIMDClientIDMismatch)
				return sentinel
			})
			require.ErrorIs(t, err, sentinel)

			// Verify durability from a brand-new Store and resolver (simulating a process restart or another instance)
			freshStore := NewStore(db, nil)
			stored, err := freshStore.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, stored.MetadataExpiresAt, "post-rollback cleanup must persist NULL metadata_expires_at to disk")

			var consent int64
			require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, rollbackUserID).Count(&consent).Error)
			assert.Zero(t, consent, "post-rollback cleanup must revoke grants durably on disk")

			freshResolver := newCIMDClientResolver(freshStore, cimdResolverConfig{
				getURLAllowlist: func() []string { return []string{"*"} },
				transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
					id: testutils.NewMockResponse(http.StatusServiceUnavailable, ``), //nolint:bodyclose // mock
				}},
			})
			_, err = freshResolver.ResolveClient(t.Context(), id, freshStore.GetClient)
			require.Error(t, err, "a fresh Store after restart must not reuse the pre-mismatch fresh cache")
			assert.False(t, store.isCIMDInvalidated(id))

			// Calling RefreshMetadataClient inside an enclosing transaction must schedule at most one rollback cleanup callback even though both fetcher and caller inspect the mismatch error
			require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &future).Error)
			var rollbackCount int
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				_, refreshErr := resolver.RefreshMetadataClient(txCtx, id)
				require.ErrorIs(t, refreshErr, fosite.ErrCIMDClientIDMismatch)
				hooks := txHooksFromContext(txCtx)
				require.NotNil(t, hooks)
				hooks.mu.Lock()
				rollbackCount = len(hooks.postTx)
				hooks.mu.Unlock()
				return sentinel
			})
			require.ErrorIs(t, err, sentinel)
			assert.Equal(t, 1, rollbackCount, "one mismatch inside an enclosing transaction must schedule rollback cleanup at most once")

			// Late async rollback callback (e.g. Fosite singleflight finishing after caller cancellation already rolled back withTx) must execute immediately and persist invalidation
			require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", id).Update("metadata_expires_at", &future).Error)
			var rolledBackTxCtx context.Context
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				rolledBackTxCtx = txCtx
				return sentinel
			})
			require.ErrorIs(t, err, sentinel)
			_ = resolver.revokeOnClientIDMismatch(rolledBackTxCtx, id, fosite.ErrCIMDClientIDMismatch)
			stored, err = freshStore.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, stored.MetadataExpiresAt, "late rollback callback registered after withTx already rolled back must still persist NULL metadata_expires_at")
			assert.False(t, store.isCIMDInvalidated(id))
		})
	})

	t.Run("failed mismatch revocation and fallback DB invalidation still blocks fresh-cache reuse until recovery", func(t *testing.T) {
		testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
			if db.Name() == "sqlite" {
				db = testutils.NewConcurrentDatabaseForTest(t)
			}
			const (
				outageUserID            = "44444444-4444-4444-4444-444444444444"
				postRepairInteractionID = "55555555-5555-5555-5555-555555555555"
			)
			store := NewStore(db, nil)
			future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
			require.NoError(t, db.Create(&model.User{Base: model.Base{ID: outageUserID}, Username: "outage-user"}).Error)
			require.NoError(t, db.Create(&model.OidcClient{
				Base:               model.Base{ID: id},
				Name:               "App",
				IsPublic:           true,
				PkceEnabled:        true,
				ClientType:         model.OidcClientTypeCIMD,
				CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
				MetadataGrantTypes: datatype.StringList{"authorization_code"},
				MetadataExpiresAt:  &future,
			}).Error)
			require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
				ClientID:   id,
				UserID:     outageUserID,
				Scope:      datatype.StringList{"openid"},
				LastUsedAt: datatype.DateTime(time.Now()),
			}).Error)

			var blockDBWrites atomic.Bool
			blockDBWrites.Store(true)
			delCB := "test:fail_all_mismatch_deletes"
			updCB := "test:fail_all_mismatch_updates"
			require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(delCB, func(tx *gorm.DB) {
				if blockDBWrites.Load() && tx.Statement != nil && tx.Statement.Table == "oauth2_sessions" {
					_ = tx.AddError(errors.New("simulated DB write outage on delete"))
				}
			}))
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register(updCB, func(tx *gorm.DB) {
				if blockDBWrites.Load() && tx.Statement != nil && tx.Statement.Table == "oidc_clients" {
					_ = tx.AddError(errors.New("simulated DB write outage on update"))
				}
			}))
			t.Cleanup(func() {
				_ = db.Callback().Delete().Remove(delCB)
				_ = db.Callback().Update().Remove(updCB)
			})

			resolver := newCIMDClientResolver(store, cimdResolverConfig{
				getURLAllowlist: func() []string { return []string{"*"} },
				transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
					id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"https://8.8.8.8/oauth/other"}`), //nolint:bodyclose // mock
				}},
			})
			store.clientResolver = resolver

			// Forced refresh detects mismatch while DB writes fail for both primary revocation and fallback invalidation
			_, err := resolver.RefreshMetadataClient(t.Context(), id)
			require.ErrorIs(t, err, fosite.ErrCIMDClientIDMismatch)

			// Even though metadata_expires_at is still non-NULL in DB, LoadCIMDClient must suppress expiry so Fosite cannot take the fresh-cache fast path
			stored, err := store.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			require.NotNil(t, stored.MetadataExpiresAt)
			cached, found, err := store.LoadCIMDClient(t.Context(), id)
			require.NoError(t, err)
			require.True(t, found)
			assert.True(t, cached.ExpiresAt.IsZero(), "failed DB invalidation must suppress fresh-cache expiry")

			_, err = resolver.ResolveClient(t.Context(), id, store.GetClient)
			require.ErrorIs(t, err, fosite.ErrCIMDClientIDMismatch, "subsequent lookup must not reuse the stale fresh cache entry")

			// Once DB recovers, resolving and repairing via ResolveClient inside an enclosing committed transaction must succeed, revoke surviving consent, and clear the failure marker on commit
			blockDBWrites.Store(false)
			repairResolver := newCIMDClientResolver(store, cimdResolverConfig{
				getURLAllowlist: func() []string { return []string{"*"} },
				transport: &testutils.MockRoundTripper{Responses: map[string]*http.Response{
					id: testutils.NewMockResponse(http.StatusOK, `{"client_id":"`+id+`","client_name":"App","redirect_uris":["https://app.example.com/cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`), //nolint:bodyclose // mock
				}},
			})
			var resolvedInTx fosite.Client
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				var resolveErr error
				resolvedInTx, resolveErr = repairResolver.ResolveClient(txCtx, id, store.GetClient)
				if resolveErr != nil {
					return resolveErr
				}
				// Create a new interaction after the first repair and perform a second identical StoreCIMDClient in the same tx; it must NOT re-revoke the newly created interaction
				if createErr := dbFromContext(txCtx, db).Create(&InteractionSession{
					Base:        model.Base{ID: postRepairInteractionID},
					ClientID:    id,
					Scopes:      datatype.StringList{"openid"},
					RequestedAt: datatype.DateTime(time.Now()),
					Parameters:  InteractionSessionParameters{},
				}).Error; createErr != nil {
					return createErr
				}
				_, storeErr := store.StoreCIMDClient(txCtx, resolvedInTx, nil, future.ToTime())
				return storeErr
			})
			require.NoError(t, err, "ResolveClient repair and second identical refresh inside an enclosing transaction must succeed")
			require.NotNil(t, resolvedInTx)
			assert.False(t, store.isCIMDInvalidated(id), "committed enclosing repair must clear the in-memory failure marker")

			var postRepairInteractions int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", postRepairInteractionID).Count(&postRepairInteractions).Error)
			assert.Equal(t, int64(1), postRepairInteractions, "a second identical refresh in the same transaction must not revoke interactions created after the first repair")

			// Exercise the exact race window between DB commit and post-commit marker clearing: synchronize on the second refresh reaching waitPendingCIMDClear before releasing the pending clear
			enteredWait := make(chan struct{}, 1)
			store.cimdInvalidationMu.Lock()
			store.onWaitPendingCIMDClear = func() {
				select {
				case enteredWait <- struct{}{}:
				default:
				}
			}
			store.cimdInvalidationMu.Unlock()
			t.Cleanup(func() {
				store.cimdInvalidationMu.Lock()
				store.onWaitPendingCIMDClear = nil
				store.cimdInvalidationMu.Unlock()
			})

			store.markCIMDInvalidationFailed(id)
			pendingCh, pendingGen, err := store.acquireCIMDPendingClear(t.Context(), id)
			require.NoError(t, err)
			refreshDone := make(chan error, 1)
			go func() {
				_, storeErr := store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
				refreshDone <- storeErr
			}()
			select {
			case <-enteredWait:
			case earlyErr := <-refreshDone:
				t.Fatalf("second refresh must enter waitPendingCIMDClear before completing, returned early: %v", earlyErr)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for second refresh to reach waitPendingCIMDClear")
			}
			store.finishCIMDClear(id, pendingGen, pendingCh, true, true)
			select {
			case refreshErr := <-refreshDone:
				require.NoError(t, refreshErr)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for second refresh to complete after finishCIMDClear")
			}
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", postRepairInteractionID).Count(&postRepairInteractions).Error)
			assert.Equal(t, int64(1), postRepairInteractions, "a second identical refresh racing the commit-to-clear window must wait for marker clear and preserve post-repair interactions")

			// Verify ResolveClient rechecks AllowCIMDClient before returning when the allowlist removes the client during in-flight/shared discovery
			var allowInFlight atomic.Bool
			allowInFlight.Store(true)
			const removedID = "https://8.8.8.8/oauth/removed-during-discovery"
			inFlightResolver := newCIMDClientResolver(store, cimdResolverConfig{
				getURLAllowlist: func() []string {
					if allowInFlight.Load() {
						return []string{"*"}
					}
					return nil
				},
				transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
					allowInFlight.Store(false)
					return testutils.NewMockResponse(http.StatusOK, `{"client_id":"`+removedID+`","client_name":"Removed","redirect_uris":["https://removed.example.com/cb"],"grant_types":["authorization_code"],"token_endpoint_auth_method":"none"}`), nil //nolint:bodyclose // mock
				}),
			})
			_, err = inFlightResolver.ResolveClient(t.Context(), removedID, store.GetClient)
			require.ErrorIs(t, err, fosite.ErrInvalidClient, "ResolveClient must reject a CIMD client whose allowlist entry was removed during discovery")

			repairedCached, found, err := store.LoadCIMDClient(t.Context(), id)
			require.NoError(t, err)
			require.True(t, found)
			assert.False(t, repairedCached.ExpiresAt.IsZero(), "committed enclosing repair must restore non-zero cache expiry")

			var consent int64
			require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, outageUserID).Count(&consent).Error)
			assert.Zero(t, consent, "repair after failed DB invalidation must still revoke surviving pre-mismatch consent")

			// If mismatch cleanup fails at consent deletion inside an enclosing transaction and the caller still commits, afterTxSettledOnce must retry and durably delete both consent and metadata_expires_at
			require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
				ClientID:   id,
				UserID:     outageUserID,
				Scope:      datatype.StringList{"openid"},
				LastUsedAt: datatype.DateTime(time.Now()),
			}).Error)
			var failConsentDeleteOnce atomic.Bool
			failConsentDeleteOnce.Store(true)
			onceDelCB := "test:fail_in_tx_mismatch_consent_once"
			require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(onceDelCB, func(tx *gorm.DB) {
				if tx.Statement != nil && tx.Statement.Table == "user_authorized_oidc_clients" && failConsentDeleteOnce.CompareAndSwap(true, false) {
					_ = tx.AddError(errors.New("simulated transient consent delete failure inside caller tx"))
				}
			}))
			t.Cleanup(func() {
				_ = db.Callback().Delete().Remove(onceDelCB)
			})

			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				_ = resolver.revokeOnClientIDMismatch(txCtx, id, fosite.ErrCIMDClientIDMismatch)
				// Caller handles the error and commits the enclosing transaction
				return nil
			})
			require.NoError(t, err)
			freshStoreAfterCommit := NewStore(db, nil)
			storedAfterCommit, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, storedAfterCommit.MetadataExpiresAt, "failed mismatch cleanup inside a committed enclosing tx must still retry and persist NULL metadata_expires_at")
			require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", id, outageUserID).Count(&consent).Error)
			assert.Zero(t, consent, "post-commit retry itself must durably revoke surviving consent")

			// A failed StoreCIMDClient repair inside a caller-owned committed transaction must roll back its savepoint rather than committing fresh metadata alongside surviving grants
			require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{
				ClientID:   id,
				UserID:     outageUserID,
				Scope:      datatype.StringList{"openid"},
				LastUsedAt: datatype.DateTime(time.Now()),
			}).Error)
			failConsentDeleteOnce.Store(true)
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				_, storeErr := store.StoreCIMDClient(txCtx, resolvedInTx, nil, future.ToTime())
				require.Error(t, storeErr)
				// Caller handles the repair error and commits unrelated work
				return nil
			})
			require.NoError(t, err)
			storedAfterFailedRepair, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, storedAfterFailedRepair.MetadataExpiresAt, "failed repair inside a committed caller tx must roll back its savepoint and leave metadata_expires_at NULL")

			// If mismatch cleanup fails and a valid repair succeeds later in the same committed transaction, the queued commit retry must be disarmed and later replays of that cached error must be suppressed
			const disarmedInteractionID = "66666666-6666-6666-6666-666666666666"
			failConsentDeleteOnce.Store(true)
			var failedInTxErr error
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				failedInTxErr = resolver.revokeOnClientIDMismatch(txCtx, id, fosite.ErrCIMDClientIDMismatch)
				_, storeErr := store.StoreCIMDClient(txCtx, resolvedInTx, nil, future.ToTime())
				if storeErr != nil {
					return storeErr
				}
				return dbFromContext(txCtx, db).Create(&InteractionSession{
					Base:        model.Base{ID: disarmedInteractionID},
					ClientID:    id,
					Scopes:      datatype.StringList{"openid"},
					RequestedAt: datatype.DateTime(time.Now()),
					Parameters:  InteractionSessionParameters{},
				}).Error
			})
			require.NoError(t, err)
			require.ErrorIs(t, failedInTxErr, fosite.ErrCIMDClientIDMismatch)
			// Replaying the failed in-tx mismatch error after the same transaction committed a valid repair must be a no-op
			_ = resolver.revokeOnClientIDMismatch(t.Context(), id, failedInTxErr)
			storedAfterSuperseded, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.NotNil(t, storedAfterSuperseded.MetadataExpiresAt, "valid repair later in the same committed tx must disarm the earlier failed mismatch commit retry and suppress cached error replays")
			var disarmedCount int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", disarmedInteractionID).Count(&disarmedCount).Error)
			assert.Equal(t, int64(1), disarmedCount, "post-repair interaction must survive commit and cached error replay when an earlier failed mismatch in the same tx was superseded by repair")

			// If an enclosing transaction rolls back and another transaction commits a valid repair (plus new interaction) before the deferred rollback retry acquires the client lock, the deferred retry must skip without undoing the repair
			const externalRepairInteractionID = "77777777-7777-7777-7777-777777777777"
			rollbackSentinel := errors.New("rollback after mismatch")
			var externalRepairErr error
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				_ = resolver.revokeOnClientIDMismatch(txCtx, id, fosite.ErrCIMDClientIDMismatch)
				// afterTxFinish runs right after DB rollback releases locks and right before postTx runs the deferred mismatch retry
				afterTxFinish(txCtx, func(committed bool) {
					if committed {
						return
					}
					repairCtx := contextWithoutTx(txCtx)
					if _, repairErr := store.StoreCIMDClient(repairCtx, resolvedInTx, nil, future.ToTime()); repairErr != nil {
						externalRepairErr = repairErr
						return
					}
					externalRepairErr = db.WithContext(repairCtx).Create(&InteractionSession{
						Base:        model.Base{ID: externalRepairInteractionID},
						ClientID:    id,
						Scopes:      datatype.StringList{"openid"},
						RequestedAt: datatype.DateTime(time.Now()),
						Parameters:  InteractionSessionParameters{},
					}).Error
				})
				return rollbackSentinel
			})
			require.ErrorIs(t, err, rollbackSentinel)
			require.NoError(t, externalRepairErr)
			storedAfterExternalRepair, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.NotNil(t, storedAfterExternalRepair.MetadataExpiresAt, "deferred rollback retry must not invalidate a valid repair committed by another transaction")
			var externalRepairCount int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", externalRepairInteractionID).Count(&externalRepairCount).Error)
			assert.Equal(t, int64(1), externalRepairCount, "deferred rollback retry must not revoke interactions created after another transaction's repair")

			// Concurrent replay of a cached mismatch error: a second caller that passes the unlocked fast path before the first caller completes and a repair commits must recheck under the client lock and skip without clobbering the repair
			failConsentDeleteOnce.Store(true)
			cachedErr := resolver.revokeOnClientIDMismatch(t.Context(), id, fosite.ErrCIMDClientIDMismatch)
			require.ErrorIs(t, cachedErr, fosite.ErrCIMDClientIDMismatch)
			enteredBeforeTx := make(chan struct{})
			proceedBeforeTx := make(chan struct{})
			resolver.onBeforeMismatchTx = func() {
				close(enteredBeforeTx)
				<-proceedBeforeTx
			}
			concurrentReplayDone := make(chan error, 1)
			go func() {
				concurrentReplayDone <- resolver.revokeOnClientIDMismatch(t.Context(), id, cachedErr)
			}()
			<-enteredBeforeTx
			resolver.onBeforeMismatchTx = nil
			// First caller finishes cleanup on the same cached error, then a valid repair commits with a new interaction
			_ = resolver.revokeOnClientIDMismatch(t.Context(), id, cachedErr)
			const concurrentRepairInteractionID = "88888888-8888-8888-8888-888888888888"
			_, err = store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
			require.NoError(t, err)
			require.NoError(t, db.Create(&InteractionSession{
				Base:        model.Base{ID: concurrentRepairInteractionID},
				ClientID:    id,
				Scopes:      datatype.StringList{"openid"},
				RequestedAt: datatype.DateTime(time.Now()),
				Parameters:  InteractionSessionParameters{},
			}).Error)
			close(proceedBeforeTx)
			<-concurrentReplayDone
			storedAfterCachedReplay, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.NotNil(t, storedAfterCachedReplay.MetadataExpiresAt, "concurrent cached mismatch replay must recheck completion under the client lock and preserve intervening repairs")
			var concurrentRepairCount int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", concurrentRepairInteractionID).Count(&concurrentRepairCount).Error)
			assert.Equal(t, int64(1), concurrentRepairCount, "concurrent cached mismatch replay must not revoke interactions created after an intervening repair")

			// An older deferred retry sharing state with a newer retry must skip only itself without setting state.done and cancelling the newer retry
			sharedState := &mismatchCleanupState{repairGen: store.cimdRepairGen(id)}
			sharedErr := revokedMismatchError{err: fosite.ErrCIMDClientIDMismatch, state: sharedState}
			olderGen := store.markCIMDInvalidationFailed(id)
			newerGen := store.markCIMDInvalidationFailed(id)
			_ = resolver.revokeOnClientIDMismatchExpectedGen(t.Context(), id, sharedErr, olderGen)
			assert.False(t, sharedState.done.Load(), "obsolete deferred retry must not mark shared cleanup state done when superseded by a newer generation")
			_ = resolver.revokeOnClientIDMismatchExpectedGen(t.Context(), id, sharedErr, newerGen)
			assert.True(t, sharedState.done.Load(), "current deferred retry must still execute and mark shared cleanup state done")
			storedAfterNewerRetry, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, storedAfterNewerRetry.MetadataExpiresAt, "current deferred retry must invalidate persisted expiry after obsolete retry skipped")

			// A repair followed by a LATER failed mismatch in the same committed transaction must not advance the repair epoch and suppress the later mismatch's retry
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				if _, storeErr := store.StoreCIMDClient(txCtx, resolvedInTx, nil, future.ToTime()); storeErr != nil {
					return storeErr
				}
				failConsentDeleteOnce.Store(true)
				_ = resolver.revokeOnClientIDMismatch(txCtx, id, fosite.ErrCIMDClientIDMismatch)
				return nil
			})
			require.NoError(t, err)
			storedAfterRepairThenMismatch, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, storedAfterRepairThenMismatch.MetadataExpiresAt, "a repair earlier in the same committed tx must not suppress a later failed mismatch's deferred retry")

			// A fresh mismatch suppressed by an intervening repair must preserve its completed state on the returned error so ResolveClient/RefreshMetadataClient passing it through cleanup again does not capture the new epoch and undo the repair
			enteredSuppressed := make(chan struct{})
			proceedSuppressed := make(chan struct{})
			resolver.onBeforeMismatchTx = func() {
				close(enteredSuppressed)
				<-proceedSuppressed
			}
			suppressedDone := make(chan error, 1)
			go func() {
				suppressedDone <- resolver.revokeOnClientIDMismatch(t.Context(), id, fosite.ErrCIMDClientIDMismatch)
			}()
			<-enteredSuppressed
			resolver.onBeforeMismatchTx = nil
			const postSuppressedInteractionID = "99999999-9999-9999-9999-999999999999"
			_, err = store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
			require.NoError(t, err)
			require.NoError(t, db.Create(&InteractionSession{
				Base:        model.Base{ID: postSuppressedInteractionID},
				ClientID:    id,
				Scopes:      datatype.StringList{"openid"},
				RequestedAt: datatype.DateTime(time.Now()),
				Parameters:  InteractionSessionParameters{},
			}).Error)
			close(proceedSuppressed)
			suppressedErr := <-suppressedDone
			// Simulate ResolveClient/RefreshMetadataClient passing the fetcher's returned error through revokeOnClientIDMismatch a second time
			_ = resolver.revokeOnClientIDMismatch(t.Context(), id, suppressedErr)
			storedAfterSuppressedReplay, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.NotNil(t, storedAfterSuppressedReplay.MetadataExpiresAt, "second cleanup call on a repair-suppressed fresh mismatch error must not invalidate the repaired row")
			var postSuppressedCount int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", postSuppressedInteractionID).Count(&postSuppressedCount).Error)
			assert.Equal(t, int64(1), postSuppressedCount, "second cleanup call on a repair-suppressed fresh mismatch error must not revoke post-repair interactions")

			// Atomic pending-clear + failure-marker coordination: wait timeout while a repair is pending must defer publication to finishCIMDClear (discarded on commit, published on rollback)
			canceledCtx, cancelWait := context.WithCancel(t.Context())
			cancelWait()
			epochBeforePendingCommit := store.cimdRepairGen(id)
			pendingRepairCh, _, err := store.acquireCIMDPendingClear(t.Context(), id)
			require.NoError(t, err)
			_, repairedOnTimeout := store.markCIMDInvalidationFailedAfterPending(canceledCtx, id, epochBeforePendingCommit)
			assert.False(t, repairedOnTimeout)
			assert.False(t, store.isCIMDInvalidated(id), "timed-out publication must not mark the client invalid while a repair is still pending")
			store.finishCIMDClear(id, 0, pendingRepairCh, true, true)
			assert.False(t, store.isCIMDInvalidated(id), "committing the pending repair must discard the deferred failure publication")

			epochBeforePendingRollback := store.cimdRepairGen(id)
			pendingRollbackCh, _, err := store.acquireCIMDPendingClear(t.Context(), id)
			require.NoError(t, err)
			_, repairedOnRollbackTimeout := store.markCIMDInvalidationFailedAfterPending(canceledCtx, id, epochBeforePendingRollback)
			assert.False(t, repairedOnRollbackTimeout)
			assert.False(t, store.isCIMDInvalidated(id), "timed-out publication must wait for the pending repair outcome")
			store.finishCIMDClear(id, 0, pendingRollbackCh, false, true)
			assert.True(t, store.isCIMDInvalidated(id), "rolling back the pending repair must publish the deferred failure marker")

			// Verify StoreCIMDClient registers cimdPendingClear BEFORE executing DB writes so a standalone double-failure arriving mid-repair sees the pending entry, leaves the existing failure generation unchanged, and records its epoch in deferredFailEpochs
			var checkPendingDuringRepair atomic.Bool
			var sawDeferredDuringRepair atomic.Bool
			var epochBeforeMidRepair uint64
			var genBeforeMidRepair uint64
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:check_pending_during_repair", func(tx *gorm.DB) {
				if checkPendingDuringRepair.CompareAndSwap(true, false) {
					_, repairedNow := store.markCIMDInvalidationFailedAfterPending(canceledCtx, id, epochBeforeMidRepair)
					store.cimdInvalidationMu.Lock()
					currentGen := store.cimdFailedInvalidation[id]
					entry := store.cimdPendingClear[id]
					var hasEpoch bool
					for _, e := range entry.deferredFailEpochs {
						if e == epochBeforeMidRepair {
							hasEpoch = true
							break
						}
					}
					store.cimdInvalidationMu.Unlock()
					if !repairedNow && currentGen == genBeforeMidRepair && hasEpoch {
						sawDeferredDuringRepair.Store(true)
					}
				}
			}))
			t.Cleanup(func() {
				_ = db.Callback().Update().Remove("test:check_pending_during_repair")
			})
			// Seed a stale deleted-client repair entry to verify retention pruning on repair commit
			const deletedClientID = "https://8.8.8.8/oauth/deleted-repaired-client"
			store.cimdInvalidationMu.Lock()
			if store.cimdRepairSeq == nil {
				store.cimdRepairSeq = make(map[string]cimdRepairEntry)
			}
			store.cimdRepairSeq[deletedClientID] = cimdRepairEntry{seq: 1, updatedAt: time.Now().Add(-2 * cimdRepairSeqRetention)}
			store.cimdInvalidationMu.Unlock()
			epochBeforeMidRepair = store.cimdRepairGen(id)
			genBeforeMidRepair = store.cimdInvalidationFailedGen(id)
			require.NotZero(t, genBeforeMidRepair)
			checkPendingDuringRepair.Store(true)
			_, err = store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
			require.NoError(t, err)
			assert.True(t, sawDeferredDuringRepair.Load(), "StoreCIMDClient must register cimdPendingClear before executing DB writes and defer concurrent failures without overwriting the existing generation")
			assert.False(t, store.isCIMDInvalidated(id), "committing StoreCIMDClient must discard deferred failures recorded during its DB writes")
			store.cimdInvalidationMu.Lock()
			_, deletedEntryStillPresent := store.cimdRepairSeq[deletedClientID]
			store.cimdInvalidationMu.Unlock()
			assert.False(t, deletedEntryStillPresent, "stale cimdRepairSeq entries past retention must be pruned on subsequent repairs")

			// Pruning an already-hour-old repair entry after a mismatch captures its epoch must NOT suppress the genuine mismatch
			observedBeforePrune := store.cimdRepairGen(id)
			require.Greater(t, observedBeforePrune, uint64(1))
			store.cimdInvalidationMu.Lock()
			prunableEntry := store.cimdRepairSeq[id]
			prunableEntry.updatedAt = time.Now().Add(-2 * cimdRepairSeqRetention)
			store.cimdRepairSeq[id] = prunableEntry
			store.cimdInvalidationMu.Unlock()
			const unrelatedRepairedClientID = "https://8.8.8.8/oauth/unrelated-repaired-client"
			unrelatedCh, _, err := store.acquireCIMDPendingClear(t.Context(), unrelatedRepairedClientID)
			require.NoError(t, err)
			store.finishCIMDClear(unrelatedRepairedClientID, 0, unrelatedCh, true, true)
			store.cimdInvalidationMu.Lock()
			_, prunedActiveClient := store.cimdRepairSeq[id]
			store.cimdInvalidationMu.Unlock()
			assert.False(t, prunedActiveClient, "hour-old entry should be pruned by an unrelated repair")
			prePruneMismatchErr := revokedMismatchError{
				err:   fosite.ErrCIMDClientIDMismatch,
				state: &mismatchCleanupState{repairGen: observedBeforePrune},
			}
			_ = resolver.revokeOnClientIDMismatch(t.Context(), id, prePruneMismatchErr)
			storedAfterPrunedMismatch, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.Nil(t, storedAfterPrunedMismatch.MetadataExpiresAt, "pruning an hour-old repair entry must not suppress a genuine mismatch that captured that epoch")
			_, err = store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
			require.NoError(t, err)

			// A caller-owned transaction whose mismatch cleanup fails while waiting on another transaction's pending repair must not publish a new failure marker or undo the committing repair
			const callerTxRepairInteractionID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
			require.NoError(t, db.Create(&InteractionSession{
				Base:        model.Base{ID: callerTxRepairInteractionID},
				ClientID:    id,
				Scopes:      datatype.StringList{"openid"},
				RequestedAt: datatype.DateTime(time.Now()),
				Parameters:  InteractionSessionParameters{},
			}).Error)
			pendingOtherRepairCh, _, err := store.acquireCIMDPendingClear(t.Context(), id)
			require.NoError(t, err)
			callerWaitRollback := errors.New("rollback after wait failure")
			err = withTx(t.Context(), db, func(txCtx context.Context) error {
				canceledTxCtx, cancelTxWait := context.WithCancel(txCtx)
				cancelTxWait()
				_ = resolver.revokeOnClientIDMismatch(canceledTxCtx, id, fosite.ErrCIMDClientIDMismatch)
				assert.False(t, store.isCIMDInvalidated(id), "failed caller-tx mismatch must defer publication while another transaction's repair is pending")
				store.finishCIMDClear(id, 0, pendingOtherRepairCh, true, true)
				return callerWaitRollback
			})
			require.ErrorIs(t, err, callerWaitRollback)
			assert.False(t, store.isCIMDInvalidated(id), "committing the other transaction's repair must supersede the failed caller-tx mismatch")
			storedAfterCallerTxWaitFail, err := freshStoreAfterCommit.firstClientByID(t.Context(), id)
			require.NoError(t, err)
			assert.NotNil(t, storedAfterCallerTxWaitFail.MetadataExpiresAt, "queued retry from failed caller-tx wait must not invalidate a committed repair")
			var callerTxRepairCount int64
			require.NoError(t, db.Model(&InteractionSession{}).Where("id = ?", callerTxRepairInteractionID).Count(&callerTxRepairCount).Error)
			assert.Equal(t, int64(1), callerTxRepairCount, "queued retry from failed caller-tx wait must not revoke post-repair interactions")

			// Re-invalidate via standalone rollback of deferred publication before testing standalone double DB failure
			epochBeforeDoubleFailSetup := store.cimdRepairGen(id)
			setupRollbackCh, _, err := store.acquireCIMDPendingClear(t.Context(), id)
			require.NoError(t, err)
			_, _ = store.markCIMDInvalidationFailedAfterPending(canceledCtx, id, epochBeforeDoubleFailSetup)
			store.finishCIMDClear(id, 0, setupRollbackCh, false, true)
			require.True(t, store.isCIMDInvalidated(id))

			// Standalone double DB failure while a repair commits concurrently must not publish a new failure marker on the repaired client and must return completed state so a second cleanup call stays a no-op
			enteredDoubleFail := make(chan struct{})
			proceedDoubleFail := make(chan struct{})
			resolver.onBeforeMismatchTx = func() {
				close(enteredDoubleFail)
				<-proceedDoubleFail
			}
			doubleFailDone := make(chan error, 1)
			go func() {
				doubleFailDone <- resolver.revokeOnClientIDMismatch(t.Context(), id, fosite.ErrCIMDClientIDMismatch)
			}()
			<-enteredDoubleFail
			resolver.onBeforeMismatchTx = nil
			_, err = store.StoreCIMDClient(t.Context(), resolvedInTx, nil, future.ToTime())
			require.NoError(t, err)
			require.False(t, store.isCIMDInvalidated(id))
			rawDB, err := db.DB()
			require.NoError(t, err)
			require.NoError(t, rawDB.Close())
			close(proceedDoubleFail)
			doubleFailErr := <-doubleFailDone
			var doubleFailTracked revokedMismatchError
			require.ErrorAs(t, doubleFailErr, &doubleFailTracked)
			require.NotNil(t, doubleFailTracked.state)
			assert.True(t, doubleFailTracked.state.done.Load(), "repair-suppressed standalone double failure must preserve completed state on its returned error")
			_ = resolver.revokeOnClientIDMismatch(t.Context(), id, doubleFailErr)
			assert.False(t, store.isCIMDInvalidated(id), "standalone double DB failure observed before a committed repair must not re-invalidate the repaired client")

			// A deferred retry (expectedGen != 0) whose primary and fallback DB attempts both fail before checking expectedGen must preserve the current store generation on both SQLite and PostgreSQL handles
			_ = resolver.revokeOnClientIDMismatchExpectedGen(t.Context(), id, fosite.ErrCIMDClientIDMismatch, 99)
			assert.False(t, store.isCIMDInvalidated(id), "double DB failure on a deferred retry for an older mismatch generation must preserve the current cleared store generation")
		})
	})

	t.Run("repairing incompatible cached auth settings with non-NULL expiry revokes surviving grants", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			seed func(*model.OidcClient)
		}{
			{
				name: "pkce disabled",
				seed: func(c *model.OidcClient) { c.PkceEnabled = false },
			},
			{
				name: "lingering federated identity",
				seed: func(c *model.OidcClient) {
					c.Credentials = model.OidcClientCredentials{
						FederatedIdentities: []model.OidcClientFederatedIdentity{{Issuer: "https://issuer.example.com", Subject: "sub"}},
					}
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newMetadataStore(t, nil)
				future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
				seed := model.OidcClient{
					Base:               model.Base{ID: id},
					Name:               "App",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
					MetadataExpiresAt:  &future,
				}
				tc.seed(&seed)
				require.NoError(t, s.db.Create(&seed).Error)
				require.NoError(t, s.db.Exec(
					"INSERT INTO user_authorized_oidc_clients (client_id, user_id, scope, last_used_at) VALUES (?, ?, ?, ?)",
					id, "user-incompat", "openid", time.Now()).Error)
				require.NoError(t, s.CreateRefreshTokenSession(t.Context(), "refresh-incompat", "", newTestRequester("req-incompat", id, "user-incompat", "")))
				require.NoError(t, s.db.Create(&InteractionSession{
					Base:        model.Base{ID: "interaction-incompat"},
					ClientID:    id,
					RequestedAt: datatype.DateTime(time.Now()),
				}).Error)

				_, err := s.StoreCIMDClient(t.Context(), Client{OidcClient: model.OidcClient{
					Base:               model.Base{ID: id},
					Name:               "App",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
				}}, nil, future.ToTime())
				require.NoError(t, err)

				var consent int64
				require.NoError(t, s.db.Table("user_authorized_oidc_clients").Where("client_id = ?", id).Count(&consent).Error)
				assert.Zero(t, consent, "repairing incompatible cached auth settings must revoke consent")
				_, err = s.GetRefreshTokenSession(t.Context(), "refresh-incompat", nil)
				require.ErrorIs(t, err, fosite.ErrNotFound, "repairing incompatible cached auth settings must revoke tokens")
				var interactions int64
				require.NoError(t, s.db.Model(&InteractionSession{}).Where("client_id = ?", id).Count(&interactions).Error)
				assert.Zero(t, interactions, "repairing incompatible cached auth settings must revoke pending interactions")
			})
		}
	})

	t.Run("loadConsistentCIMDClient returns a single consistent snapshot on SQLite and PostgreSQL", func(t *testing.T) {
		testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
			if db.Name() == "sqlite" {
				db = testutils.NewConcurrentDatabaseForTest(t)
			}
			for _, useTx := range []bool{false, true} {
				mode := "standalone"
				if useTx {
					mode = "in-snapshot-tx"
				}
				t.Run(mode, func(t *testing.T) {
					clientID := id + "/" + mode
					future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
					require.NoError(t, db.Create(&model.OidcClient{
						Base:               model.Base{ID: clientID},
						Name:               "first",
						IsPublic:           true,
						PkceEnabled:        true,
						ClientType:         model.OidcClientTypeCIMD,
						CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
						MetadataGrantTypes: datatype.StringList{"authorization_code"},
						MetadataExpiresAt:  &future,
					}).Error)

					var updatedOnce atomic.Bool
					var observedScopeName string
					apiAccess := &txCapturingAPIAccess{
						onClientAPIScopes: func(ctx context.Context, tx *gorm.DB, cid string) ([]string, []string, error) {
							if updatedOnce.CompareAndSwap(false, true) {
								// Update the client row from an independent connection between firstClientByID and ClientAPIScopes
								done := make(chan error, 1)
								go func() {
									done <- db.Model(&model.OidcClient{}).Where("id = ?", cid).Update("name", "second").Error
								}()
								if err := <-done; err != nil {
									return nil, nil, err
								}
							}
							var row model.OidcClient
							if err := tx.WithContext(ctx).Select("name").Where("id = ?", cid).First(&row).Error; err != nil {
								return nil, nil, err
							}
							observedScopeName = row.Name
							return []string{"scope:" + row.Name}, nil, nil
						},
					}
					store := NewStore(db, apiAccess)

					var (
						loadedModel model.OidcClient
						loaded      Client
						found       bool
						err         error
					)
					if useTx {
						opts := &sql.TxOptions{ReadOnly: true}
						if db.Name() == "postgres" {
							opts.Isolation = sql.LevelRepeatableRead
						}
						err = db.WithContext(t.Context()).Transaction(func(roTx *gorm.DB) error {
							loadedModel, loaded, found, err = store.loadConsistentCIMDClient(contextWithTx(t.Context(), roTx), clientID)
							return err
						}, opts)
					} else {
						loadedModel, loaded, found, err = store.loadConsistentCIMDClient(t.Context(), clientID)
					}
					require.NoError(t, err)
					require.True(t, found)
					assert.Equal(t, loadedModel.Name, observedScopeName, "client metadata and runtime scope lookup must observe the same database snapshot")
					assert.Equal(t, []string{"scope:" + loadedModel.Name}, loaded.apiScopes)
				})
			}

			t.Run("unrelated writes inside caller tx do not fail lookup", func(t *testing.T) {
				targetID := id + "/stable-target"
				unrelatedID := id + "/unrelated-churn"
				future := datatype.DateTime(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
				require.NoError(t, db.Create(&model.OidcClient{
					Base:               model.Base{ID: targetID},
					Name:               "Stable Target",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://app.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
					MetadataExpiresAt:  &future,
				}).Error)
				require.NoError(t, db.Create(&model.OidcClient{
					Base:               model.Base{ID: unrelatedID},
					Name:               "Unrelated 0",
					IsPublic:           true,
					PkceEnabled:        true,
					ClientType:         model.OidcClientTypeCIMD,
					CallbackURLs:       datatype.StringList{"https://other.example.com/cb"},
					MetadataGrantTypes: datatype.StringList{"authorization_code"},
					MetadataExpiresAt:  &future,
				}).Error)

				var churnCount atomic.Int32
				apiAccess := &txCapturingAPIAccess{
					onClientAPIScopes: func(_ context.Context, _ *gorm.DB, _ string) ([]string, []string, error) {
						n := churnCount.Add(1)
						// Commit an unrelated write during the lookup so cluster-wide snapshots advance continuously
						done := make(chan error, 1)
						go func() {
							done <- db.Model(&model.OidcClient{}).Where("id = ?", unrelatedID).Update("name", "Unrelated "+strings.Repeat("x", int(n))).Error
						}()
						if err := <-done; err != nil {
							return nil, nil, err
						}
						return []string{"scope:stable"}, nil, nil
					},
				}
				store := NewStore(db, apiAccess)

				var (
					loadedModel model.OidcClient
					found       bool
					err         error
				)
				if db.Name() == "postgres" {
					err = withTx(t.Context(), db, func(txCtx context.Context) error {
						loadedModel, _, found, err = store.loadConsistentCIMDClient(txCtx, targetID)
						return err
					})
				} else {
					err = db.WithContext(t.Context()).Transaction(func(roTx *gorm.DB) error {
						loadedModel, _, found, err = store.loadConsistentCIMDClient(contextWithTx(t.Context(), roTx), targetID)
						return err
					}, &sql.TxOptions{ReadOnly: true})
				}
				require.NoError(t, err, "unrelated cluster writes must not make a valid CIMD lookup fail inside a caller transaction")
				require.True(t, found)
				assert.Equal(t, "Stable Target", loadedModel.Name)
			})
		})
	})
}

func TestMatchRedirectURI_CIMDStrictMatching(t *testing.T) {
	newCIMDClient := func(callbacks ...string) Client {
		return Client{OidcClient: model.OidcClient{
			Base:         model.Base{ID: "https://app.example.com/oauth/client"},
			ClientType:   model.OidcClientTypeCIMD,
			CallbackURLs: datatype.StringList(callbacks),
		}}
	}

	t.Run("rejects wildcard glob fallback, userinfo, fragments, and non-loopback port changes", func(t *testing.T) {
		client := newCIMDClient("https://app.example.com/callback", "https://app.example.com/dir/cb")
		for _, bad := range []string{
			"https://app.example.com/callback#frag",
			"https://app.example.com/callback#",
			"https://user@app.example.com/callback",
			"https://app.example.com:8443/callback",
			"https://app.example.com/dir/../callback",
			"https://app.example.com/callback%2Fextra",
		} {
			_, err := matchRedirectURI(bad, client)
			require.Error(t, err, "must reject %q", bad)
		}
	})

	t.Run("requires exact string matching for scheme, host, path, and query", func(t *testing.T) {
		client := newCIMDClient("https://app.example.com/Callback?x=%2F", "com.example.app:/Callback")
		for _, valid := range []string{
			"https://app.example.com/Callback?x=%2F",
			"com.example.app:/Callback",
		} {
			matched, err := matchRedirectURI(valid, client)
			require.NoError(t, err, "must accept %q", valid)
			require.NotNil(t, matched)
		}
		for _, bad := range []string{
			"HTTPS://app.example.com/Callback?x=%2F",
			"https://APP.EXAMPLE.COM/Callback?x=%2F",
			"https://app.example.com/Callback?x=/",
			"https://app.example.com/Callback?x=%2F&extra=1",
			"https://app.example.com/Callback",
			"COM.EXAMPLE.APP:/Callback",
			"com.example.app:/callback",
		} {
			_, err := matchRedirectURI(bad, client)
			require.Error(t, err, "must reject non-exact redirect_uri %q", bad)
		}
	})

	t.Run("accepts RFC 8252 ephemeral port changes only for HTTP loopback IPs with identical scheme and host spelling", func(t *testing.T) {
		client := newCIMDClient(
			"http://127.0.0.1/callback",
			"http://127.0.0.1/callback?a=1",
			"http://127.0.0.2/callback",
			"http://[::1]:8080/callback",
			"http://[0:0:0:0:0:0:0:1]:8080/expanded",
			"HTTP://127.0.0.1:8080/upper",
		)
		for _, valid := range []string{
			"http://127.0.0.1:49152/callback",
			"http://127.0.0.1:49152/callback?a=1",
			"http://127.0.0.2:49152/callback",
			"http://[::1]:49153/callback",
			"http://[0:0:0:0:0:0:0:1]:49153/expanded",
			"HTTP://127.0.0.1:49154/upper",
		} {
			matched, err := matchRedirectURI(valid, client)
			require.NoError(t, err, "must accept loopback %q", valid)
			require.NotNil(t, matched)
		}
		for _, bad := range []string{
			"http://127.0.0.1:49152/callback?a=2",
			"http://127.0.0.1:49152/callback?a=1&b=2",
			"http://[::1]:49153/expanded",
			"http://127.0.0.1:49154/upper",
			"http://[::1%25lo]:49153/callback",
			"http://localhost:49152/callback",
			"https://127.0.0.1:49152/callback",
			"http://127.0.0.1:49152/callback#",
		} {
			_, err := matchRedirectURI(bad, client)
			require.Error(t, err, "must reject %q", bad)
		}
	})

	t.Run("omitted redirect_uri requires a single valid callback without userinfo or fragment", func(t *testing.T) {
		validClient := newCIMDClient("https://app.example.com/callback")
		matched, err := matchRedirectURI("", validClient)
		require.NoError(t, err)
		assert.Equal(t, "https://app.example.com/callback", matched.String())

		for _, invalidClient := range []Client{
			newCIMDClient("https://user:pass@app.example.com/callback"),
			newCIMDClient("https://app.example.com/callback#"),
			newCIMDClient("https://app.example.com/cb1", "https://app.example.com/cb2"),
		} {
			_, err := matchRedirectURI("", invalidClient)
			require.Error(t, err)
		}
	})
}

type txCapturingAPIAccess struct {
	seenTx                     *gorm.DB
	onClientAPIScopes          func(ctx context.Context, tx *gorm.DB, clientID string) ([]string, []string, error)
	onAllowedScopesForAudience func(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType SubjectType) ([]string, bool, bool, error)
	onHeldScopes               func(ctx context.Context, tx *gorm.DB, userID, audience string, scopes []string) ([]string, error)
}

func (a *txCapturingAPIAccess) ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string, _ bool) ([]string, []string, error) {
	if a.onClientAPIScopes != nil {
		return a.onClientAPIScopes(ctx, tx, clientID)
	}
	return nil, nil, nil
}
func (a *txCapturingAPIAccess) AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType SubjectType) ([]string, bool, bool, error) {
	a.seenTx = tx
	if a.onAllowedScopesForAudience != nil {
		return a.onAllowedScopesForAudience(ctx, tx, clientID, audience, subjectType)
	}
	return nil, false, false, nil
}
func (a *txCapturingAPIAccess) DescribePermissions(context.Context, string, []string) ([]dto.ScopeInfoDto, error) {
	return nil, nil
}
func (a *txCapturingAPIAccess) HeldScopes(ctx context.Context, tx *gorm.DB, userID, audience string, scopes []string) ([]string, error) {
	if a.onHeldScopes != nil {
		return a.onHeldScopes(ctx, tx, userID, audience, scopes)
	}
	return scopes, nil
}
func (a *txCapturingAPIAccess) RequireAdminAccessForClients(context.Context, *gorm.DB, authz.Principal, ...string) error {
	return nil
}

func TestCIMDPolicy_AllowCIMDClientPassesActiveTransaction(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	apiAccess := &txCapturingAPIAccess{}
	store := NewStore(db, apiAccess)
	resolver := newCIMDClientResolver(store, cimdResolverConfig{
		getURLAllowlist: func() []string { return []string{"*"} },
	})

	var activeConnPool gorm.ConnPool
	err := withTx(t.Context(), db, func(txCtx context.Context) error {
		activeConnPool = dbFromContext(txCtx, db).Statement.ConnPool
		return resolver.policy.AllowCIMDClient(txCtx, "https://app.example.com/oauth/client")
	})
	require.NoError(t, err)
	require.NotNil(t, apiAccess.seenTx)
	assert.Equal(t, activeConnPool, apiAccess.seenTx.Statement.ConnPool, "AllowCIMDClient must pass the active transaction connection pool")
	assert.NotEqual(t, db.Statement.ConnPool, apiAccess.seenTx.Statement.ConnPool, "AllowCIMDClient must not use the root DB pool inside a transaction")
}

func TestAuthorizationHandler_CIMDErrorRedirectRechecksPersistedMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.NewDatabaseForTest(t)
	const id = "https://app.example.com/oauth/client"
	const cb = "https://app.example.com/callback"

	seed := model.OidcClient{
		Base:              model.Base{ID: id},
		Name:              "App",
		IsPublic:          true,
		PkceEnabled:       true,
		ClientType:        model.OidcClientTypeCIMD,
		CallbackURLs:      datatype.StringList{cb},
		MetadataExpiresAt: nil, // invalidated after mismatch
	}
	require.NoError(t, db.Create(&seed).Error)

	const issuer = "https://issuer.example.com"
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: key}, Config{
		BaseURL: issuer, TokenBaseURL: issuer, Secret: []byte("test-secret"),
	}, nil)
	require.NoError(t, err)

	ar := fosite.NewAuthorizeRequest()
	ar.Client = Client{OidcClient: seed}
	ar.State = "secret-state"
	ar.Form = url.Values{"redirect_uri": {cb}, "state": {"secret-state"}}
	parsedCB, err := url.Parse(cb)
	require.NoError(t, err)
	ar.RedirectURI = parsedCB

	handler := &authorizationHandler{
		provider:             provider,
		authorizationService: &authorizationService{db: db},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/authorize", nil)
	handler.writeAuthorizeError(t.Context(), c, ar, fosite.ErrLoginRequired)

	location := rec.Header().Get("Location")
	assert.True(t, strings.HasPrefix(location, "/interaction/error?"), "invalidated CIMD client must not redirect errors/state to callback, got %q", location)
	assert.NotContains(t, location, "secret-state")
	assert.Equal(t, http.StatusFound, c.Writer.Status())

	t.Run("valid uppercase-scheme loopback callback still receives redirectable authorize errors", func(t *testing.T) {
		const upperClientID = "https://upper.example.com/oauth/client"
		const registeredCB = "HTTP://127.0.0.1:8080/upper"
		const requestedCB = "HTTP://127.0.0.1:49154/upper"
		expiresAt := datatype.DateTime(time.Now().Add(time.Hour))

		upperSeed := model.OidcClient{
			Base:              model.Base{ID: upperClientID},
			Name:              "Upper App",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{registeredCB},
			MetadataExpiresAt: &expiresAt,
		}
		require.NoError(t, db.Create(&upperSeed).Error)

		upperAR := fosite.NewAuthorizeRequest()
		upperAR.Client = Client{OidcClient: upperSeed}
		upperAR.State = "secret-state"
		upperAR.Form = url.Values{"redirect_uri": {requestedCB}, "state": {"secret-state"}}
		upperAR.RedirectURIMatcher = matchRedirectURI
		parsedUpperCB, parseErr := url.Parse(requestedCB)
		require.NoError(t, parseErr)
		upperAR.RedirectURI = parsedUpperCB

		upperRec := httptest.NewRecorder()
		upperCtx, _ := gin.CreateTestContext(upperRec)
		upperCtx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/authorize", nil)
		handler.writeAuthorizeError(t.Context(), upperCtx, upperAR, fosite.ErrLoginRequired)
		upperCtx.Writer.WriteHeaderNow()

		assert.Equal(t, http.StatusSeeOther, upperRec.Code)
		upperLocation := upperRec.Header().Get("Location")
		assert.True(t, strings.HasPrefix(upperLocation, "http://127.0.0.1:49154/upper?"), "valid uppercase-scheme loopback callback must receive error redirect, got %q", upperLocation)
		assert.Contains(t, upperLocation, "error=login_required")
		assert.Contains(t, upperLocation, "state=secret-state")
	})

	t.Run("locked authorization path rejects CIMD client when policy removes it before code issuance", func(t *testing.T) {
		const policyClientID = "https://policy.example.com/oauth/client"
		const policyCB = "https://policy.example.com/callback"
		expiresAt := datatype.DateTime(time.Now().Add(time.Hour))
		policySeed := model.OidcClient{
			Base:              model.Base{ID: policyClientID},
			Name:              "Policy App",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{policyCB},
			MetadataExpiresAt: &expiresAt,
		}
		require.NoError(t, db.Create(&policySeed).Error)

		policyAR := fosite.NewAuthorizeRequest()
		policyAR.Client = Client{OidcClient: policySeed}
		policyAR.State = "policy-state"
		policyAR.Form = url.Values{
			"redirect_uri":          {policyCB},
			"code_challenge":        {"test-challenge"},
			"code_challenge_method": {"S256"},
		}
		parsedPolicyCB, parseErr := url.Parse(policyCB)
		require.NoError(t, parseErr)
		policyAR.RedirectURI = parsedPolicyCB

		policyStore := NewStore(db, nil)
		policyResolver := newCIMDClientResolver(policyStore, cimdResolverConfig{
			getURLAllowlist: func() []string { return nil },
		})
		policyProvider, provErr := newProvider(policyStore, nil, testTokenSigner{key: key}, Config{
			BaseURL: issuer, TokenBaseURL: issuer, Secret: []byte("test-secret"),
		}, policyResolver)
		require.NoError(t, provErr)

		authSvc := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "https://issuer.example.com", nil, appconfig.StaticResolver{}), nil, nil, nil)
		policyHandler := newAuthorizationHandler(policyProvider, authSvc)

		err := withTx(t.Context(), db, func(txCtx context.Context) error {
			_, authErr := authSvc.authorize(txCtx, authorizeInput{
				requester:          policyAR,
				validateCIMDClient: policyHandler.cimdPolicyValidator(),
			})
			return authErr
		})
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("locked authorization reloads PAR and group restrictions committed before lock acquisition", func(t *testing.T) {
		const restrictedID = "https://restricted.example.com/oauth/client"
		const restrictedCB = "https://restricted.example.com/callback"
		expiresAt := datatype.DateTime(time.Now().Add(time.Hour))

		staleSnapshot := model.OidcClient{
			Base:              model.Base{ID: restrictedID},
			Name:              "Restricted App",
			IsPublic:          true,
			PkceEnabled:       true,
			ClientType:        model.OidcClientTypeCIMD,
			CallbackURLs:      datatype.StringList{restrictedCB},
			MetadataExpiresAt: &expiresAt,
		}
		require.NoError(t, db.Create(&staleSnapshot).Error)
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "user-not-in-group"}, Username: "outsider"}).Error)
		group := model.UserGroup{Base: model.Base{ID: "group-required"}, FriendlyName: "Required", Name: "required"}
		require.NoError(t, db.Create(&group).Error)

		// Commit PAR requirement after the client snapshot was taken
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", restrictedID).Update("requires_pushed_authorization_requests", true).Error)

		restrictedAR := fosite.NewAuthorizeRequest()
		restrictedAR.Client = Client{OidcClient: staleSnapshot}
		restrictedAR.Form = url.Values{
			"redirect_uri":          {restrictedCB},
			"code_challenge":        {"test-challenge"},
			"code_challenge_method": {"S256"},
		}
		parsedRestrictedCB, parseErr := url.Parse(restrictedCB)
		require.NoError(t, parseErr)
		restrictedAR.RedirectURI = parsedRestrictedCB

		authSvc := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, "https://issuer.example.com", nil, appconfig.StaticResolver{}), nil, nil, nil)

		err := withTx(t.Context(), db, func(txCtx context.Context) error {
			_, authErr := authSvc.authorize(txCtx, authorizeInput{
				userID:    "user-not-in-group",
				requester: restrictedAR,
			})
			return authErr
		})
		require.ErrorIs(t, err, apperror.OidcPARRequired())

		// Clear PAR, enable group restriction, and attach an allowed user group after the snapshot; authorize must reload IsGroupRestricted and AllowedUserGroups
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", restrictedID).Updates(map[string]any{
			"requires_pushed_authorization_requests": false,
			"is_group_restricted":                    true,
		}).Error)
		require.NoError(t, db.Model(&staleSnapshot).Association("AllowedUserGroups").Append(&group))

		err = withTx(t.Context(), db, func(txCtx context.Context) error {
			_, authErr := authSvc.authorize(txCtx, authorizeInput{
				userID:    "user-not-in-group",
				requester: restrictedAR,
			})
			return authErr
		})
		require.ErrorIs(t, err, fosite.ErrAccessDenied)

		// A member of the allowed group must be permitted, proving AllowedUserGroups was loaded under the lock rather than left empty
		member := model.User{Base: model.Base{ID: "user-in-group"}, Username: "insider"}
		require.NoError(t, db.Create(&member).Error)
		require.NoError(t, db.Model(&member).Association("UserGroups").Append(&group))

		err = withTx(t.Context(), db, func(txCtx context.Context) error {
			authRes, authErr := authSvc.authorize(txCtx, authorizeInput{
				userID:    member.ID,
				requester: restrictedAR,
			})
			if authErr != nil {
				return authErr
			}
			assert.True(t, authRes.RequiresInteraction)
			return nil
		})
		require.NoError(t, err)
	})
}

type hookSessions struct {
	onIsActive func()
}

func (s *hookSessions) IsActive(context.Context, string, string) (bool, error) {
	if s.onIsActive != nil {
		s.onIsActive()
	}
	return true, nil
}

func (s *hookSessions) Revoke(context.Context, string, string, auditlogs.Origin) error {
	return nil
}

func TestTokenHandler_CIMDRefreshRechecksPolicyUnderClientLock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		const (
			baseURL  = "https://issuer.example.com"
			secret   = "test-secret"
			clientID = "https://8.8.8.8/oauth/client"
			userID   = "11111111-1111-1111-1111-111111111111"
		)

		require.NoError(t, claimmappingpolicy.Bootstrap(t.Context(), db))
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "tim"}).Error)

		expiresAt := datatype.DateTime(time.Now().Add(time.Hour))
		clientModel := model.OidcClient{
			Base:               model.Base{ID: clientID},
			Name:               "CIMD App",
			IsPublic:           true,
			PkceEnabled:        true,
			ClientType:         model.OidcClientTypeCIMD,
			CallbackURLs:       datatype.StringList{"https://app.example.com/callback"},
			MetadataGrantTypes: datatype.StringList{"authorization_code", "refresh_token"},
			MetadataExpiresAt:  &expiresAt,
		}
		require.NoError(t, db.Create(&clientModel).Error)

		var allowClient atomic.Bool
		allowClient.Store(true)
		apiAccess := &txCapturingAPIAccess{}
		store := NewStore(db, apiAccess)
		resolver := newCIMDClientResolver(store, cimdResolverConfig{
			getURLAllowlist: func() []string {
				if allowClient.Load() {
					return []string{"*"}
				}
				return nil
			},
		})
		store.clientResolver = resolver

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		provider, err := newProvider(store, nil, testTokenSigner{key: key}, Config{
			BaseURL:      baseURL,
			TokenBaseURL: baseURL,
			Secret:       []byte(secret),
		}, resolver)
		require.NoError(t, err)

		globalSecret, err := DeriveGlobalSecret([]byte(secret))
		require.NoError(t, err)
		strategy := compose.NewOAuth2HMACStrategy(&fosite.Config{
			GlobalSecret:         globalSecret,
			RefreshTokenLifespan: 30 * 24 * time.Hour,
		})
		refreshToken, signature, err := strategy.GenerateRefreshToken(t.Context(), nil)
		require.NoError(t, err)

		now := time.Now().UTC()
		session := NewEmptySession()
		session.Subject = userID
		session.SessionID = "session-1"
		session.Claims = &fositejwt.IDTokenClaims{
			Subject:     userID,
			RequestedAt: now,
			AuthTime:    now,
			Extra:       map[string]any{},
		}
		session.SetExpiresAt(fosite.RefreshToken, now.Add(30*24*time.Hour))
		session.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

		request := fosite.NewRequest()
		request.ID = "cimd-refresh-req-1"
		request.RequestedAt = now
		request.Client = Client{OidcClient: clientModel}
		request.RequestedScope = fosite.Arguments{"openid"}
		request.GrantedScope = fosite.Arguments{"openid"}
		request.RequestedAudience = fosite.Arguments{clientID}
		request.GrantedAudience = fosite.Arguments{clientID}
		request.Session = session
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), signature, "", request))

		t.Run("rejects in-flight refresh when allowlist removes client before locked response", func(t *testing.T) {
			handler := newTokenHandler(
				provider,
				newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{}),
				nil,
				&hookSessions{onIsActive: func() { allowClient.Store(false) }},
			)

			form := url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {refreshToken},
				"client_id":     {clientID},
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			handler.token(c)

			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "invalid_client", body["error"], "refresh must fail once allowlist removes the CIMD client")
			assert.Empty(t, body["access_token"])
			assert.Empty(t, body["refresh_token"])

			var storedSession OAuth2Session
			require.NoError(t, db.Where("kind = ? AND key = ?", sessionKindRefreshToken, signature).First(&storedSession).Error)
			assert.True(t, storedSession.Active, "rejected refresh must not rotate the stored refresh token")
			assert.Nil(t, storedSession.RotatedAt)
		})

		t.Run("rejects in-flight refresh when group restriction is committed before locked response", func(t *testing.T) {
			allowClient.Store(true)
			restrictedGroup := model.UserGroup{Base: model.Base{ID: "22222222-2222-2222-2222-222222222222"}, FriendlyName: "Refresh Only", Name: "refresh-only"}
			require.NoError(t, db.Create(&restrictedGroup).Error)
			t.Cleanup(func() {
				_ = db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("is_group_restricted", false).Error
				_ = db.Model(&clientModel).Association("AllowedUserGroups").Clear()
			})

			handler := newTokenHandler(
				provider,
				newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{}),
				nil,
				&hookSessions{onIsActive: func() {
					require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("is_group_restricted", true).Error)
					require.NoError(t, db.Model(&clientModel).Association("AllowedUserGroups").Append(&restrictedGroup))
				}},
			)

			form := url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {refreshToken},
				"client_id":     {clientID},
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			handler.token(c)

			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "access_denied", body["error"], "refresh must fail when group restriction is committed before client lock")
			assert.Empty(t, body["access_token"])
			assert.Empty(t, body["refresh_token"])

			var storedSession OAuth2Session
			require.NoError(t, db.Where("kind = ? AND key = ?", sessionKindRefreshToken, signature).First(&storedSession).Error)
			assert.True(t, storedSession.Active, "rejected refresh must not rotate the stored refresh token")
			assert.Nil(t, storedSession.RotatedAt)
		})

		t.Run("re-applies fresh user claims loaded under the client lock", func(t *testing.T) {
			allowClient.Store(true)
			freshToken, freshSig, genErr := strategy.GenerateRefreshToken(t.Context(), nil)
			require.NoError(t, genErr)

			freshSession := NewEmptySession()
			freshSession.Subject = userID
			freshSession.SessionID = "session-claims"
			freshSession.Claims = &fositejwt.IDTokenClaims{
				Subject:     userID,
				RequestedAt: now,
				AuthTime:    now,
				Extra:       map[string]any{},
			}
			freshSession.SetExpiresAt(fosite.RefreshToken, now.Add(30*24*time.Hour))
			freshSession.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

			freshReq := fosite.NewRequest()
			freshReq.ID = "cimd-refresh-req-claims"
			freshReq.RequestedAt = now
			freshReq.Client = Client{OidcClient: clientModel}
			freshReq.RequestedScope = fosite.Arguments{"openid", "profile"}
			freshReq.GrantedScope = fosite.Arguments{"openid", "profile"}
			freshReq.RequestedAudience = fosite.Arguments{clientID}
			freshReq.GrantedAudience = fosite.Arguments{clientID}
			freshReq.Session = freshSession
			require.NoError(t, store.CreateRefreshTokenSession(t.Context(), freshSig, "", freshReq))
			t.Cleanup(func() {
				_ = db.Model(&model.User{}).Where("id = ?", userID).Update("username", "tim").Error
				_ = db.Model(&model.OidcClient{}).Where("id = ?", clientID).Updates(map[string]any{
					"access_token_duration_minutes":  model.DefaultAccessTokenDurationMinutes,
					"refresh_token_duration_minutes": model.DefaultRefreshTokenDurationMinutes,
				}).Error
			})

			handler := newTokenHandler(
				provider,
				newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{}),
				nil,
				&hookSessions{onIsActive: func() {
					require.NoError(t, db.Model(&model.User{}).Where("id = ?", userID).Update("username", "tim-locked").Error)
					require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", clientID).Updates(map[string]any{
						"access_token_duration_minutes":  int64(5),
						"refresh_token_duration_minutes": int64(10),
					}).Error)
				}},
			)

			form := url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {freshToken},
				"client_id":     {clientID},
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			handler.token(c)

			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.NotEmpty(t, body["id_token"], "expected refreshed id_token, got error: %v", body["error"])
			idClaims := decodeJWTPart(t, body["id_token"].(string), 1)
			assert.Equal(t, "tim-locked", idClaims["preferred_username"], "id_token must carry claims refreshed under the client lock")
			require.NotEmpty(t, body["access_token"])
			atClaims := decodeJWTPart(t, body["access_token"].(string), 1)
			atExp := time.Unix(int64(atClaims["exp"].(float64)), 0)
			assert.LessOrEqual(t, time.Until(atExp), 6*time.Minute, "access token lifetime must reflect the shortened locked client duration")

			var newRefreshSession OAuth2Session
			require.NoError(t, db.Where("kind = ? AND request_id = ? AND active = ?", sessionKindRefreshToken, freshReq.ID, true).First(&newRefreshSession).Error)
			assert.LessOrEqual(t, time.Until(newRefreshSession.ExpiresAt.ToTime()), 11*time.Minute, "refresh token lifetime must reflect the shortened locked client duration")
		})

		t.Run("revalidates API grant and held scopes under the client lock on refresh", func(t *testing.T) {
			allowClient.Store(true)
			const apiAudience = "https://api.example.com"
			var apiChecks atomic.Int32

			apiAccess.onClientAPIScopes = func(context.Context, *gorm.DB, string) ([]string, []string, error) {
				return []string{"read:data"}, []string{apiAudience}, nil
			}
			// Allow the pre-lock check (call 1) and withdraw the API grant on the locked revalidation inside withTx (call 2)
			apiAccess.onAllowedScopesForAudience = func(_ context.Context, tx *gorm.DB, _, audience string, _ SubjectType) ([]string, bool, bool, error) {
				if audience == apiAudience {
					if apiChecks.Add(1) == 1 {
						return []string{"read:data"}, true, true, nil
					}
					assert.NotNil(t, tx, "locked API revalidation must use the active transaction")
					return nil, true, false, nil
				}
				return nil, false, false, nil
			}
			apiAccess.onHeldScopes = func(_ context.Context, _ *gorm.DB, _, _ string, scopes []string) ([]string, error) {
				return scopes, nil
			}
			t.Cleanup(func() {
				apiAccess.onClientAPIScopes = nil
				apiAccess.onAllowedScopesForAudience = nil
				apiAccess.onHeldScopes = nil
			})

			apiRefreshToken, apiSig, genErr := strategy.GenerateRefreshToken(t.Context(), nil)
			require.NoError(t, genErr)
			apiSession := NewEmptySession()
			apiSession.Subject = userID
			apiSession.SessionID = "session-api"
			apiSession.Claims = &fositejwt.IDTokenClaims{
				Subject:     userID,
				RequestedAt: now,
				AuthTime:    now,
				Extra:       map[string]any{},
			}
			apiSession.SetExpiresAt(fosite.RefreshToken, now.Add(30*24*time.Hour))
			apiSession.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

			apiReq := fosite.NewRequest()
			apiReq.ID = "cimd-refresh-req-api"
			apiReq.RequestedAt = now
			apiReq.Client = Client{OidcClient: clientModel}
			apiReq.RequestedScope = fosite.Arguments{"openid", "read:data"}
			apiReq.GrantedScope = fosite.Arguments{"openid", "read:data"}
			apiReq.RequestedAudience = fosite.Arguments{apiAudience}
			apiReq.GrantedAudience = fosite.Arguments{apiAudience}
			apiReq.Session = apiSession
			require.NoError(t, store.CreateRefreshTokenSession(t.Context(), apiSig, "", apiReq))

			handler := newTokenHandler(
				provider,
				newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{}),
				apiAccess,
				&hookSessions{},
			)

			form := url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {apiRefreshToken},
				"client_id":     {clientID},
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			handler.token(c)

			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "access_denied", body["error"], "refresh must fail when API grant is withdrawn before client lock")
			assert.Empty(t, body["access_token"])
			assert.Empty(t, body["refresh_token"])
			assert.GreaterOrEqual(t, apiChecks.Load(), int32(2))

			var storedSession OAuth2Session
			require.NoError(t, db.Where("kind = ? AND key = ?", sessionKindRefreshToken, apiSig).First(&storedSession).Error)
			assert.True(t, storedSession.Active, "rejected refresh must not rotate the stored refresh token")
		})

		t.Run("device acceptance revalidates API grant under the client lock", func(t *testing.T) {
			allowClient.Store(true)
			const apiAudience = "https://api.example.com"
			var resolveCalls atomic.Int32
			deviceAPIAccess := &txCapturingAPIAccess{
				onClientAPIScopes: func(context.Context, *gorm.DB, string) ([]string, []string, error) {
					return []string{"read:data"}, []string{apiAudience}, nil
				},
				onAllowedScopesForAudience: func(_ context.Context, tx *gorm.DB, _, audience string, _ SubjectType) ([]string, bool, bool, error) {
					if audience == apiAudience {
						// First call is pre-lock resolveGrant (allowed); second call is locked resolveGrant inside withTx (withdrawn)
						if resolveCalls.Add(1) == 1 {
							return []string{"read:data"}, true, true, nil
						}
						assert.NotNil(t, tx, "locked device resolveGrant must use the active transaction")
						return nil, true, false, nil
					}
					return nil, false, false, nil
				},
				onHeldScopes: func(_ context.Context, _ *gorm.DB, _, _ string, scopes []string) ([]string, error) {
					return scopes, nil
				},
			}

			claimsSvc := newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{})
			authSvc := newAuthorizationService(db, newInteractionSessionService(db), claimsSvc, nil, &fakeAuditLogger{}, deviceAPIAccess)
			authSvc.validateCIMDClient = resolver.policy.AllowCIMDClient
			authSvc.isCIMDInvalidated = store.isCIMDInvalidatedForContext
			devSvc := newDeviceService(provider, store, provider.deviceStrategy, authSvc, claimsSvc, &fakeAuditLogger{}, db)

			userCode, userCodeSig, genErr := provider.deviceStrategy.GenerateUserCode(t.Context())
			require.NoError(t, genErr)
			_, deviceCodeSig, genErr := provider.deviceStrategy.GenerateDeviceCode(t.Context())
			require.NoError(t, genErr)
			devReq := fosite.NewDeviceRequest()
			devReq.ID = "device-req-api-withdrawn"
			devReq.RequestedAt = now
			devReq.Client = Client{OidcClient: clientModel}
			devReq.Form = url.Values{"resource": {apiAudience}}
			devReq.RequestedScope = fosite.Arguments{"openid", "read:data"}
			devReq.GrantedScope = fosite.Arguments{"openid", "read:data"}
			devReq.RequestedAudience = fosite.Arguments{apiAudience}
			devReq.GrantedAudience = fosite.Arguments{apiAudience}
			devReq.Session = NewEmptySession()
			devReq.Session.SetExpiresAt(fosite.DeviceCode, now.Add(15*time.Minute))
			devReq.Session.SetExpiresAt(fosite.UserCode, now.Add(15*time.Minute))
			require.NoError(t, store.CreateDeviceAuthSession(t.Context(), deviceCodeSig, userCodeSig, devReq))

			devErr := devSvc.acceptDeviceCode(t.Context(), userCode, userID, "phr", now, nil, "", requestMeta{})
			require.ErrorIs(t, devErr, fosite.ErrAccessDenied, "device acceptance must reject when API grant is withdrawn before client lock")
			assert.GreaterOrEqual(t, resolveCalls.Load(), int32(2))

			var consent int64
			require.NoError(t, db.Table("user_authorized_oidc_clients").Where("client_id = ? AND user_id = ?", clientID, userID).Count(&consent).Error)
			assert.Zero(t, consent, "rejected device acceptance must not persist consent")
		})

		t.Run("authorization_code exchange rejects token issuance when API permission is withdrawn before client lock", func(t *testing.T) {
			allowClient.Store(true)
			const apiAudience = "https://api.example.com"

			apiAccess.onClientAPIScopes = func(context.Context, *gorm.DB, string) ([]string, []string, error) {
				return []string{"read:data"}, []string{apiAudience}, nil
			}
			apiAccess.onAllowedScopesForAudience = func(_ context.Context, tx *gorm.DB, _, audience string, _ SubjectType) ([]string, bool, bool, error) {
				if audience == apiAudience {
					assert.NotNil(t, tx, "code exchange API revalidation must use the active locking transaction")
					return nil, true, false, nil
				}
				return nil, false, false, nil
			}
			apiAccess.onHeldScopes = func(_ context.Context, _ *gorm.DB, _, _ string, scopes []string) ([]string, error) {
				return scopes, nil
			}
			t.Cleanup(func() {
				apiAccess.onClientAPIScopes = nil
				apiAccess.onAllowedScopesForAudience = nil
				apiAccess.onHeldScopes = nil
			})

			code, codeSig, genErr := strategy.GenerateAuthorizeCode(t.Context(), nil)
			require.NoError(t, genErr)
			codeSession := NewEmptySession()
			codeSession.Subject = userID
			codeSession.SessionID = "session-code-api"
			codeSession.Claims = &fositejwt.IDTokenClaims{
				Subject:     userID,
				RequestedAt: now,
				AuthTime:    now,
				Extra:       map[string]any{},
			}
			codeSession.SetExpiresAt(fosite.AuthorizeCode, now.Add(10*time.Minute))
			codeSession.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

			const codeVerifier = "verifier-0123456789abcdefghijklmnopqrstuvwxyz"
			challengeSum := sha256.Sum256([]byte(codeVerifier))
			codeChallenge := base64.RawURLEncoding.EncodeToString(challengeSum[:])

			codeReq := fosite.NewAuthorizeRequest()
			codeReq.ID = "cimd-code-req-api"
			codeReq.RequestedAt = now
			codeReq.Client = Client{OidcClient: clientModel}
			codeReq.RequestedScope = fosite.Arguments{"openid", "read:data"}
			codeReq.GrantedScope = fosite.Arguments{"openid", "read:data"}
			codeReq.RequestedAudience = fosite.Arguments{apiAudience}
			codeReq.GrantedAudience = fosite.Arguments{apiAudience}
			codeReq.Form = url.Values{
				"redirect_uri":          {"https://app.example.com/callback"},
				"code_challenge":        {codeChallenge},
				"code_challenge_method": {"S256"},
			}
			codeReq.Session = codeSession
			require.NoError(t, store.CreateAuthorizeCodeSession(t.Context(), codeSig, codeReq))
			require.NoError(t, store.CreatePKCERequestSession(t.Context(), codeSig, codeReq))

			handler := newTokenHandler(
				provider,
				newClaimsService(db, nil, baseURL, nil, appconfig.StaticResolver{}),
				apiAccess,
				&hookSessions{},
			)

			form := url.Values{
				"grant_type":    {"authorization_code"},
				"code":          {code},
				"client_id":     {clientID},
				"redirect_uri":  {"https://app.example.com/callback"},
				"code_verifier": {codeVerifier},
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			handler.token(c)

			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "access_denied", body["error"], "authorization_code exchange must fail when API grant was withdrawn before client lock")
			assert.Empty(t, body["access_token"])

			var tokenCount int64
			require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ? AND request_id = ?", sessionKindAccessToken, codeReq.ID).Count(&tokenCount).Error)
			assert.Zero(t, tokenCount, "rolled-back code exchange must not persist an access token")
		})

		t.Run("direct RotateRefreshToken wraps row lock and policy check in one transaction", func(t *testing.T) {
			allowClient.Store(false)
			err := store.RotateRefreshToken(t.Context(), request.ID, signature)
			require.ErrorIs(t, err, fosite.ErrInactiveToken)

			allowClient.Store(true)
			require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("metadata_expires_at", nil).Error)
			err = store.RotateRefreshToken(t.Context(), request.ID, signature)
			require.ErrorIs(t, err, fosite.ErrInactiveToken)
			require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("metadata_expires_at", &expiresAt).Error)

			apiAccess.seenTx = nil
			err = store.RotateRefreshToken(t.Context(), request.ID, signature)
			require.NoError(t, err)
			require.NotNil(t, apiAccess.seenTx)
			assert.NotEqual(t, db.Statement.ConnPool, apiAccess.seenTx.Statement.ConnPool, "standalone RotateRefreshToken must run AllowCIMDClient inside the locking transaction")
		})
	})
}
