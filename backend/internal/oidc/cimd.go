package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/pocket-id/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type cimdResolverConfig struct {
	getURLAllowlist    func() []string
	transport          http.RoundTripper
	transportDecorator func(http.RoundTripper) http.RoundTripper
}

type cimdClientResolver struct {
	resolver *fosite.CIMDResolver
	store    *Store
	policy   cimdPolicy
}

var _ fosite.ClientResolver = (*cimdClientResolver)(nil)
var _ fosite.CIMDClientPolicy = cimdPolicy{}

func newCIMDClientResolver(store *Store, config cimdResolverConfig) *cimdClientResolver {
	options := []fosite.CIMDFetcherOption{
		fosite.WithCIMDUserAgent("pocket-id/oidc-client-metadata-fetcher"),
		// A document that forbids caching is still kept briefly, because clients are persisted and must not be able to force a fetch on every request
		fosite.WithCIMDMinCacheTTL(time.Minute),
	}
	if config.transport != nil {
		// The provided transport is the outbound package's guarded transport so we skip Fosite's own pre-resolution checks
		options = append(options, fosite.WithCIMDTransport(config.transport), fosite.WithCIMDAllowPrivateIPs(true))
	}
	if config.transportDecorator != nil {
		options = append(options, fosite.WithCIMDTransportDecorator(config.transportDecorator))
	}

	policy := cimdPolicy{getURLAllowlist: config.getURLAllowlist, apiAccess: store.apiAccess}
	return &cimdClientResolver{
		resolver: &fosite.CIMDResolver{
			Fetcher:                  fosite.NewDefaultCIMDFetcher(options...),
			Cache:                    store,
			Materializer:             store,
			Policy:                   policy,
			MaxConcurrentDiscoveries: 10,
			// An outage of a client's host must not lock out its users, so the last document stays in use for a day
			StaleIfUnavailable: 24 * time.Hour,
			// A client ID that failed is left alone as long as a fetched document is kept at least
			RetryAfterFailure: time.Minute,
		},
		store:  store,
		policy: policy,
	}
}

func (r *cimdClientResolver) ResolveClient(ctx context.Context, clientID string, next fosite.ClientLookupFunc) (fosite.Client, error) {
	if next == nil {
		return nil, errors.New("registered client resolver is required")
	}

	// Exclude persisted metadata clients so Fosite can apply its cache policy while still giving real registrations precedence
	registeredOnly := func(ctx context.Context, clientID string) (fosite.Client, error) {
		client, err := next(ctx, clientID)
		if err != nil {
			return nil, err
		}
		if pocketIDClient, ok := client.(Client); ok && pocketIDClient.IsMetadataDocument() {
			return nil, fosite.ErrNotFound
		}
		return client, nil
	}
	client, err := r.resolver.ResolveClient(ctx, clientID, registeredOnly)
	return client, r.revokeOnClientIDMismatch(ctx, clientID, err)
}

// revokeOnClientIDMismatch revokes what users granted to a client whose document no longer claims its client ID
// The lookup error is returned either way, so the request still fails as an invalid client
func (r *cimdClientResolver) revokeOnClientIDMismatch(ctx context.Context, clientID string, err error) error {
	if !errors.Is(err, fosite.ErrCIMDClientIDMismatch) {
		return err
	}

	var revoked int64
	revokeErr := withTx(ctx, r.store.db, func(ctx context.Context) error {
		var err error
		revoked, err = r.store.revokeClientGrants(ctx, clientID)
		return err
	})
	if revokeErr != nil {
		return errors.Join(err, revokeErr)
	}

	// The mismatch is reported on every request for the client ID, also for one that never was a client
	if revoked > 0 {
		slog.WarnContext(ctx, "Revoked the user consent and tokens of a client whose metadata document no longer matches its client ID",
			slog.String("client_id", clientID),
		)
	}
	return err
}

// RefreshMetadataClient forces a re-fetch of the metadata document for an already-cached CIMD client, bypassing the cache TTL
func (r *cimdClientResolver) RefreshMetadataClient(ctx context.Context, id string) (model.OidcClient, error) {
	if !fosite.LooksLikeCIMDURL(id) {
		return model.OidcClient{}, errors.New("client is not a client ID metadata document client")
	}
	if err := r.policy.AllowCIMDClient(ctx, id); err != nil {
		return model.OidcClient{}, err
	}
	existing, err := r.store.firstClientByID(ctx, id)
	if err != nil {
		return model.OidcClient{}, err
	}
	if !existing.IsMetadataDocument() {
		return model.OidcClient{}, errors.New("client is not a client ID metadata document client")
	}
	client, err := r.resolver.RefreshClient(ctx, id)
	if err != nil {
		return model.OidcClient{}, r.revokeOnClientIDMismatch(ctx, id, err)
	}
	pocketIDClient, ok := client.(Client)
	if !ok {
		return model.OidcClient{}, errors.New("metadata resolver returned an incompatible client")
	}
	return pocketIDClient.OidcClient, nil
}

type cimdPolicy struct {
	getURLAllowlist func() []string
	apiAccess       APIAccessProvider
}

func (p cimdPolicy) cimdURLAllowed(id string) bool {
	if p.getURLAllowlist == nil {
		return false
	}
	return utils.MatchesAnyURLPattern(p.getURLAllowlist(), id)
}

// AllowCIMDClient applies Pocket ID's operator-managed dynamic-client allowlist
func (p cimdPolicy) AllowCIMDClient(ctx context.Context, id string) error {
	if !p.cimdURLAllowed(id) {
		return errors.New("client ID is not in the metadata document allowlist")
	}

	// A client ID is also the audience of the client's login tokens, so one that names an API could have them accepted there
	// The issuer needs no check of its own, because it has no path and a client ID must have one
	if p.apiAccess != nil {
		_, apiExists, _, err := p.apiAccess.AllowedScopesForAudience(ctx, nil, id, canonicalResource(id), SubjectTypeUser)
		if err != nil {
			return err
		}
		if apiExists {
			return errors.New("client ID must not be the audience of an API")
		}
	}
	return nil
}

// ValidateCIMDClient restricts generic CIMD features to those supported by Pocket ID's client model
func (cimdPolicy) ValidateCIMDClient(_ context.Context, doc *fosite.ClientMetadataDocument) error {
	// Require public-client authentication because Pocket ID does not persist CIMD key material
	// A client like ChatGPT lists "none" next to the methods it prefers, and those are not an error
	if methods := doc.TokenEndpointAuthMethods(); !slices.Contains(methods, "none") {
		return fmt.Errorf("client metadata documents must support the token endpoint authentication method %q, got %q", "none", methods)
	}

	// Require a flow that can initiate authorization (ignoring the grant types Pocket ID does not implement)
	if !hasInitiatingGrantType(supportedMetadataGrantTypes(doc.GrantTypes)) {
		return errInitiatingGrantRequired
	}

	// Pocket ID only implements the code response type for metadata clients, which is also the default of an omitted list
	// Other declared response types are ignored, so the document is rejected only when code is not among them
	if len(doc.ResponseTypes) > 0 && !slices.Contains(doc.ResponseTypes, "code") {
		return fmt.Errorf("client metadata document must support the response type %q, got %q", "code", doc.ResponseTypes)
	}

	return nil
}

var errInitiatingGrantRequired = errors.New("client metadata document must enable authorization_code or device_code")

// supportedMetadataGrantTypes keeps only the grant types Pocket ID implements for metadata clients and drops the rest
func supportedMetadataGrantTypes(grantTypes []string) []string {
	if len(grantTypes) == 0 {
		// Per RFC 7591 section 2: an omitted grant_types defaults to authorization_code
		return []string{string(fosite.GrantTypeAuthorizationCode)}
	}

	supported := make([]string, 0, len(grantTypes))
	for _, grantType := range grantTypes {
		switch grantType {
		case string(fosite.GrantTypeAuthorizationCode), string(fosite.GrantTypeDeviceCode), string(fosite.GrantTypeRefreshToken):
			supported = append(supported, grantType)
		}
	}
	return supported
}

func hasInitiatingGrantType(grantTypes []string) bool {
	return slices.Contains(grantTypes, string(fosite.GrantTypeAuthorizationCode)) ||
		slices.Contains(grantTypes, string(fosite.GrantTypeDeviceCode))
}

// validateMetadataRedirectURIs rejects self-asserted redirect URIs Pocket ID must not accept
func validateMetadataRedirectURIs(field string, uris []string) error {
	for _, raw := range uris {
		if strings.Contains(raw, "*") {
			return fmt.Errorf("%s entry %q must not contain a wildcard", field, raw)
		}
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("%s entry %q is not a valid URL: %w", field, raw, err)
		}
		if !u.IsAbs() {
			return fmt.Errorf("%s entry %q must be an absolute URL", field, raw)
		}
		// Mirrors the scheme restriction every administrator-registered callback URL passes
		switch strings.ToLower(u.Scheme) {
		case "javascript", "data":
			return fmt.Errorf("%s entry %q uses a disallowed scheme", field, raw)
		}
	}
	return nil
}

// buildClientFromMetadata applies Pocket ID's persisted-client projection to validated generic metadata
func buildClientFromMetadata(doc *fosite.ClientMetadataDocument, rawURL string) (model.OidcClient, error) {
	if err := validateMetadataRedirectURIs("redirect_uris", doc.RedirectURIs); err != nil {
		return model.OidcClient{}, err
	}
	if err := validateMetadataRedirectURIs("post_logout_redirect_uris", doc.PostLogoutRedirectURIs); err != nil {
		return model.OidcClient{}, err
	}

	// Record the supported grant types the document declares, so the client is neither silently granted capabilities it never declared nor persisted with grants Pocket ID cannot honor
	grantTypes := supportedMetadataGrantTypes(doc.GrantTypes)
	if !hasInitiatingGrantType(grantTypes) {
		return model.OidcClient{}, errInitiatingGrantRequired
	}

	client := model.OidcClient{
		Base:                        model.Base{ID: rawURL},
		Name:                        doc.ClientName,
		CallbackURLs:                datatype.StringList(doc.RedirectURIs),
		LogoutCallbackURLs:          datatype.StringList(doc.PostLogoutRedirectURIs),
		ClientType:                  model.OidcClientTypeCIMD,
		IsPublic:                    true,
		PkceEnabled:                 true,
		MetadataGrantTypes:          datatype.StringList(grantTypes),
		AccessTokenDurationMinutes:  model.DefaultAccessTokenDurationMinutes,
		RefreshTokenDurationMinutes: model.DefaultRefreshTokenDurationMinutes,
	}

	if client.Name == "" {
		if u, err := url.Parse(rawURL); err == nil {
			client.Name = u.Host
		}
	}

	return client, nil
}

// MaterializeCIMDClient converts validated generic metadata into Pocket ID's runtime client
func (s *Store) MaterializeCIMDClient(_ context.Context, doc *fosite.ClientMetadataDocument) (fosite.Client, error) {
	client, err := buildClientFromMetadata(doc, doc.ClientID)
	if err != nil {
		return nil, err
	}
	return Client{OidcClient: client}, nil
}
