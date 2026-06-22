package oidc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/httprc/v3/errsink"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/ory/fosite"
)

const clientAssertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" //nolint:gosec

var errNoFederatedClientAssertion = errors.New("no federated client assertion")

// federatedClientStore is the subset of the store the federated authenticator needs.
type federatedClientStore interface {
	GetClient(ctx context.Context, id string) (fosite.Client, error)
	ClientAssertionJWTValid(ctx context.Context, jti string) error
	SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error
}

// federatedClientAuthenticator authenticates clients via JWT bearer assertions issued
// by a federated identity provider configured per client.
type federatedClientAuthenticator struct {
	clients         federatedClientStore
	httpClient      *http.Client
	jwksCache       *jwk.Cache
	defaultAudience string
}

func newFederatedClientAuthenticator(ctx context.Context, clients federatedClientStore, httpClient *http.Client, defaultAudience string) (*federatedClientAuthenticator, error) {
	authenticator := &federatedClientAuthenticator{
		clients:         clients,
		httpClient:      httpClient,
		defaultAudience: defaultAudience,
	}

	jwksCache, err := authenticator.getJWKCache(ctx)
	if err != nil {
		return nil, err
	}
	authenticator.jwksCache = jwksCache

	return authenticator, nil
}

func (a *federatedClientAuthenticator) getJWKCache(ctx context.Context) (*jwk.Cache, error) {
	// We need to create a custom HTTP client to set a timeout.
	client := a.httpClient
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}

		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			// Indicates a development-time error
			panic("Default transport is not of type *http.Transport")
		}
		transport := defaultTransport.Clone()
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
		client.Transport = transport
	}

	return jwk.NewCache(ctx,
		httprc.NewClient(
			httprc.WithErrorSink(errsink.NewSlog(slog.Default())),
			httprc.WithHTTPClient(client),
		),
	)
}

// newClientAuthenticationStrategy accepts federated client assertions before falling
// back to fosite's default client authentication.
func newClientAuthenticationStrategy(authenticator *federatedClientAuthenticator, provider *fosite.Fosite) fosite.ClientAuthenticationStrategy {
	return func(ctx context.Context, r *http.Request, form url.Values) (fosite.Client, error) {
		client, err := authenticator.authenticateForm(ctx, form)
		if err == nil {
			return client, nil
		}
		if !errors.Is(err, errNoFederatedClientAssertion) {
			return nil, err
		}

		return provider.DefaultClientAuthenticationStrategy(ctx, r, form)
	}
}

// authenticateForm returns errNoFederatedClientAssertion when the form carries no
// federated assertion, so the caller can fall back to other authentication methods.
func (a *federatedClientAuthenticator) authenticateForm(ctx context.Context, form url.Values) (fosite.Client, error) {
	if form.Get("client_assertion_type") != clientAssertionTypeJWTBearer || form.Get("client_assertion") == "" {
		return nil, errNoFederatedClientAssertion
	}

	return a.authenticateAssertion(ctx, form.Get("client_assertion"), form.Get("client_id"))
}

// authenticateAssertion validates the assertion JWT against the client's configured
// federated identity. An empty clientID falls back to the assertion's subject.
func (a *federatedClientAuthenticator) authenticateAssertion(ctx context.Context, assertion string, clientID string) (fosite.Client, error) {
	rawAssertion := []byte(assertion)
	insecureToken, err := jwt.ParseInsecure(rawAssertion)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Invalid client assertion.").WithWrap(err)
	}

	issuer, _ := insecureToken.Issuer()
	if issuer == "" {
		return nil, fosite.ErrInvalidClient.WithHint("Client assertion is missing issuer.")
	}

	if clientID == "" {
		clientID, _ = insecureToken.Subject()
	}
	if clientID == "" {
		return nil, fosite.ErrInvalidClient.WithHint("Client assertion is missing subject.")
	}

	client, err := a.clients.GetClient(ctx, clientID)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithWrap(err)
	}

	oidcClient, ok := client.(Client)
	if !ok {
		return nil, errNoFederatedClientAssertion
	}

	federatedIdentity, ok := oidcClient.Credentials.FederatedIdentityForIssuer(issuer)
	if !ok {
		return nil, errNoFederatedClientAssertion
	}

	jwksURL := federatedIdentity.JWKS
	if jwksURL == "" {
		jwksURL = strings.TrimRight(issuer, "/") + "/.well-known/jwks.json"
	}

	jwks, err := a.fetchJWKSet(ctx, jwksURL)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unable to fetch client assertion JWKS.").WithWrap(err)
	}

	audience := federatedIdentity.Audience
	if audience == "" {
		audience = a.defaultAudience
	}
	subject := federatedIdentity.Subject
	if subject == "" {
		subject = client.GetID()
	}

	parsed, err := jwt.Parse(rawAssertion,
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(30*time.Second),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithIssuer(issuer),
		jwt.WithSubject(subject),
		jwt.WithAudience(audience),
		jwt.WithKeySet(jwks, jws.WithInferAlgorithmFromKey(true), jws.WithUseDefault(true)),
	)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Invalid client assertion.").WithWrap(err)
	}

	if federatedIdentity.ReplayProtection {
		jti, ok := parsed.JwtID()
		if !ok || jti == "" {
			return nil, fosite.ErrInvalidClient.WithHint("Client assertion is missing jti claim, which is required for replay protection.")
		}

		// Check if the jti has been used before
		if err := a.clients.ClientAssertionJWTValid(ctx, jti); err != nil {
			return nil, fosite.ErrInvalidClient.WithHint("Client assertion has already been used.").WithWrap(err)
		}
		// Store the jti to prevent future reuse
		exp, _ := parsed.Expiration()
		if err := a.clients.SetClientAssertionJWT(ctx, jti, exp); err != nil {
			return nil, fosite.ErrInvalidClient.WithWrap(err)
		}

	}

	return client, nil
}

func (a *federatedClientAuthenticator) fetchJWKSet(ctx context.Context, jwksURL string) (jwk.Set, error) {
	if !a.jwksCache.IsRegistered(ctx, jwksURL) {
		// We set a timeout because otherwise Register will keep trying in case of errors
		registerCtx, registerCancel := context.WithTimeout(ctx, 15*time.Second)
		defer registerCancel()

		registerOptions := []jwk.RegisterOption{
			jwk.WithMaxInterval(24 * time.Hour),
			jwk.WithMinInterval(15 * time.Minute),
			jwk.WithWaitReady(true),
		}
		if a.httpClient != nil {
			registerOptions = append(registerOptions, jwk.WithHTTPClient(a.httpClient))
		}

		// We need to register the URL
		err := a.jwksCache.Register(registerCtx, jwksURL, registerOptions...)
		// In case of race conditions (two goroutines calling jwkCache.Register at the same time), it's possible we can get a conflict anyways, so we ignore that error
		if err != nil && !errors.Is(err, httprc.ErrResourceAlreadyExists()) {
			return nil, fmt.Errorf("failed to register JWK set: %w", err)
		}
	}

	jwks, err := a.jwksCache.CachedSet(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached JWK set: %w", err)
	}

	return jwks, nil
}
