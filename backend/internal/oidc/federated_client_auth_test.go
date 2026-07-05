package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type fakeFederatedStore struct {
	client fosite.Client
	jtis   map[string]time.Time
}

func (f *fakeFederatedStore) GetClient(_ context.Context, id string) (fosite.Client, error) {
	if f.client == nil || f.client.GetID() != id {
		return nil, fosite.ErrNotFound
	}
	return f.client, nil
}

func (f *fakeFederatedStore) ClientAssertionJWTValid(_ context.Context, jti string) error {
	if exp, ok := f.jtis[jti]; ok && exp.After(time.Now()) {
		return fosite.ErrJTIKnown
	}
	return nil
}

func (f *fakeFederatedStore) SetClientAssertionJWT(_ context.Context, jti string, exp time.Time) error {
	f.jtis[jti] = exp
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newJWKSetHTTPClient(t *testing.T, set jwk.Set) *http.Client {
	t.Helper()

	client, _ := newCountingJWKSetHTTPClient(t, set, false)
	return client
}

func newCountingJWKSetHTTPClient(t *testing.T, set jwk.Set, failAfterFirst bool) (*http.Client, *atomic.Int64) {
	t.Helper()

	raw, err := json.Marshal(set)
	require.NoError(t, err)

	var requests atomic.Int64
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if failAfterFirst && requests.Load() > 0 {
			return nil, errors.New("jwks endpoint unavailable")
		}
		requests.Add(1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(raw))),
			Request:    req,
		}, nil
	})}, &requests
}

// TestFederatedClientAuthenticatorAssertionValidation covers federated client assertions
// issued by providers that may cache and reuse tokens until their exp claim.
func TestFederatedClientAuthenticatorAssertionValidation(t *testing.T) {
	signingKey, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	require.NoError(t, err)
	signingAlg, ok := signingKey.Algorithm()
	require.True(t, ok)

	publicKey, err := signingKey.PublicKey()
	require.NoError(t, err)
	jwks := jwk.NewSet()
	require.NoError(t, jwks.AddKey(publicKey))

	const (
		issuer   = "https://idp.example.com"
		clientID = "federated-client"
		audience = "https://pocket-id.example.com"
		jwksURL  = "https://idp.example.com/jwks.json"
	)

	newAuthenticator := func(t *testing.T, replayProtection bool) (*federatedClientAuthenticator, *fakeFederatedStore) {
		t.Helper()
		store := &fakeFederatedStore{
			client: Client{OidcClient: model.OidcClient{
				Base: model.Base{ID: clientID},
				Name: "Federated Client",
				Credentials: model.OidcClientCredentials{
					FederatedIdentities: []model.OidcClientFederatedIdentity{{
						Issuer:           issuer,
						JWKS:             jwksURL,
						ReplayProtection: replayProtection,
					}},
				},
			}},
			jtis: map[string]time.Time{},
		}
		authenticator, err := newFederatedClientAuthenticator(t.Context(), store, newJWKSetHTTPClient(t, jwks), audience)
		require.NoError(t, err)
		return authenticator, store
	}

	signAssertion := func(t *testing.T, mutate func(b *jwt.Builder) *jwt.Builder) string {
		t.Helper()
		builder := jwt.NewBuilder().
			Issuer(issuer).
			Subject(clientID).
			Audience([]string{audience}).
			IssuedAt(time.Now())
		token, err := mutate(builder).Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithKey(signingAlg, signingKey))
		require.NoError(t, err)
		return string(signed)
	}

	t.Run("valid assertion with jti authenticates once when replay protection is enabled", func(t *testing.T) {
		authenticator, _ := newAuthenticator(t, true)
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.JwtID("replay-protected-token").Expiration(time.Now().Add(5 * time.Minute))
		})

		client, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.NoError(t, err)
		require.Equal(t, clientID, client.GetID())

		_, err = authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("valid assertion with jti can be reused when replay protection is disabled", func(t *testing.T) {
		authenticator, store := newAuthenticator(t, false)
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.JwtID("cached-provider-token").Expiration(time.Now().Add(5 * time.Minute))
		})

		client, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.NoError(t, err)
		require.Equal(t, clientID, client.GetID())

		client, err = authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.NoError(t, err)
		require.Equal(t, clientID, client.GetID())
		require.Empty(t, store.jtis)
	})

	t.Run("assertion without exp is rejected", func(t *testing.T) {
		authenticator, _ := newAuthenticator(t, true)
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.JwtID("jti-no-exp")
		})
		_, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("assertion without jti is rejected when replay protection is enabled", func(t *testing.T) {
		authenticator, _ := newAuthenticator(t, true)
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.Expiration(time.Now().Add(5 * time.Minute))
		})
		_, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("assertion without jti authenticates when replay protection is disabled", func(t *testing.T) {
		authenticator, store := newAuthenticator(t, false)
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.Expiration(time.Now().Add(5 * time.Minute))
		})
		client, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.NoError(t, err)
		require.Equal(t, clientID, client.GetID())
		require.Empty(t, store.jtis)
	})
}

func TestFederatedClientAuthenticatorAllowsConfiguredSubjectDifferentFromClientID(t *testing.T) {
	signingKey, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	require.NoError(t, err)
	signingAlg, ok := signingKey.Algorithm()
	require.True(t, ok)

	publicKey, err := signingKey.PublicKey()
	require.NoError(t, err)
	jwks := jwk.NewSet()
	require.NoError(t, jwks.AddKey(publicKey))

	const (
		issuer   = "https://idp.example.com"
		clientID = "pocket-id-client"
		subject  = "external-workload-subject"
		audience = "api://pocket-id"
		jwksURL  = "https://idp.example.com/jwks.json"
	)

	store := &fakeFederatedStore{
		client: Client{OidcClient: model.OidcClient{
			Base: model.Base{ID: clientID},
			Name: "Federated Client",
			Credentials: model.OidcClientCredentials{
				FederatedIdentities: []model.OidcClientFederatedIdentity{{
					Issuer:   issuer,
					Subject:  subject,
					Audience: audience,
					JWKS:     jwksURL,
				}},
			},
		}},
		jtis: map[string]time.Time{},
	}
	authenticator, err := newFederatedClientAuthenticator(t.Context(), store, newJWKSetHTTPClient(t, jwks), "unused-default-audience")
	require.NoError(t, err)

	token, err := jwt.NewBuilder().
		Issuer(issuer).
		Subject(subject).
		Audience([]string{audience}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(5 * time.Minute)).
		JwtID("jti-custom-subject").
		Build()
	require.NoError(t, err)
	assertion, err := jwt.Sign(token, jwt.WithKey(signingAlg, signingKey))
	require.NoError(t, err)

	client, err := authenticator.authenticateAssertion(t.Context(), string(assertion), clientID)
	require.NoError(t, err)
	require.Equal(t, clientID, client.GetID())
}

func TestFederatedClientAuthenticatorCachesJWKS(t *testing.T) {
	signingKey, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	require.NoError(t, err)
	signingAlg, ok := signingKey.Algorithm()
	require.True(t, ok)

	publicKey, err := signingKey.PublicKey()
	require.NoError(t, err)
	jwks := jwk.NewSet()
	require.NoError(t, jwks.AddKey(publicKey))

	const (
		issuer   = "https://idp.example.com"
		clientID = "federated-client"
		audience = "https://pocket-id.example.com"
		jwksURL  = "https://idp.example.com/jwks.json"
	)

	httpClient, requests := newCountingJWKSetHTTPClient(t, jwks, true)
	store := &fakeFederatedStore{
		client: Client{OidcClient: model.OidcClient{
			Base: model.Base{ID: clientID},
			Name: "Federated Client",
			Credentials: model.OidcClientCredentials{
				FederatedIdentities: []model.OidcClientFederatedIdentity{{
					Issuer: issuer,
					JWKS:   jwksURL,
				}},
			},
		}},
		jtis: map[string]time.Time{},
	}
	authenticator, err := newFederatedClientAuthenticator(t.Context(), store, httpClient, audience)
	require.NoError(t, err)

	signAssertion := func(t *testing.T, jti string) string {
		t.Helper()
		token, err := jwt.NewBuilder().
			Issuer(issuer).
			Subject(clientID).
			Audience([]string{audience}).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5 * time.Minute)).
			JwtID(jti).
			Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithKey(signingAlg, signingKey))
		require.NoError(t, err)
		return string(signed)
	}

	client, err := authenticator.authenticateAssertion(t.Context(), signAssertion(t, "jti-cache-1"), clientID)
	require.NoError(t, err)
	require.Equal(t, clientID, client.GetID())

	client, err = authenticator.authenticateAssertion(t.Context(), signAssertion(t, "jti-cache-2"), clientID)
	require.NoError(t, err)
	require.Equal(t, clientID, client.GetID())
	require.EqualValues(t, 1, requests.Load())
}
