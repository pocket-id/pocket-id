package oidc

import (
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	fositeoauth2 "github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/handler/rfc8628"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/crypto/hkdf"
)

type oidcProvider struct {
	fosite.OAuth2Provider
	deviceStrategy *rfc8628.DefaultDeviceStrategy
	tokenStrategies
}

type tokenStrategies struct {
	accessToken fositeoauth2.AccessTokenStrategy
	idToken     openid.OpenIDConnectTokenStrategy
	config      *fosite.Config
}

func newProvider(store *Store, authenticator *federatedClientAuthenticator, signer TokenSigner, config Config) (*oidcProvider, error) {
	secret, err := DeriveGlobalSecret(config.Secret)
	if err != nil {
		return nil, err
	}

	var fositeConfig = &fosite.Config{
		RefreshTokenLifespan:           30 * 24 * time.Hour,
		DeviceAndUserCodeLifespan:      15 * time.Minute,
		DeviceAuthTokenPollingInterval: 5 * time.Second,
		DeviceVerificationURL:          config.BaseURL + "/device",
		PushedAuthorizeContextLifespan: 90 * time.Second,
		IDTokenIssuer:                  config.BaseURL,
		AccessTokenIssuer:              config.BaseURL,
		TokenURL:                       config.TokenBaseURL + "/api/oidc/token",
		ScopeStrategy:                  fosite.ExactScopeStrategy,
		AudienceMatchingStrategy:       fosite.ExactAudienceMatchingStrategy,
		RedirectURIMatcher:             matchRedirectURI,
		EnforcePKCEForPublicClients:    true,
		EnablePKCEPlainChallengeMethod: true,
		FormPostHTMLTemplate:           formPostTemplate,
		RefreshTokenScopes:             []string{},
		GlobalSecret:                   secret,
	}

	keyGetter := func(context.Context) (interface{}, error) {
		return SigningKeyFromSigner(signer)
	}
	sig := newJWTSigner(keyGetter)
	coreStrategy := compose.NewOAuth2HMACStrategy(fositeConfig)
	deviceStrategy := compose.NewDeviceStrategy(fositeConfig)
	accessTokenStrategy := &fositeoauth2.DefaultJWTStrategy{
		Signer:          sig,
		HMACSHAStrategy: coreStrategy,
		Config:          fositeConfig,
	}
	idTokenStrategy := &openid.DefaultStrategy{
		Signer: sig,
		Config: fositeConfig,
	}
	provider := compose.Compose(
		fositeConfig,
		store,
		&compose.CommonStrategy{
			CoreStrategy:               accessTokenStrategy,
			RFC8628CodeStrategy:        deviceStrategy,
			OpenIDConnectTokenStrategy: idTokenStrategy,
			Signer:                     sig,
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

func matchRedirectURI(rawurl string, client fosite.Client) (*url.URL, error) {
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

// DeriveGlobalSecret derives a 32-byte secret from the provided secret.
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
