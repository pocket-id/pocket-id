package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ory/fosite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// buildClientFromMetadata maps a validated client metadata document to an
// OidcClient. The document is expected to have already been validated against its
// URL by the fetcher (see fosite.ClientMetadataDocument.Validate); this function
// only performs the pocket-id-specific mapping.
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
	case "private_key_jwt":
		if doc.JwksURI == "" {
			return model.OidcClient{}, errors.New("private_key_jwt requires jwks_uri")
		}
		client.Credentials = model.OidcClientCredentials{
			FederatedIdentities: []model.OidcClientFederatedIdentity{{
				Issuer:  rawURL,
				Subject: rawURL,
				JWKS:    doc.JwksURI,
			}},
		}
	default:
		return model.OidcClient{}, fmt.Errorf("unsupported token_endpoint_auth_method %q", doc.TokenEndpointAuthMethod)
	}

	if client.Name == "" {
		if u, err := url.Parse(rawURL); err == nil {
			client.Name = u.Host
		}
	}

	return client, nil
}

// RefreshMetadataClient forces a re-fetch of the metadata document for an
// already-cached CIMD client, bypassing the cache TTL. It returns an error if
// CIMD is disabled, the id is not a CIMD URL, or no metadata-document client with
// that id exists.
func (s *Store) RefreshMetadataClient(ctx context.Context, id string) (model.OidcClient, error) {
	if !s.cimdEnabled {
		return model.OidcClient{}, errors.New("client ID metadata documents are not enabled")
	}
	if !fosite.LooksLikeCIMDURL(id) {
		return model.OidcClient{}, errors.New("client is not a client ID metadata document client")
	}
	if !s.cimdURLAllowed(id) {
		return model.OidcClient{}, errors.New("client ID is not in the metadata document allowlist")
	}
	existing, err := s.firstClientByID(ctx, id)
	if err != nil {
		return model.OidcClient{}, err
	}
	if !existing.IsMetadataDocument() {
		return model.OidcClient{}, errors.New("client is not a client ID metadata document client")
	}
	return s.resolveMetadataClient(ctx, id, true)
}

// resolveMetadataClient resolves a CIMD client_id (an https URL) to an OidcClient,
// fetching and caching the metadata document as needed.
func (s *Store) resolveMetadataClient(ctx context.Context, id string, force bool) (model.OidcClient, error) {
	existing, err := s.firstClientByID(ctx, id)
	found := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.OidcClient{}, err
	}
	if !force && found && existing.MetadataExpiresAt != nil && time.Time(*existing.MetadataExpiresAt).After(time.Now()) {
		return existing, nil
	}

	doc, ttl, err := s.cimdFetcher().Fetch(ctx, id)
	if err != nil {
		return model.OidcClient{}, fmt.Errorf("failed to fetch client metadata document: %w", err)
	}
	client, err := buildClientFromMetadata(doc, id)
	if err != nil {
		return model.OidcClient{}, fmt.Errorf("invalid client metadata document: %w", err)
	}
	expiresAt := datatype.DateTime(time.Now().Add(ttl))
	client.MetadataExpiresAt = &expiresAt

	db := s.dbFor(ctx)
	if found {
		if changes := metadataClientChanges(existing, client); len(changes) > 0 && s.auditLog != nil {
			s.auditLog.Create(ctx, model.AuditLogEventClientMetadataChanged, "", "", "", model.AuditLogData{
				"clientID":      id,
				"changedFields": strings.Join(changes, ","),
			}, db)
		}
	}
	if err := s.upsertMetadataClient(ctx, db, &client, found); err != nil {
		return model.OidcClient{}, err
	}

	// Reload so DB-managed columns and preloads (e.g. AllowedUserGroups) are populated.
	return s.firstClientByID(ctx, id)
}

// metadataClientChanges returns the names of security-relevant metadata fields
// that differ between the stored client and a freshly fetched one.
func metadataClientChanges(old, next model.OidcClient) []string {
	var changed []string
	if !slices.Equal([]string(old.CallbackURLs), []string(next.CallbackURLs)) {
		changed = append(changed, "redirect_uris")
	}
	if !slices.Equal([]string(old.LogoutCallbackURLs), []string(next.LogoutCallbackURLs)) {
		changed = append(changed, "post_logout_redirect_uris")
	}
	if old.IsPublic != next.IsPublic {
		changed = append(changed, "token_endpoint_auth_method")
	}
	if metadataJWKSURL(old) != metadataJWKSURL(next) {
		changed = append(changed, "jwks_uri")
	}
	if old.Name != next.Name {
		changed = append(changed, "client_name")
	}
	return changed
}

// metadataJWKSURL returns the JWKS URL of the client's single synthesized
// federated identity (buildClientFromMetadata always creates at most one).
func metadataJWKSURL(c model.OidcClient) string {
	if len(c.Credentials.FederatedIdentities) == 0 {
		return ""
	}
	return c.Credentials.FederatedIdentities[0].JWKS
}

// upsertMetadataClient inserts a new managed client or updates the metadata-derived
// columns of an existing one, leaving consent, grants, and group links untouched.
func (s *Store) upsertMetadataClient(ctx context.Context, tx *gorm.DB, client *model.OidcClient, update bool) error {
	if !update {
		return tx.WithContext(ctx).
			Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).
			Create(client).Error
	}
	return tx.WithContext(ctx).
		Model(&model.OidcClient{Base: model.Base{ID: client.ID}}).
		Select("Name", "CallbackURLs", "LogoutCallbackURLs", "Credentials",
			"IsPublic", "PkceEnabled", "ClientType", "MetadataExpiresAt").
		Updates(client).Error
}
