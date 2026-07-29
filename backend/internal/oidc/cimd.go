package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ory/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type cimdResolverConfig struct {
	httpClient         *http.Client
	getURLAllowlist    func() []string
	transportDecorator func(http.RoundTripper) http.RoundTripper
}

type cimdClientResolver struct {
	resolver *fosite.CIMDResolver
	store    *Store
	policy   cimdPolicy
}

var _ fosite.ClientResolver = (*cimdClientResolver)(nil)
var _ fosite.CIMDClientPolicy = (cimdPolicy{})

func newCIMDClientResolver(store *Store, config cimdResolverConfig) *cimdClientResolver {
	options := []fosite.CIMDFetcherOption{
		fosite.WithCIMDUserAgent("pocket-id/oidc-client-metadata-fetcher"),
		fosite.WithCIMDExtraPrivateRanges(utils.LocalIPv6IPNets()),
	}
	if config.transportDecorator != nil {
		options = append(options, fosite.WithCIMDTransportDecorator(config.transportDecorator))
		if config.httpClient != nil {
			if transport, ok := config.httpClient.Transport.(*http.Transport); ok {
				options = append(options, fosite.WithCIMDTransport(transport))
			}
		}
	} else if config.httpClient != nil {
		options = append(options, fosite.WithCIMDTransport(config.httpClient.Transport))
	}

	policy := cimdPolicy{getURLAllowlist: config.getURLAllowlist}
	return &cimdClientResolver{
		resolver: &fosite.CIMDResolver{
			Fetcher:      fosite.NewDefaultCIMDFetcher(options...),
			Cache:        store,
			Materializer: store,
			Policy:       policy,
		},
		store:  store,
		policy: policy,
	}
}

func (r *cimdClientResolver) ResolveClient(ctx context.Context, clientID string, next fosite.ClientLookupFunc) (fosite.Client, error) {
	return r.resolver.ResolveClient(ctx, clientID, next)
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
	switch doc.TokenEndpointAuthMethod {
	case "", "none":
		return nil
	default:
		return fmt.Errorf("client metadata documents only support token_endpoint_auth_method %q, got %q", "none", doc.TokenEndpointAuthMethod)
	}
}

// buildClientFromMetadata applies Pocket ID's persisted-client projection to validated generic metadata
func buildClientFromMetadata(doc *fosite.ClientMetadataDocument, rawURL string) (model.OidcClient, error) {
	client := model.OidcClient{
		Base:               model.Base{ID: rawURL},
		Name:               doc.ClientName,
		CallbackURLs:       model.UrlList(doc.RedirectURIs),
		LogoutCallbackURLs: model.UrlList(doc.PostLogoutRedirectURIs),
		ClientType:         model.OidcClientTypeCIMD,
	}

	switch doc.TokenEndpointAuthMethod {
	case "", "none":
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
