package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// This file implements the OAuth Client ID Metadata Document feature
// (draft-ietf-oauth-client-id-metadata-document), part of the MCP client
// registration spec. When enabled, a client_id that is an https URL is resolved
// by fetching and validating the metadata document it points to, synthesizing an
// OidcClient that is cached in the database.

// looksLikeClientIDMetadataDocumentURL reports whether clientID looks like a
// CIMD URL. Detection is intentionally cheap and inprecise.
func looksLikeClientIDMetadataDocumentURL(clientID string) bool {
	u, err := url.Parse(clientID)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Host != ""
}

// validateClientIDMetadataDocumentURL enforces the client_id URL rules from
// draft-ietf-oauth-client-id-metadata-document §2.
func validateClientIDMetadataDocumentURL(u *url.URL) error {
	if u.Scheme != "https" {
		return errors.New("client_id URL must use the https scheme")
	}
	if u.Host == "" {
		return errors.New("client_id URL must have a host")
	}
	if u.User != nil {
		return errors.New("client_id URL must not contain userinfo")
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return errors.New("client_id URL must not contain a fragment")
	}
	if u.RawQuery != "" {
		return errors.New("client_id URL must not contain a query string")
	}
	if u.Path == "" || u.Path == "/" {
		return errors.New("client_id URL must contain a path component")
	}
	// url.Parse preserves dot segments in Path (it does not clean them), so this catches them.
	for seg := range strings.SplitSeq(u.Path, "/") {
		if seg == "." || seg == ".." {
			return fmt.Errorf("client_id URL must not contain %q path segments", seg)
		}
	}
	return nil
}

// clientMetadataDocument is the subset of RFC 7591 client metadata that pocket-id
// consumes. Unknown properties are ignored (the draft allows additional members).
type clientMetadataDocument struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	JwksURI                 string   `json:"jwks_uri"`
	LogoURI                 string   `json:"logo_uri"`

	// Presence of these is forbidden by the draft; pointers detect presence.
	ClientSecret          *string `json:"client_secret"`
	ClientSecretExpiresAt *int64  `json:"client_secret_expires_at"`
}

// buildClientFromMetadata validates a fetched metadata document against its URL
// and maps it to an OidcClient. It does not touch the database.
func buildClientFromMetadata(doc *clientMetadataDocument, rawURL string) (model.OidcClient, error) {
	if doc.ClientID != rawURL {
		return model.OidcClient{}, fmt.Errorf("client_id %q does not match document URL %q", doc.ClientID, rawURL)
	}
	if doc.ClientSecret != nil || doc.ClientSecretExpiresAt != nil {
		return model.OidcClient{}, errors.New("client metadata document must not contain a client secret")
	}

	client := model.OidcClient{
		Base:               model.Base{ID: rawURL},
		Name:               doc.ClientName,
		CallbackURLs:       model.UrlList(doc.RedirectURIs),
		LogoutCallbackURLs: model.UrlList(doc.PostLogoutRedirectURIs),
		IsMetadataDocument: true,
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

const (
	clientMetadataMaxSize      = 5 * 1024 // 5 KiB (draft recommendation)
	clientMetadataFetchTimeout = 15 * time.Second
	clientMetadataMinCacheTTL  = 5 * time.Minute
	clientMetadataMaxCacheTTL  = 24 * time.Hour
	clientMetadataDefaultTTL   = 1 * time.Hour
)

// resolveMetadataClient resolves a CIMD client_id (an https URL) to an OidcClient,
// fetching and caching the metadata document as needed. The caller must ensure id
// looks like a CIMD URL.
func (s *Store) resolveMetadataClient(ctx context.Context, id string) (model.OidcClient, error) {
	existing, err := s.firstClientByID(ctx, id)
	found := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.OidcClient{}, err
	}
	if found && existing.MetadataExpiresAt != nil && time.Time(*existing.MetadataExpiresAt).After(time.Now()) {
		return existing, nil
	}

	doc, ttl, err := s.fetchClientIDMetadataDocument(ctx, id)
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

// fetchClientIDMetadataDocument retrieves and parses the metadata document at
// rawURL. It blocks private URLs (SSRF), refuses redirects, caps the body, and
// requires a 200 response. The returned duration is the cache lifetime derived
// from the response headers.
func (s *Store) fetchClientIDMetadataDocument(ctx context.Context, rawURL string) (*clientMetadataDocument, time.Duration, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	if err := validateClientIDMetadataDocumentURL(u); err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, clientMetadataFetchTimeout)
	defer cancel()

	ok, err := utils.IsURLPrivate(ctx, u)
	if err != nil {
		return nil, 0, err
	} else if ok {
		return nil, 0, errors.New("private IP addresses are not allowed")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pocket-id/oidc-client-metadata-fetcher")

	resp, err := s.metadataHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status %d fetching client metadata", resp.StatusCode)
	}

	body, err := io.ReadAll(utils.NewLimitReader(resp.Body, clientMetadataMaxSize+1))
	if errors.Is(err, utils.ErrSizeExceeded) {
		return nil, 0, errors.New("client metadata document exceeds maximum size")
	} else if err != nil {
		return nil, 0, err
	}

	var doc clientMetadataDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, 0, fmt.Errorf("invalid client metadata JSON: %w", err)
	}

	return &doc, parseClientMetadataCacheTTL(resp.Header.Get("Cache-Control")), nil
}

func (s *Store) metadataHTTPClient() *http.Client {
	var transport http.RoundTripper
	if s.httpClient != nil {
		transport = s.httpClient.Transport
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirects are not allowed when fetching client metadata")
		}}
}

func parseClientMetadataCacheTTL(cacheControl string) time.Duration {
	for part := range strings.SplitSeq(cacheControl, ",") {
		v, ok := strings.CutPrefix(strings.TrimSpace(part), "max-age=")
		if !ok {
			continue
		}
		secs, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		d := time.Duration(secs) * time.Second
		switch {
		case d < clientMetadataMinCacheTTL:
			return clientMetadataMinCacheTTL
		case d > clientMetadataMaxCacheTTL:
			return clientMetadataMaxCacheTTL
		default:
			return d
		}
	}
	return clientMetadataDefaultTTL
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
			"IsPublic", "PkceEnabled", "IsMetadataDocument", "MetadataExpiresAt").
		Updates(client).Error
}
