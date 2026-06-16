package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/ory/fosite"
)

const clientAssertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" //nolint:gosec

var errNoFederatedClientAssertion = errors.New("no federated client assertion")

// federatedClientStore is the subset of the store the federated authenticator needs:
// loading clients and tracking client-assertion JTIs for single-use replay protection.
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
	defaultAudience string
}

func newFederatedClientAuthenticator(clients federatedClientStore, httpClient *http.Client, defaultAudience string) *federatedClientAuthenticator {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &federatedClientAuthenticator{
		clients:         clients,
		httpClient:      httpClient,
		defaultAudience: defaultAudience,
	}
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

	// Enforce single use of the assertion to prevent replay
	jti, ok := parsed.JwtID()
	if !ok || jti == "" {
		return nil, fosite.ErrInvalidClient.WithHint("Client assertion is missing the 'jti' claim.")
	}
	if err := a.clients.ClientAssertionJWTValid(ctx, jti); err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Client assertion has already been used.").WithWrap(err)
	}
	exp, _ := parsed.Expiration()
	if err := a.clients.SetClientAssertionJWT(ctx, jti, exp); err != nil {
		return nil, fosite.ErrInvalidClient.WithWrap(err)
	}

	return client, nil
}

func (a *federatedClientAuthenticator) fetchJWKSet(ctx context.Context, jwksURL string) (jwk.Set, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	return jwk.ParseReader(resp.Body)
}
