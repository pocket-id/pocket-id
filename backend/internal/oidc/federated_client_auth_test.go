package oidc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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

	raw, err := json.Marshal(set)
	require.NoError(t, err)

	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(raw))),
			Request:    req,
		}, nil
	})}
}

// TestFederatedClientAuthenticatorAssertionReplayProtection covers the replay/exp hardening
// of federated client assertions: a JTI may only be used once, and assertions must carry
// both a jti and an exp claim.
func TestFederatedClientAuthenticatorAssertionReplayProtection(t *testing.T) {
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
	authenticator := newFederatedClientAuthenticator(store, newJWKSetHTTPClient(t, jwks), audience)

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

	t.Run("valid assertion authenticates once, replay is rejected", func(t *testing.T) {
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.JwtID("jti-once").Expiration(time.Now().Add(5 * time.Minute))
		})

		client, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.NoError(t, err)
		require.Equal(t, clientID, client.GetID())

		_, err = authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("assertion without exp is rejected", func(t *testing.T) {
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.JwtID("jti-no-exp")
		})
		_, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
	})

	t.Run("assertion without jti is rejected", func(t *testing.T) {
		assertion := signAssertion(t, func(b *jwt.Builder) *jwt.Builder {
			return b.Expiration(time.Now().Add(5 * time.Minute))
		})
		_, err := authenticator.authenticateAssertion(t.Context(), assertion, clientID)
		require.ErrorIs(t, err, fosite.ErrInvalidClient)
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
	authenticator := newFederatedClientAuthenticator(store, newJWKSetHTTPClient(t, jwks), "unused-default-audience")

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
