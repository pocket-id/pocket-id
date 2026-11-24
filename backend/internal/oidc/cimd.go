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
	"sync/atomic"
	"time"

	"github.com/pocket-id/fosite"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
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
	resolver           *fosite.CIMDResolver
	store              *Store
	policy             cimdPolicy
	onBeforeMismatchTx func()
}

var _ fosite.ClientResolver = (*cimdClientResolver)(nil)
var _ fosite.CIMDClientPolicy = cimdPolicy{}

type mismatchRevokingFetcher struct {
	*fosite.DefaultCIMDFetcher
	onMismatch func(ctx context.Context, clientID string, err error) error
}

func (f mismatchRevokingFetcher) Fetch(ctx context.Context, clientID string) (*fosite.ClientMetadataDocument, fosite.CIMDCachePolicy, error) {
	doc, policy, err := f.DefaultCIMDFetcher.Fetch(ctx, clientID)
	if errors.Is(err, fosite.ErrCIMDClientIDMismatch) && f.onMismatch != nil {
		// Trigger revocation where the mismatch is produced so caller cancellation cannot mask cleanup
		err = f.onMismatch(ctx, clientID, err)
	}
	return doc, policy, err
}

const cimdStaleIfUnavailable = 24 * time.Hour

func (r *cimdClientResolver) now() time.Time {
	if r.resolver != nil && r.resolver.Now != nil {
		return r.resolver.Now()
	}
	return time.Now()
}

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

	policy := cimdPolicy{getURLAllowlist: config.getURLAllowlist, apiAccess: store.apiAccess, db: store.db}
	r := &cimdClientResolver{
		store:  store,
		policy: policy,
	}
	r.resolver = &fosite.CIMDResolver{
		Fetcher: mismatchRevokingFetcher{
			DefaultCIMDFetcher: fosite.NewDefaultCIMDFetcher(options...),
			onMismatch:         r.revokeOnClientIDMismatch,
		},
		Cache:                    store,
		Materializer:             store,
		Policy:                   policy,
		MaxConcurrentDiscoveries: 10,
		// An outage of a client's host must not lock out its users, so the last document stays in use for a day
		StaleIfUnavailable: cimdStaleIfUnavailable,
		// A client ID that failed is left alone as long as a fetched document is kept at least
		RetryAfterFailure: time.Minute,
	}
	return r
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
	if err != nil {
		if errors.Is(err, fosite.ErrCIMDClientIDMismatch) {
			return nil, r.revokeOnClientIDMismatch(ctx, clientID, err)
		}
		// If discovery failed only because the host was temporarily unavailable while another request committed fresh metadata, return that fresh persisted client instead of failing
		if errors.Is(err, fosite.ErrCIMDTemporarilyUnavailable) && r.store != nil && r.store.db != nil && fosite.LooksLikeCIMDURL(clientID) && !r.store.isCIMDInvalidatedForContext(ctx, clientID) {
			if policyErr := r.policy.AllowCIMDClient(ctx, clientID); policyErr == nil {
				if freshModel, freshClient, ok, loadErr := r.store.loadConsistentCIMDClient(ctx, clientID); loadErr == nil && ok &&
					!freshModel.Disabled &&
					isCIMDClientCompatible(freshModel) &&
					freshModel.MetadataExpiresAt != nil && freshModel.MetadataExpiresAt.ToTime().After(r.now()) {
					return freshClient, nil
				}
			}
		}
		return nil, err
	}
	// Re-read the persisted CIMD client after discovery under one consistent snapshot and re-verify policy and eligibility so shared discoveries and stale-on-outage returns neither bypass allowlist changes nor return invalidated/superseded/ineligible metadata
	if pocketClient, ok := client.(Client); ok && pocketClient.IsMetadataDocument() && r.store != nil && r.store.db != nil {
		if r.store.isCIMDInvalidatedForContext(ctx, clientID) {
			return nil, fosite.ErrInvalidClient.WithHint("The client metadata document is no longer valid.")
		}
		if policyErr := r.policy.AllowCIMDClient(ctx, clientID); policyErr != nil {
			return nil, fosite.ErrInvalidClient.WithHint("The client metadata document could not be resolved.").WithWrap(policyErr).WithDebug(policyErr.Error())
		}
		current, refreshedClient, found, loadErr := r.store.loadConsistentCIMDClient(ctx, clientID)
		if loadErr != nil {
			return nil, loadErr
		}
		if !found {
			return nil, fosite.ErrInvalidClient.WithHint("The client is no longer valid.")
		}
		if current.Disabled {
			return nil, apperror.OidcClientDisabled()
		}
		if !isCIMDMetadataActive(current, r.now()) {
			return nil, fosite.ErrInvalidClient.WithHint("The client metadata document is no longer valid.")
		}
		return refreshedClient, nil
	}
	return client, nil
}

type mismatchCleanupState struct {
	done      atomic.Bool
	repairGen uint64
}

type revokedMismatchError struct {
	err     error
	state   *mismatchCleanupState
	txHooks *txHooks
}

func (e revokedMismatchError) Error() string { return e.err.Error() }
func (e revokedMismatchError) Unwrap() error { return e.err }

// revokeOnClientIDMismatch revokes what users granted to a client whose document no longer claims its client ID
// The lookup error is returned either way, so the request still fails as an invalid client
func (r *cimdClientResolver) revokeOnClientIDMismatch(ctx context.Context, clientID string, err error) error {
	return r.revokeOnClientIDMismatchExpectedGen(ctx, clientID, err, 0)
}

func (r *cimdClientResolver) revokeOnClientIDMismatchExpectedGen(ctx context.Context, clientID string, err error, expectedGen uint64) error {
	if !errors.Is(err, fosite.ErrCIMDClientIDMismatch) {
		return err
	}
	// Reuse any shared cleanup state attached to a cached Fosite failure so a later successful retry or intervening repair stops repeated cleanup
	var tracked revokedMismatchError
	var state *mismatchCleanupState
	if errors.As(err, &tracked) && tracked.state != nil {
		state = tracked.state
		if state.done.Load() {
			return err
		}
		if r.store.cimdRepairedSince(clientID, state.repairGen) {
			state.done.Store(true)
			return err
		}
		if callerHooks := txHooksFromContext(ctx); callerHooks != nil && tracked.txHooks == callerHooks {
			return err
		}
	} else {
		// Capture the repair epoch when the mismatch is observed so a concurrent repair committing before failure publication suppresses stale invalidation
		state = &mismatchCleanupState{repairGen: r.store.cimdRepairGen(clientID)}
	}
	if r.onBeforeMismatchTx != nil {
		r.onBeforeMismatchTx()
	}

	_, inCallerTx := txFromContext(ctx)
	cleanupCtx := ctx
	var cancel context.CancelFunc
	if !inCallerTx {
		// Detach mismatch cleanup from HTTP request cancellation so a canceled client cannot abort revocation or cache invalidation
		cleanupCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
	}

	var revoked int64
	var skipped bool
	revokeErr := withTx(cleanupCtx, r.store.db, func(txCtx context.Context) error {
		if err := lockCIMDClientID(r.store.dbFor(txCtx), clientID); err != nil {
			return err
		}
		pendingCh, failedGen, err := r.store.acquireCIMDPendingClear(txCtx, clientID)
		if err != nil {
			return err
		}
		handedOff := false
		defer func() {
			if !handedOff {
				r.store.abortCIMDRepair(clientID, pendingCh)
			}
		}()
		// Recheck shared completion and repair epoch under the client lock so concurrent replays cannot repeat cleanup or clobber an intervening repair
		if state.done.Load() {
			skipped = true
			return nil
		}
		if r.store.cimdRepairedSince(clientID, state.repairGen) {
			state.done.Store(true)
			skipped = true
			return nil
		}
		// For a deferred post-tx retry, if another transaction already repaired and cleared this mismatch generation (or a newer mismatch superseded it), skip only this obsolete callback without cancelling a newer retry on the same shared state
		if expectedGen != 0 && failedGen != expectedGen {
			skipped = true
			return nil
		}
		registerDone := func() {
			if !inCallerTx {
				handedOff = true
				clearGen := failedGen
				afterTxFinish(txCtx, func(committed bool) {
					if committed {
						state.done.Store(true)
					}
					r.store.finishCIMDClear(clientID, clearGen, pendingCh, committed, false)
				})
				return
			}
			// Inside a caller-owned transaction the caller may still roll back: mark failed in memory until commit and retry on rollback
			mismatchGen, repaired, _ := r.store.markCIMDInvalidationFailedIfUnrepaired(txCtx, clientID, state.repairGen)
			if repaired {
				state.done.Store(true)
				skipped = true
				return
			}
			if hasTxHooks(txCtx) {
				cancelPostTxOnCommit(txCtx, "cimd-mismatch:"+clientID)
				handedOff = true
				afterTxFinish(txCtx, func(committed bool) {
					if committed {
						state.done.Store(true)
					}
					r.store.finishCIMDClear(clientID, mismatchGen, pendingCh, committed, false)
				})
				retryErr := revokedMismatchError{err: fosite.ErrCIMDClientIDMismatch, state: state}
				afterTxRollbackOnce(txCtx, "cimd-mismatch:"+clientID, func(rollbackCtx context.Context) {
					_ = r.revokeOnClientIDMismatchExpectedGen(rollbackCtx, clientID, retryErr, mismatchGen)
				})
			}
		}
		var current model.OidcClient
		lockErr := withClientNoKeyUpdateLock(r.store.dbFor(txCtx)).Where("id = ?", clientID).First(&current).Error
		if errors.Is(lockErr, gorm.ErrRecordNotFound) {
			registerDone()
			return nil
		}
		if lockErr != nil {
			return lockErr
		}
		if !current.IsMetadataDocument() {
			registerDone()
			return nil
		}
		var rErr error
		revoked, rErr = r.store.revokeClientGrants(txCtx, clientID)
		if rErr != nil {
			return rErr
		}
		// Invalidate the cached metadata entry so the rejected document cannot be served fresh or via stale-on-outage fallback
		if err := r.store.invalidateCIMDClientCache(txCtx, clientID); err != nil {
			return err
		}
		registerDone()
		return nil
	})
	if revokeErr != nil {
		if !inCallerTx {
			// Best-effort persisted cache invalidation on its own fresh bounded context so a grant-deletion error or timeout does not leave fresh cached metadata usable across restarts
			fallbackCtx, fallbackCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer fallbackCancel()
			var fallbackSkipped bool
			if fallbackErr := withTx(fallbackCtx, r.store.db, func(invCtx context.Context) error {
				if err := lockCIMDClientID(r.store.dbFor(invCtx), clientID); err != nil {
					return err
				}
				pendingCh, fallbackGen, err := r.store.acquireCIMDPendingClear(invCtx, clientID)
				if err != nil {
					return err
				}
				handedOff := false
				defer func() {
					if !handedOff {
						r.store.abortCIMDRepair(clientID, pendingCh)
					}
				}()
				if state.done.Load() {
					fallbackSkipped = true
					return nil
				}
				if r.store.cimdRepairedSince(clientID, state.repairGen) {
					state.done.Store(true)
					fallbackSkipped = true
					return nil
				}
				if expectedGen != 0 && fallbackGen != expectedGen {
					fallbackSkipped = true
					return nil
				}
				var current model.OidcClient
				lockErr := withClientNoKeyUpdateLock(r.store.dbFor(invCtx)).Where("id = ?", clientID).First(&current).Error
				if errors.Is(lockErr, gorm.ErrRecordNotFound) {
					handedOff = true
					afterTxFinish(invCtx, func(committed bool) {
						if committed {
							state.done.Store(true)
						}
						r.store.finishCIMDClear(clientID, fallbackGen, pendingCh, committed, false)
					})
					return nil
				}
				if lockErr != nil {
					return lockErr
				}
				if !current.IsMetadataDocument() {
					handedOff = true
					afterTxFinish(invCtx, func(committed bool) {
						if committed {
							state.done.Store(true)
						}
						r.store.finishCIMDClear(clientID, fallbackGen, pendingCh, committed, false)
					})
					return nil
				}
				if err := r.store.invalidateCIMDClientCache(invCtx, clientID); err != nil {
					return err
				}
				// Persisted metadata_expires_at = NULL will be durable on commit; clear fallbackGen before unblocking waiters
				handedOff = true
				afterTxFinish(invCtx, func(committed bool) {
					r.store.finishCIMDClear(clientID, fallbackGen, pendingCh, committed, false)
				})
				return nil
			}); fallbackErr != nil {
				// Both revocation and fallback DB invalidation failed; suppress cached expiry in memory so Fosite cannot take the fresh-cache fast path before retrying discovery.
				// Preserve the current store generation when retrying an older deferred mismatch (expectedGen != 0) or when an intervening repair has already committed.
				if expectedGen == 0 {
					waitCtx, waitCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
					_, repaired := r.store.markCIMDInvalidationFailedAfterPending(waitCtx, clientID, state.repairGen)
					waitCancel()
					if repaired {
						state.done.Store(true)
						return revokedMismatchError{err: err, state: state}
					}
				}
				slog.ErrorContext(fallbackCtx, "Failed fallback CIMD metadata cache invalidation after revocation error",
					slog.String("client_id", clientID),
					slog.Any("error", fallbackErr),
				)
				revokeErr = errors.Join(revokeErr, fallbackErr)
			} else if fallbackSkipped {
				return revokedMismatchError{err: err, state: state}
			}
		} else {
			mismatchGen, repaired, _ := r.store.markCIMDInvalidationFailedIfUnrepaired(ctx, clientID, state.repairGen)
			if repaired {
				state.done.Store(true)
				return revokedMismatchError{err: err, state: state}
			}
			// Retry durable cleanup after the enclosing transaction settles (commit or rollback), verifying under the client lock that mismatchGen/state.repairGen was not already repaired by another transaction
			retryErr := revokedMismatchError{err: fosite.ErrCIMDClientIDMismatch, state: state}
			afterTxSettledOnce(ctx, "cimd-mismatch:"+clientID, func(settledCtx context.Context) {
				_ = r.revokeOnClientIDMismatchExpectedGen(settledCtx, clientID, retryErr, mismatchGen)
			})
		}
		// Keep state.done false so later lookups sharing this cached error can retry revocation and mark the shared state done once it succeeds
		return revokedMismatchError{err: errors.Join(err, revokeErr), state: state, txHooks: txHooksFromContext(ctx)}
	}
	if skipped {
		return revokedMismatchError{err: err, state: state}
	}

	// The mismatch is reported on every request for the client ID, also for one that never was a client
	if revoked > 0 {
		slog.WarnContext(cleanupCtx, "Revoked the user consent and tokens of a client whose metadata document no longer matches its client ID",
			slog.String("client_id", clientID),
		)
	}
	return revokedMismatchError{err: err, state: state, txHooks: txHooksFromContext(ctx)}
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
	db              *gorm.DB
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
		var tx *gorm.DB
		if p.db != nil {
			tx = dbFromContext(ctx, p.db)
		}
		_, apiExists, _, err := p.apiAccess.AllowedScopesForAudience(ctx, tx, id, canonicalResource(id), SubjectTypeUser)
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
