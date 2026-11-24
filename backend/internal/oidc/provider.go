package oidc

import (
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/pocket-id/fosite"
	"github.com/pocket-id/fosite/compose"
	fositeoauth2 "github.com/pocket-id/fosite/handler/oauth2"
	"github.com/pocket-id/fosite/handler/openid"
	"github.com/pocket-id/fosite/token/jwt"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/crypto/hkdf"
)

type oidcProvider struct {
	fosite.OAuth2Provider
	deviceStrategy *deviceStrategy
	tokenStrategies
}

type tokenStrategies struct {
	accessToken fositeoauth2.AccessTokenStrategy
	idToken     openid.OpenIDConnectTokenStrategy
	config      *fosite.Config
}

func newProvider(store *Store, authenticator *federatedClientAuthenticator, signer TokenSigner, config Config, clientResolver fosite.ClientResolver) (*oidcProvider, error) {
	secret, err := DeriveGlobalSecret(config.Secret)
	if err != nil {
		return nil, err
	}

	devicePollingInterval := 5 * time.Second
	if common.EnvConfig.AppEnv.IsTest() {
		// The E2E tests wait out the polling interval before each device token request
		devicePollingInterval = time.Second
	}

	var fositeConfig = &fosite.Config{
		RefreshTokenLifespan:                    30 * 24 * time.Hour,
		DeviceAndUserCodeLifespan:               15 * time.Minute,
		DeviceAuthTokenPollingInterval:          devicePollingInterval,
		DeviceVerificationURL:                   config.BaseURL + "/device",
		PushedAuthorizeContextLifespan:          90 * time.Second,
		IDTokenIssuer:                           config.BaseURL,
		AccessTokenIssuer:                       config.BaseURL,
		AuthorizationResponseIssuer:             config.BaseURL,
		TokenURL:                                config.TokenBaseURL + "/api/oidc/token",
		ScopeStrategy:                           fosite.ExactScopeStrategy,
		IgnoreUnknownScopes:                     true,
		AudienceMatchingStrategy:                fosite.ExactAudienceMatchingStrategy,
		RedirectURIMatcher:                      matchRedirectURI,
		RedirectSecureChecker:                   redirectSecureChecker(config.AllowInsecureCallbackURLs),
		EnforcePKCEForPublicClients:             true,
		EnablePKCEPlainChallengeMethod:          true,
		SupportedRequestObjectSigningAlgorithms: []string{"none"},
		FormPostHTMLTemplate:                    formPostTemplate,
		RefreshTokenScopes:                      []string{},
		GlobalSecret:                            secret,
		JWTScopeClaimKey:                        jwt.JWTScopeFieldBoth,
		ClientResolver:                          clientResolver,
		ClientSecretsHasher:                     clientSecretHasher{},
	}

	keyGetter := func(context.Context) (interface{}, error) {
		return SigningKeyFromSigner(signer)
	}
	sig := newJWTSigner(keyGetter)
	deviceStrategy := &deviceStrategy{DefaultDeviceStrategy: compose.NewDeviceStrategy(fositeConfig)}
	rfc9068AccessTokenStrategy := &fositeoauth2.RFC9068JWTStrategy{
		DefaultJWTStrategy: &fositeoauth2.DefaultJWTStrategy{
			Signer: sig,
			Config: fositeConfig,
		},
	}

	// Apply Pocket ID's identity-audience policy outside Fosite's reusable RFC 9068 token profile
	accessTokenStrategy := NewAccessTokenStrategy(rfc9068AccessTokenStrategy, config.BaseURL)
	idTokenStrategy := &openid.DefaultStrategy{
		Signer: sig,
		Config: fositeConfig,
	}
	provider := compose.Compose(
		fositeConfig,
		store,
		&strategyProvider{
			opaque:      compose.NewOAuth2HMACStrategy(fositeConfig),
			accessToken: accessTokenStrategy,
			device:      deviceStrategy,
			idToken:     idTokenStrategy,
			Signer:      sig,
		},
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2ClientCredentialsGrantFactory,
		compose.OAuth2RefreshTokenGrantFactory,
		compose.RFC8628DeviceFactory,
		compose.RFC8628DeviceAuthorizationTokenFactory,
		compose.OpenIDConnectExplicitFactory,
		compose.OpenIDConnectRefreshFactory,
		compose.OpenIDConnectDeviceFactory,
		compose.OAuth2TokenIntrospectionFactory,
		compose.OAuth2PKCEFactory,
		compose.PushedAuthorizeHandlerFactory,
	).(*fosite.Fosite)

	fositeConfig.ClientAuthenticationStrategy = newClientAuthenticationStrategy(authenticator, provider)
	return &oidcProvider{
		OAuth2Provider: provider,
		deviceStrategy: deviceStrategy,
		tokenStrategies: tokenStrategies{
			accessToken: accessTokenStrategy,
			idToken:     idTokenStrategy,
			config:      fositeConfig,
		},
	}, nil
}

func redirectSecureChecker(allowInsecureCallbackURLs bool) func(context.Context, *url.URL) bool {
	return func(ctx context.Context, redirectURI *url.URL) bool {
		if allowInsecureCallbackURLs || fosite.IsRedirectURISecure(ctx, redirectURI) {
			return true
		}

		slog.InfoContext(ctx, "HTTP callback URL rejected; set ALLOW_INSECURE_CALLBACK_URLS=true to allow it", "callback_url", redirectURI.Redacted())
		return false
	}
}

func matchRedirectURI(rawurl string, client fosite.Client) (*url.URL, error) {
	// CIMD clients register exact callback URIs and must never fall through to Fosite's decoded-path loopback or glob matching
	if pocketClient, ok := client.(Client); ok && pocketClient.IsMetadataDocument() {
		if matched := matchExactOrLoopbackRedirectURI(client.GetRedirectURIs(), rawurl); matched != nil {
			return matched, nil
		}
		return nil, fosite.ErrInvalidRequest.WithHint("The 'redirect_uri' parameter does not match any of the OAuth 2.0 Client's pre-registered 'redirect_uris'.")
	}

	redirectURI, err := fosite.MatchRedirectURIWithClientRedirectURIs(rawurl, client)
	if err == nil || rawurl == "" {
		return redirectURI, err
	}

	invalidRedirectErr := err
	matchedURL, matchErr := utils.GetCallbackURLFromList(client.GetRedirectURIs(), rawurl)
	if matchErr != nil || matchedURL == "" {
		slog.Debug("Redirect URI does not match any of the registered callback URLs", "rawurl", rawurl, "client_id", client.GetID(), "error", matchErr)
		return nil, invalidRedirectErr
	}

	redirectURI, err = url.Parse(matchedURL)
	if err != nil || !fosite.IsValidRedirectURI(redirectURI) {
		slog.Debug("Matched callback URL is invalid", "matchedURL", matchedURL, "client_id", client.GetID(), "error", err)
		return nil, invalidRedirectErr
	}

	return redirectURI, nil
}

// matchExactRedirectURI requires exact string equality without loopback port variance (used for RP-Initiated Logout §3 post_logout_redirect_uri)
func matchExactRedirectURI(registered []string, rawurl string) *url.URL {
	if rawurl == "" {
		if len(registered) != 1 || strings.Contains(registered[0], "#") {
			return nil
		}
		parsed, err := url.Parse(registered[0])
		if err != nil || !fosite.IsValidRedirectURI(parsed) || parsed.User != nil {
			return nil
		}
		return parsed
	}
	if strings.Contains(rawurl, "#") {
		return nil
	}
	reqURL, err := url.Parse(rawurl)
	if err != nil || !fosite.IsValidRedirectURI(reqURL) || reqURL.User != nil {
		return nil
	}
	for _, candidate := range registered {
		if candidate == rawurl {
			return reqURL
		}
	}
	return nil
}

// matchExactOrLoopbackRedirectURI requires exact string equality, allowing only RFC 8252 ephemeral port differences for HTTP loopback IP redirects
func matchExactOrLoopbackRedirectURI(registered []string, rawurl string) *url.URL {
	if exact := matchExactRedirectURI(registered, rawurl); exact != nil || rawurl == "" {
		return exact
	}
	if strings.Contains(rawurl, "#") {
		return nil
	}
	reqURL, err := url.Parse(rawurl)
	if err != nil || !fosite.IsValidRedirectURI(reqURL) || reqURL.User != nil {
		return nil
	}
	reqStripped, ok := stripLoopbackPort(rawurl, reqURL)
	if !ok {
		return nil
	}
	for _, candidate := range registered {
		if strings.Contains(candidate, "#") {
			continue
		}
		regURL, parseErr := url.Parse(candidate)
		if parseErr != nil || !fosite.IsValidRedirectURI(regURL) || regURL.User != nil {
			continue
		}
		if candStripped, ok := stripLoopbackPort(candidate, regURL); ok && candStripped == reqStripped {
			return reqURL
		}
	}
	return nil
}

// stripLoopbackPort removes only the optional :port component from an HTTP loopback IP URI while preserving the original scheme, host literal, path, and query bytes
func stripLoopbackPort(raw string, parsed *url.URL) (string, bool) {
	if parsed.Scheme != "http" {
		return "", false
	}
	addr, err := netip.ParseAddr(parsed.Hostname())
	if err != nil || addr.Zone() != "" || !addr.IsLoopback() {
		return "", false
	}
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return "", false
	}
	authStart := schemeEnd + 3
	rest := raw[authStart:]
	authEnd := strings.IndexAny(rest, "/?")
	if authEnd < 0 {
		authEnd = len(rest)
	}
	authority := rest[:authEnd]
	host := authority
	if strings.HasPrefix(authority, "[") {
		closeBracket := strings.IndexByte(authority, ']')
		if closeBracket < 0 {
			return "", false
		}
		host = authority[:closeBracket+1]
		portPart := authority[closeBracket+1:]
		if portPart != "" && !strings.HasPrefix(portPart, ":") {
			return "", false
		}
	} else if colon := strings.IndexByte(authority, ':'); colon >= 0 {
		host = authority[:colon]
	}
	return raw[:authStart] + host + rest[authEnd:], true
}

// DeriveGlobalSecret derives a 32-byte secret from the provided secret.
// Note: changing this function in any is considered a breaking change.
func DeriveGlobalSecret(secret []byte) ([]byte, error) {
	const info = "pocketid/fosite_global_secret"
	r := hkdf.New(sha256.New, secret, nil, []byte(info))

	key := make([]byte, 32)
	_, err := io.ReadFull(r, key)
	if err != nil {
		return nil, err
	}

	return key, nil
}
