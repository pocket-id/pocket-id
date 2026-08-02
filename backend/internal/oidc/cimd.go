package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ory/fosite"

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
		fosite.WithCIMDExtraPrivateRanges(utils.LocalIPv6IPNets()),
	}
	if config.transport != nil {
		options = append(options, fosite.WithCIMDTransport(config.transport))
	}
	if config.transportDecorator != nil {
		options = append(options, fosite.WithCIMDTransportDecorator(config.transportDecorator))
	}

	policy := cimdPolicy{getURLAllowlist: config.getURLAllowlist}
	return &cimdClientResolver{
		resolver: &fosite.CIMDResolver{
			Fetcher:                  fosite.NewDefaultCIMDFetcher(options...),
			Cache:                    store,
			Materializer:             store,
			Policy:                   policy,
			MaxConcurrentDiscoveries: 10,
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
	return r.resolver.ResolveClient(ctx, clientID, registeredOnly)
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
		return model.OidcClient{}, err
	}
	pocketIDClient, ok := client.(Client)
	if !ok {
		return model.OidcClient{}, errors.New("metadata resolver returned an incompatible client")
	}
	return pocketIDClient.OidcClient, nil
}

type cimdPolicy struct {
	getURLAllowlist func() []string
}

func (p cimdPolicy) cimdURLAllowed(id string) bool {
	if p.getURLAllowlist == nil {
		return false
	}
	return utils.MatchesAnyURLPattern(p.getURLAllowlist(), id)
}

// AllowCIMDClient applies Pocket ID's operator-managed dynamic-client allowlist
func (p cimdPolicy) AllowCIMDClient(_ context.Context, id string) error {
	if !p.cimdURLAllowed(id) {
		return errors.New("client ID is not in the metadata document allowlist")
	}
	return nil
}

// ValidateCIMDClient restricts generic CIMD features to those supported by Pocket ID's client model
func (cimdPolicy) ValidateCIMDClient(_ context.Context, doc *fosite.ClientMetadataDocument) error {
	// Require public-client authentication because Pocket ID does not persist CIMD key material
	switch doc.TokenEndpointAuthMethod {
	case "none":
	default:
		return fmt.Errorf("client metadata documents only support token_endpoint_auth_method %q, got %q", "none", doc.TokenEndpointAuthMethod)
	}

	// Restrict metadata clients to grant types implemented by Pocket ID and require a flow that can initiate authorization
	grantTypes := doc.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{string(fosite.GrantTypeAuthorizationCode)}
	}
	hasInitiatingGrant := false
	for _, grantType := range grantTypes {
		switch grantType {
		case string(fosite.GrantTypeAuthorizationCode), string(fosite.GrantTypeDeviceCode):
			hasInitiatingGrant = true
		case string(fosite.GrantTypeRefreshToken):
		default:
			return fmt.Errorf("client metadata document contains unsupported grant_type %q", grantType)
		}
	}
	if !hasInitiatingGrant {
		return errors.New("client metadata document must enable authorization_code or device_code")
	}

	// Pocket ID only implements the code response type for metadata clients
	responseTypes := doc.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	for _, responseType := range responseTypes {
		if responseType != "code" {
			return fmt.Errorf("client metadata document contains unsupported response_type %q", responseType)
		}
	}

	return nil
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

	// Record what the document says the client restricts itself to, so it is not silently granted capabilities it never declared
	// RFC 7591 section 2 defaults an omitted grant_types to authorization_code
	grantTypes := doc.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	client := model.OidcClient{
		Base:                        model.Base{ID: rawURL},
		Name:                        doc.ClientName,
		CallbackURLs:                datatype.StringList(doc.RedirectURIs),
		LogoutCallbackURLs:          datatype.StringList(doc.PostLogoutRedirectURIs),
		ClientType:                  model.OidcClientTypeCIMD,
		MetadataGrantTypes:          datatype.StringList(grantTypes),
		AccessTokenDurationSeconds:  model.DefaultAccessTokenDurationSeconds,
		RefreshTokenDurationSeconds: model.DefaultRefreshTokenDurationSeconds,
	}

	switch doc.TokenEndpointAuthMethod {
	case "none":
		client.IsPublic = true
		client.PkceEnabled = true
	default:
		return model.OidcClient{}, fmt.Errorf("client metadata documents only support token_endpoint_auth_method %q, got %q", "none", doc.TokenEndpointAuthMethod)
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
