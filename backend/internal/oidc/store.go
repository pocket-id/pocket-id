package oidc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/pocket-id/fosite"
	fositeoauth2 "github.com/pocket-id/fosite/handler/oauth2"
	"github.com/pocket-id/fosite/handler/openid"
	"github.com/pocket-id/fosite/handler/pkce"
	"github.com/pocket-id/fosite/handler/rfc8628"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	sessionKindAuthorizeCode = "authorize_code"
	sessionKindAccessToken   = "access_token"
	sessionKindRefreshToken  = "refresh_token"
	sessionKindPKCE          = "pkce"
	sessionKindOpenID        = "openid"
	sessionKindPAR           = "par"
	sessionKindDeviceCode    = "device_code"
	sessionKindUserCode      = "user_code"
	refreshTokenGracePeriod  = time.Minute
)

var (
	_ fosite.Storage                              = (*Store)(nil)
	_ fosite.ClientManager                        = (*Store)(nil)
	_ fosite.PARStorageProvider                   = (*Store)(nil)
	_ fositeoauth2.AuthorizeCodeStorageProvider   = (*Store)(nil)
	_ fositeoauth2.AccessTokenStorageProvider     = (*Store)(nil)
	_ fositeoauth2.RefreshTokenStorageProvider    = (*Store)(nil)
	_ fositeoauth2.TokenRevocationStorageProvider = (*Store)(nil)
	_ rfc8628.DeviceAuthStorageProvider           = (*Store)(nil)
	_ openid.OpenIDConnectRequestStorageProvider  = (*Store)(nil)
	_ pkce.PKCERequestStorageProvider             = (*Store)(nil)
	_ fosite.Transactional                        = (*Store)(nil)
)

// NewStore creates the fosite storage. Exported for packages that need to seed or
// revoke sessions (e.g. the e2e test service).
func NewStore(db *gorm.DB, apiAccess APIAccessProvider) *Store {
	return &Store{db: db, apiAccess: apiAccess}
}

const cimdRepairSeqRetention = time.Hour

type cimdRepairEntry struct {
	seq       uint64
	updatedAt time.Time
}

type cimdPendingClearEntry struct {
	ch                 chan struct{}
	owner              *txHooks
	deferredFailEpochs []uint64
	prev               *cimdPendingClearEntry
}
type Store struct {
	db             *gorm.DB
	apiAccess      APIAccessProvider
	issuer         string
	clientResolver fosite.ClientResolver

	// cimdFailedInvalidation records client IDs whose mismatch revocation and fallback DB invalidation both failed,
	// preventing stale fresh-cache hits until a subsequent cleanup or valid repair commits to the database.
	cimdInvalidationMu     sync.Mutex
	cimdInvalidationSeq    uint64
	cimdFailedInvalidation map[string]uint64
	cimdRepairCounter      uint64
	cimdRepairSeq          map[string]cimdRepairEntry
	cimdPendingClear       map[string]cimdPendingClearEntry
	onWaitPendingCIMDClear func()
}

// acquireCIMDPendingClear waits until no other transaction owns cimdPendingClear[clientID],
// then atomically reserves ownership and reads the current failure generation under the same mutex acquisition.
func (s *Store) acquireCIMDPendingClear(ctx context.Context, clientID string) (chan struct{}, uint64, error) {
	callerHooks := txHooksFromContext(ctx)
	for {
		s.cimdInvalidationMu.Lock()
		entry := s.cimdPendingClear[clientID]
		hook := s.onWaitPendingCIMDClear
		if entry.ch == nil || (callerHooks != nil && entry.owner == callerHooks) {
			ch := make(chan struct{})
			if s.cimdPendingClear == nil {
				s.cimdPendingClear = make(map[string]cimdPendingClearEntry)
			}
			next := cimdPendingClearEntry{
				ch:                 ch,
				owner:              callerHooks,
				deferredFailEpochs: entry.deferredFailEpochs,
			}
			if entry.ch != nil {
				prevCopy := entry
				prevCopy.deferredFailEpochs = nil
				next.prev = &prevCopy
			}
			s.cimdPendingClear[clientID] = next
			failedGen := s.cimdFailedInvalidation[clientID]
			s.cimdInvalidationMu.Unlock()
			return ch, failedGen, nil
		}
		s.cimdInvalidationMu.Unlock()
		if hook != nil {
			hook()
		}
		select {
		case <-entry.ch:
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		}
	}
}

func (s *Store) drainDeferredFailEpochsLocked(clientID string, epochs []uint64) {
	if s.cimdFailedInvalidation[clientID] != 0 || len(epochs) == 0 {
		return
	}
	for _, epoch := range epochs {
		if !s.cimdRepairedSinceLocked(clientID, epoch) {
			if s.cimdFailedInvalidation == nil {
				s.cimdFailedInvalidation = make(map[string]uint64)
			}
			s.cimdInvalidationSeq++
			s.cimdFailedInvalidation[clientID] = s.cimdInvalidationSeq
			break
		}
	}
}

func (s *Store) abortCIMDRepair(clientID string, ch chan struct{}) {
	s.cimdInvalidationMu.Lock()
	if entry := s.cimdPendingClear[clientID]; entry.ch == ch {
		if entry.prev != nil {
			restored := *entry.prev
			restored.deferredFailEpochs = entry.deferredFailEpochs
			s.cimdPendingClear[clientID] = restored
		} else {
			delete(s.cimdPendingClear, clientID)
			s.drainDeferredFailEpochsLocked(clientID, entry.deferredFailEpochs)
		}
	}
	s.cimdInvalidationMu.Unlock()
	close(ch)
}

func (s *Store) pruneCIMDRepairSeqLocked(now time.Time) {
	for id, entry := range s.cimdRepairSeq {
		if now.Sub(entry.updatedAt) <= cimdRepairSeqRetention {
			continue
		}
		if _, failed := s.cimdFailedInvalidation[id]; failed {
			continue
		}
		if _, pending := s.cimdPendingClear[id]; pending {
			continue
		}
		delete(s.cimdRepairSeq, id)
	}
}

func (s *Store) finishCIMDClear(clientID string, gen uint64, ch chan struct{}, committed bool, bumpRepairGen bool) {
	s.cimdInvalidationMu.Lock()
	if committed {
		if gen != 0 && s.cimdFailedInvalidation[clientID] == gen {
			delete(s.cimdFailedInvalidation, clientID)
		}
		if bumpRepairGen && s.cimdFailedInvalidation[clientID] == 0 {
			if s.cimdRepairSeq == nil {
				s.cimdRepairSeq = make(map[string]cimdRepairEntry)
			}
			if s.cimdRepairCounter == 0 {
				s.cimdRepairCounter = 1
			}
			s.cimdRepairCounter++
			now := time.Now()
			s.cimdRepairSeq[clientID] = cimdRepairEntry{seq: s.cimdRepairCounter, updatedAt: now}
			s.pruneCIMDRepairSeqLocked(now)
		}
	}
	if entry := s.cimdPendingClear[clientID]; entry.ch == ch {
		delete(s.cimdPendingClear, clientID)
		s.drainDeferredFailEpochsLocked(clientID, entry.deferredFailEpochs)
	}
	s.cimdInvalidationMu.Unlock()
	close(ch)
}

func (s *Store) cimdRepairGenLocked(clientID string) uint64 {
	if entry, ok := s.cimdRepairSeq[clientID]; ok && entry.seq != 0 {
		return entry.seq
	}
	return 1
}

// cimdRepairedSinceLocked reports whether a repair committed for clientID after observedRepairGen.
// Comparing strictly greater (>) keeps outstanding observations valid even if a stale entry is pruned back to 1,
// because cimdRepairCounter is strictly monotonic across the lifetime of the Store.
func (s *Store) cimdRepairedSinceLocked(clientID string, observedRepairGen uint64) bool {
	return observedRepairGen != 0 && s.cimdRepairGenLocked(clientID) > observedRepairGen
}

func (s *Store) cimdRepairedSince(clientID string, observedRepairGen uint64) bool {
	s.cimdInvalidationMu.Lock()
	defer s.cimdInvalidationMu.Unlock()
	return s.cimdRepairedSinceLocked(clientID, observedRepairGen)
}

func (s *Store) cimdRepairGen(clientID string) uint64 {
	s.cimdInvalidationMu.Lock()
	defer s.cimdInvalidationMu.Unlock()
	return s.cimdRepairGenLocked(clientID)
}

func (s *Store) markCIMDInvalidationFailed(clientID string) uint64 {
	gen, _, _ := s.markCIMDInvalidationFailedIfUnrepaired(context.Background(), clientID, 0)
	return gen
}

func (s *Store) markCIMDInvalidationFailedIfUnrepaired(ctx context.Context, clientID string, expectedRepairGen uint64) (uint64, bool, bool) {
	s.cimdInvalidationMu.Lock()
	defer s.cimdInvalidationMu.Unlock()
	if s.cimdRepairedSinceLocked(clientID, expectedRepairGen) {
		return 0, true, false
	}
	callerHooks := txHooksFromContext(ctx)
	if current := s.cimdPendingClear[clientID]; current.ch != nil && (callerHooks == nil || current.owner != callerHooks) {
		// Another transaction has a pending repair/clear: defer publication so a committing repair supersedes it while a rolled-back repair publishes it
		current.deferredFailEpochs = append(current.deferredFailEpochs, expectedRepairGen)
		s.cimdPendingClear[clientID] = current
		return 0, false, true
	}
	if s.cimdFailedInvalidation == nil {
		s.cimdFailedInvalidation = make(map[string]uint64)
	}
	s.cimdInvalidationSeq++
	s.cimdFailedInvalidation[clientID] = s.cimdInvalidationSeq
	return s.cimdInvalidationSeq, false, false
}

// markCIMDInvalidationFailedAfterPending atomically checks pending repairs, compares expectedRepairGen, and publishes a failure marker under one mutex boundary.
// If ctx expires while another transaction's clear/repair is still pending, publication is deferred to finishCIMDClear so a committing repair supersedes it while a rolled-back repair still publishes the marker.
func (s *Store) markCIMDInvalidationFailedAfterPending(ctx context.Context, clientID string, expectedRepairGen uint64) (uint64, bool) {
	callerHooks := txHooksFromContext(ctx)
	for {
		s.cimdInvalidationMu.Lock()
		if s.cimdRepairedSinceLocked(clientID, expectedRepairGen) {
			s.cimdInvalidationMu.Unlock()
			return 0, true
		}
		entry := s.cimdPendingClear[clientID]
		hook := s.onWaitPendingCIMDClear
		if entry.ch == nil || (callerHooks != nil && entry.owner == callerHooks) {
			if s.cimdFailedInvalidation == nil {
				s.cimdFailedInvalidation = make(map[string]uint64)
			}
			s.cimdInvalidationSeq++
			s.cimdFailedInvalidation[clientID] = s.cimdInvalidationSeq
			gen := s.cimdInvalidationSeq
			s.cimdInvalidationMu.Unlock()
			return gen, false
		}
		s.cimdInvalidationMu.Unlock()
		if hook != nil {
			hook()
		}
		select {
		case <-entry.ch:
		case <-ctx.Done():
			s.cimdInvalidationMu.Lock()
			if s.cimdRepairedSinceLocked(clientID, expectedRepairGen) {
				s.cimdInvalidationMu.Unlock()
				return 0, true
			}
			current := s.cimdPendingClear[clientID]
			if current.ch == nil || (callerHooks != nil && current.owner == callerHooks) {
				if s.cimdFailedInvalidation == nil {
					s.cimdFailedInvalidation = make(map[string]uint64)
				}
				s.cimdInvalidationSeq++
				s.cimdFailedInvalidation[clientID] = s.cimdInvalidationSeq
				gen := s.cimdInvalidationSeq
				s.cimdInvalidationMu.Unlock()
				return gen, false
			}
			current.deferredFailEpochs = append(current.deferredFailEpochs, expectedRepairGen)
			s.cimdPendingClear[clientID] = current
			s.cimdInvalidationMu.Unlock()
			return 0, false
		}
	}
}

func (s *Store) cimdInvalidationFailedGen(clientID string) uint64 {
	s.cimdInvalidationMu.Lock()
	defer s.cimdInvalidationMu.Unlock()
	return s.cimdFailedInvalidation[clientID]
}

func (s *Store) isCIMDInvalidated(clientID string) bool {
	return s.cimdInvalidationFailedGen(clientID) != 0
}

func (s *Store) isCIMDInvalidatedForContext(ctx context.Context, clientID string) bool {
	gen := s.cimdInvalidationFailedGen(clientID)
	if gen == 0 {
		return false
	}
	return !txRepairedCIMD(ctx, clientID, gen)
}

// WithIssuer sets the issuer that is added as an extra audience to access tokens carrying an identity scope, so they can be presented to Pocket ID's own endpoints such as /userinfo
// It returns the store to allow chaining at construction
func (s *Store) WithIssuer(issuer string) *Store {
	s.issuer = issuer
	return s
}

type storedRequester struct {
	Authorize bool `json:"authorize,omitempty"`

	ID                string               `json:"id"`
	RequestedAt       time.Time            `json:"requested_at"`
	ClientID          string               `json:"client_id"`
	RequestedScope    fosite.Arguments     `json:"requested_scope,omitempty"`
	GrantedScope      fosite.Arguments     `json:"granted_scope,omitempty"`
	Form              url.Values           `json:"form,omitempty"`
	Session           *Session             `json:"session,omitempty"`
	RequestedAudience fosite.Arguments     `json:"requested_audience,omitempty"`
	GrantedAudience   fosite.Arguments     `json:"granted_audience,omitempty"`
	Device            bool                 `json:"device,omitempty"`
	UserCodeState     fosite.UserCodeState `json:"user_code_state,omitempty"`

	ResponseTypes        fosite.Arguments        `json:"response_types,omitempty"`
	RedirectURI          string                  `json:"redirect_uri,omitempty"`
	State                string                  `json:"state,omitempty"`
	HandledResponseTypes fosite.Arguments        `json:"handled_response_types,omitempty"`
	ResponseMode         fosite.ResponseModeType `json:"response_mode,omitempty"`
	DefaultResponseMode  fosite.ResponseModeType `json:"default_response_mode,omitempty"`
}

// Satisfies fosite.ClientManager

// GetClient is the lookup behind every client Fosite authenticates or restores from a stored session, so it is where a disabled client is refused
// The error is not fosite.ErrNotFound, which keeps the CIMD resolver from fetching the metadata document of a disabled client again
func (s *Store) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	clientModel, err := s.firstClientByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fosite.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if clientModel.Disabled {
		return nil, apperror.OidcClientDisabled()
	}

	client, err := s.clientFromModel(ctx, s.dbFor(ctx), clientModel)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type lockedResolvedClientKey struct{}

func contextWithLockedResolvedClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, lockedResolvedClientKey{}, client)
}

func lockedResolvedClientFromContext(ctx context.Context, id string) (Client, bool) {
	client, ok := ctx.Value(lockedResolvedClientKey{}).(Client)
	if !ok || client.GetID() != id {
		return Client{}, false
	}
	return client, true
}

// resolvePersistedClient restores a client from storage and falls back to the configured generic resolver for uncached clients
func (s *Store) resolvePersistedClient(ctx context.Context, id string) (fosite.Client, error) {
	// Reuse the client already locked and validated for this token exchange so session decoding cannot re-enter metadata discovery while holding the client row lock, while still enforcing the current CIMD policy
	if client, ok := lockedResolvedClientFromContext(ctx, id); ok {
		if resolver, ok := s.clientResolver.(*cimdClientResolver); ok {
			if err := resolver.policy.AllowCIMDClient(ctx, id); err != nil {
				return nil, fosite.ErrInvalidClient.WithHint("The client metadata document could not be resolved.").WithWrap(err).WithDebug(err.Error())
			}
		}
		return client, nil
	}
	if s.clientResolver != nil {
		return s.clientResolver.ResolveClient(ctx, id, s.GetClient)
	}
	return s.GetClient(ctx, id)
}

// clientFromModel populates the provider-specific runtime fields on a stored client
func (s *Store) clientFromModel(ctx context.Context, tx *gorm.DB, clientModel model.OidcClient) (Client, error) {
	client := Client{OidcClient: clientModel}

	// Populate the custom-API scopes and audiences the client may request only when the API feature is wired
	if s.apiAccess != nil {
		apiScopes, apiAudiences, err := s.apiAccess.ClientAPIScopes(ctx, tx, clientModel.ID, clientModel.IsMetadataDocument())
		if err != nil {
			return Client{}, err
		}

		client.apiScopes = apiScopes
		client.apiAudiences = apiAudiences
	}

	customScopes, err := claimmappingpolicy.ClientCustomScopeKeys(ctx, tx, clientModel)
	if err != nil {
		return Client{}, err
	}
	client.customScopes = customScopes

	return client, nil
}

func isCIMDClientCompatible(clientModel model.OidcClient) bool {
	return clientModel.IsPublic && clientModel.PkceEnabled && len(clientModel.Credentials.FederatedIdentities) == 0
}

func isCIMDMetadataActive(clientModel model.OidcClient, now time.Time) bool {
	return clientModel.MetadataExpiresAt != nil &&
		isCIMDClientCompatible(clientModel) &&
		!clientModel.MetadataExpiresAt.ToTime().Add(cimdStaleIfUnavailable).Before(now)
}

func (s *Store) loadCIMDClientOnce(ctx context.Context, tx *gorm.DB, id string) (model.OidcClient, Client, bool, error) {
	clientModel, err := s.firstClientByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.OidcClient{}, Client{}, false, nil
	}
	if err != nil {
		return model.OidcClient{}, Client{}, false, err
	}
	if !clientModel.IsMetadataDocument() {
		return model.OidcClient{}, Client{}, false, nil
	}
	client, err := s.clientFromModel(ctx, tx, clientModel)
	if err != nil {
		return model.OidcClient{}, Client{}, false, err
	}
	return clientModel, client, true, nil
}

// loadConsistentCIMDClient loads a CIMD client row and its runtime scopes/audiences.
// Outside a caller transaction (all production CIMD resolution paths) it opens a single read-only snapshot transaction (REPEATABLE READ on PostgreSQL, ReadOnly on SQLite).
// Inside a caller-owned transaction it reuses the caller's transaction directly (inheriting its isolation level, matching Store.GetClient) so it observes the caller's own uncommitted writes without nesting connections.
func (s *Store) loadConsistentCIMDClient(ctx context.Context, id string) (model.OidcClient, Client, bool, error) {
	if tx, inTx := txFromContext(ctx); inTx {
		return s.loadCIMDClientOnce(ctx, tx, id)
	}

	opts := &sql.TxOptions{ReadOnly: true}
	if s.db.Name() == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}
	var (
		clientModel model.OidcClient
		client      Client
		found       bool
	)
	err := s.db.WithContext(ctx).Transaction(func(roTx *gorm.DB) error {
		roCtx := contextWithTx(ctx, roTx)
		var qErr error
		clientModel, client, found, qErr = s.loadCIMDClientOnce(roCtx, roTx, id)
		return qErr
	}, opts)
	if err != nil {
		return model.OidcClient{}, Client{}, false, err
	}
	return clientModel, client, found, nil
}

// LoadCIMDClient loads only clients that Pocket ID previously associated with a metadata document
func (s *Store) LoadCIMDClient(ctx context.Context, id string) (fosite.CIMDCachedClient, bool, error) {
	clientModel, client, found, err := s.loadConsistentCIMDClient(ctx, id)
	if err != nil || !found {
		return fosite.CIMDCachedClient{}, false, err
	}
	// An invalidated metadata entry (MetadataExpiresAt == nil or failed DB invalidation) receives a zero expiry, which is outside StaleIfUnavailable while still allowing forced manual refresh
	var expiresAt time.Time
	if clientModel.MetadataExpiresAt != nil && !s.isCIMDInvalidatedForContext(ctx, id) {
		expiresAt = time.Time(*clientModel.MetadataExpiresAt)
	}
	// Force incompatible cached entries through discovery so current policy applies before they can be used
	if !isCIMDClientCompatible(clientModel) {
		expiresAt = time.Time{}
	}
	return fosite.CIMDCachedClient{Client: client, ExpiresAt: expiresAt}, true, nil
}

// StoreCIMDClient persists metadata-derived fields while preserving local consent and policy state
func (s *Store) StoreCIMDClient(ctx context.Context, resolved fosite.Client, _ *fosite.ClientMetadataDocument, expiresAt time.Time) (fosite.Client, error) {
	client, ok := resolved.(Client)
	if !ok {
		return nil, errors.New("metadata resolver returned an incompatible client")
	}
	expiry := datatype.DateTime(expiresAt)
	client.MetadataExpiresAt = &expiry

	var changes []string
	var revokeConsent bool
	var stored fosite.Client
	_, inCallerTx := txFromContext(ctx)
	runStore := func(ctx context.Context) error {
		// Serialize first-insert and concurrent mismatch cleanup on PostgreSQL even before an oidc_clients row exists
		if err := lockCIMDClientID(s.dbFor(ctx), client.ID); err != nil {
			return err
		}
		// Atomically wait for any other transaction's pending clear/repair, reserve ownership, and read failedGen under one mutex boundary
		pendingCh, failedGen, err := s.acquireCIMDPendingClear(ctx, client.ID)
		if err != nil {
			return err
		}
		handedOffToFinish := false
		defer func() {
			if !handedOffToFinish {
				s.abortCIMDRepair(client.ID, pendingCh)
			}
		}()
		// Lock any existing row before computing metadata differences so a concurrent refresh cannot race between diffing and writing
		var existing model.OidcClient
		err = withClientNoKeyUpdateLock(s.dbFor(ctx)).
			Where("id = ?", client.ID).
			First(&existing).Error
		found := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if found && !existing.IsMetadataDocument() {
			return errors.New("client is already registered without a metadata document")
		}
		var metadataRevokesConsent bool
		if found {
			changes = metadataClientChanges(existing, client.OidcClient)
			// Every detected change except the display name invalidates what the user agreed to
			metadataRevokesConsent = slices.ContainsFunc(changes, func(field string) bool { return field != "client_name" })
		}
		// Repairing an invalidated or incompatible row (MetadataExpiresAt == nil, incompatible auth settings, or an unrepaired prior failed DB invalidation) always revokes surviving grants
		unresolvedFailure := failedGen != 0 && !txRepairedCIMD(ctx, client.ID, failedGen)
		wasInvalidated := found && (existing.MetadataExpiresAt == nil || !isCIMDClientCompatible(existing) || unresolvedFailure)
		revokeConsent = metadataRevokesConsent || wasInvalidated

		// Persist the refreshed metadata and consent invalidation together; when running inside a caller-owned transaction, wrap them in a savepoint so a failed grant deletion cannot leave fresh metadata committed alongside surviving grants
		writeMetadataAndRevoke := func(wCtx context.Context) error {
			if err := s.upsertMetadataClient(wCtx, s.dbFor(wCtx), &client.OidcClient, found); err != nil {
				return err
			}

			// A security-relevant document change invalidates consent because it changes what the user previously approved
			// No back-channel logout tokens are sent here, which is safe only because CIMD clients cannot register a backchannel_logout_url today
			// If that URL is ever mapped from the metadata document, resolve the notifications before this delete removes the rows
			if revokeConsent {
				if _, err := s.revokeClientGrants(wCtx, client.ID); err != nil {
					return err
				}
			}

			// Reload inside the transaction so DB-managed columns and preloads are populated consistently
			// This bypasses GetClient so an admin can still refresh the metadata of a disabled client
			storedModel, err := s.firstClientByID(wCtx, client.ID)
			if err != nil {
				return err
			}
			if !storedModel.IsMetadataDocument() {
				return errors.New("client is already registered without a metadata document")
			}
			stored, err = s.clientFromModel(wCtx, s.dbFor(wCtx), storedModel)
			return err
		}
		if inCallerTx {
			if spErr := s.dbFor(ctx).Transaction(func(spTx *gorm.DB) error {
				return writeMetadataAndRevoke(contextWithTx(ctx, spTx))
			}); spErr != nil {
				return spErr
			}
		} else if wErr := writeMetadataAndRevoke(ctx); wErr != nil {
			return wErr
		}

		if hasTxHooks(ctx) {
			// Disarm any earlier failed-mismatch commit retry in this same transaction so committing this repair does not get undone by that queued retry (rollback still retries)
			cancelPostTxOnCommit(ctx, "cimd-mismatch:"+client.ID)
			if wasInvalidated || unresolvedFailure {
				id := client.ID
				gen := failedGen
				if unresolvedFailure {
					markTxCIMDRepaired(ctx, id, gen)
				}
				handedOffToFinish = true
				// Coordinate the commit-and-clear/repair-epoch boundary across transactions via afterTxFinish so waiters always unblock before any rollback callback acquires database locks
				afterTxFinish(ctx, func(committed bool) {
					s.finishCIMDClear(id, gen, pendingCh, committed, true)
				})
			}
		}
		return nil
	}
	err := withTx(ctx, s.db, runStore)
	if err != nil {
		return nil, err
	}

	if len(changes) > 0 {
		slog.InfoContext(ctx, "Client metadata changed",
			slog.String("client_id", client.ID),
			slog.Any("changed_fields", changes),
		)
	}
	if revokeConsent {
		slog.WarnContext(ctx, "Revoked existing user consent after a security-relevant client metadata change",
			slog.String("client_id", client.ID),
		)
	}

	return stored, nil
}

// revokeClientGrants deletes every consent users gave to a client together with the tokens and pending interactions issued under it, and returns how many rows it deleted
// The caller runs it in a transaction so a client never keeps tokens or pending approvals whose consent is gone
func (s *Store) revokeClientGrants(ctx context.Context, clientID string) (int64, error) {
	// Lock the client row with FOR NO KEY UPDATE on PostgreSQL so revocation serializes with concurrent CIMD consent/code flows without conflicting with FOR KEY SHARE foreign-key checks from token rotation
	var lockedClient model.OidcClient
	if err := withClientNoKeyUpdateLock(s.dbFor(ctx)).Select("id").Where("id = ?", clientID).First(&lockedClient).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("failed to lock client for grant revocation: %w", err)
	}

	// Delete consents, then OAuth2 sessions, then pending interactions to match the manual revocation order in clientService and prevent lock-order deadlocks
	consents := s.dbFor(ctx).
		Where("client_id = ?", clientID).
		Delete(&model.UserAuthorizedOidcClient{})
	if consents.Error != nil {
		return 0, fmt.Errorf("failed to revoke the consent of the client: %w", consents.Error)
	}

	// Users revoke tokens through their consent, so the tokens issued under it must go with it
	tokens := s.dbFor(ctx).
		Where("client_id = ?", clientID).
		Delete(&OAuth2Session{})
	if tokens.Error != nil {
		return 0, fmt.Errorf("failed to revoke the tokens of the client: %w", tokens.Error)
	}

	interactions := s.dbFor(ctx).
		Where("client_id = ?", clientID).
		Delete(&InteractionSession{})
	if interactions.Error != nil {
		return 0, fmt.Errorf("failed to revoke pending interactions of the client: %w", interactions.Error)
	}
	return consents.RowsAffected + tokens.RowsAffected + interactions.RowsAffected, nil
}

// invalidateCIMDClientCache clears metadata_expires_at so a rejected metadata document cannot be served fresh or stale until a valid fetch succeeds
func (s *Store) invalidateCIMDClientCache(ctx context.Context, clientID string) error {
	return s.dbFor(ctx).
		Model(&model.OidcClient{}).
		Where("id = ? AND client_type = ?", clientID, model.OidcClientTypeCIMD).
		Update("metadata_expires_at", nil).Error
}

// lockCIMDClientID acquires a transaction-scoped advisory lock on PostgreSQL so first-time CIMD inserts and missing-row mismatch cleanups serialize even before an oidc_clients row exists.
func lockCIMDClientID(tx *gorm.DB, clientID string) error {
	if tx.Name() != "postgres" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "pocket-id:cimd:"+clientID).Error
}

// withClientNoKeyUpdateLock acquires FOR NO KEY UPDATE on PostgreSQL so CIMD revocation and authorization flows serialize without blocking FOR KEY SHARE foreign-key checks from oauth2_sessions writes.
func withClientNoKeyUpdateLock(db *gorm.DB) *gorm.DB {
	if db.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "NO KEY UPDATE"})
	}
	return db
}

// metadataClientChanges returns the names of security-relevant metadata fields that differ between the stored client and a freshly fetched one
func metadataClientChanges(old, next model.OidcClient) []string {
	var changed []string
	if !slices.Equal([]string(old.CallbackURLs), next.CallbackURLs) {
		changed = append(changed, "redirect_uris")
	}
	if !slices.Equal([]string(old.LogoutCallbackURLs), next.LogoutCallbackURLs) {
		changed = append(changed, "post_logout_redirect_uris")
	}
	federatedChanged := (len(old.Credentials.FederatedIdentities) > 0 || len(next.Credentials.FederatedIdentities) > 0) &&
		!reflect.DeepEqual(old.Credentials.FederatedIdentities, next.Credentials.FederatedIdentities)
	if old.IsPublic != next.IsPublic || old.PkceEnabled != next.PkceEnabled || federatedChanged {
		changed = append(changed, "token_endpoint_auth_method")
	}
	if old.Name != next.Name {
		changed = append(changed, "client_name")
	}
	if !slices.Equal(effectiveMetadataGrantTypes(old.MetadataGrantTypes), effectiveMetadataGrantTypes(next.MetadataGrantTypes)) {
		changed = append(changed, "grant_types")
	}
	return changed
}

func effectiveMetadataGrantTypes(grantTypes datatype.StringList) []string {
	if len(grantTypes) == 0 {
		return []string{string(fosite.GrantTypeAuthorizationCode)}
	}
	return grantTypes
}

// upsertMetadataClient inserts a new managed client or updates the metadata-derived columns of an existing one, leaving consent, grants, and group links untouched
func (s *Store) upsertMetadataClient(ctx context.Context, tx *gorm.DB, client *model.OidcClient, update bool) error {
	if !update {
		return tx.WithContext(ctx).
			Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).
			Create(client).Error
	}
	return tx.WithContext(ctx).
		Model(&model.OidcClient{Base: model.Base{ID: client.ID}}).
		Select("Name", "CallbackURLs", "LogoutCallbackURLs", "Credentials",
			"IsPublic", "PkceEnabled", "ClientType", "MetadataExpiresAt",
			"MetadataGrantTypes").
		Updates(client).Error
}

func (s *Store) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	var count int64
	err := s.dbFor(ctx).
		Model(&clientAssertionJTI{}).
		Where("jti = ? AND expires_at > ?", jti, datatype.DateTime(time.Now())).
		Count(&count).
		Error
	if err != nil {
		return err
	}
	if count > 0 {
		return fosite.ErrJTIKnown
	}
	return nil
}

func (s *Store) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	err := s.dbFor(ctx).Create(&clientAssertionJTI{
		JTI:       jti,
		ExpiresAt: datatype.DateTime(exp),
	}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fosite.ErrJTIKnown
	}
	return err
}

func (s *Store) firstClientByID(ctx context.Context, id string) (model.OidcClient, error) {
	var client model.OidcClient
	err := s.dbFor(ctx).
		Preload("AllowedUserGroups").
		First(&client, "id = ?", id).
		Error
	if err != nil {
		return model.OidcClient{}, err
	}
	return client, nil
}

// Satisfies fositeoauth2.AuthorizeCodeStorage, fositeoauth2.AccessTokenStorage and fositeoauth2.RefreshTokenStorage

func (s *Store) CreateAuthorizeCodeSession(ctx context.Context, signature string, request fosite.Requester) error {
	return s.upsertSession(ctx, sessionKindAuthorizeCode, signature, request, "", true, fosite.AuthorizeCode)
}

func (s *Store) GetAuthorizeCodeSession(ctx context.Context, _ string, signature string, _ fosite.Session) (fosite.Requester, error) {
	request, active, err := s.getRequesterSession(ctx, sessionKindAuthorizeCode, signature)
	if err != nil {
		return nil, err
	}
	if !active {
		return request, fosite.ErrInvalidatedAuthorizeCode
	}
	return request, nil
}

func (s *Store) InvalidateAuthorizeCodeSession(ctx context.Context, _ string, signature string) error {
	err := s.deactivateSession(ctx, sessionKindAuthorizeCode, signature)

	// The code was active when the token request read it, so a missing active row means a concurrent request redeemed it first
	if errors.Is(err, fosite.ErrNotFound) {
		return fosite.ErrInvalidGrant.WithHint("The authorization code has already been used.")
	}
	return err
}

func (s *Store) CreateAccessTokenSession(ctx context.Context, signature string, request fosite.Requester) error {
	// userinfo and introspection read the granted audience from the persisted access token session, so an access token granted an identity scope is stored with the issuer added to its audience, letting it be presented to Pocket ID's own identity endpoints
	// A token audienced only to a custom API carries no identity scope here and so never gains the issuer audience
	return s.upsertSession(ctx, sessionKindAccessToken, signature, withIdentityAudience(request, s.issuer), "", true, fosite.AccessToken)
}

func (s *Store) GetAccessTokenSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	request, _, err := s.getRequesterSession(ctx, sessionKindAccessToken, signature)
	return request, err
}

func (s *Store) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, sessionKindAccessToken, signature)
}

func (s *Store) CreateRefreshTokenSession(ctx context.Context, signature string, accessSignature string, request fosite.Requester) error {
	return s.upsertSession(ctx, sessionKindRefreshToken, signature, request, accessSignature, true, fosite.RefreshToken)
}

func (s *Store) GetRefreshTokenSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	session, err := s.getSession(ctx, sessionKindRefreshToken, signature)
	if err != nil {
		return nil, err
	}
	request, err := s.decodeRequester(ctx, session.RequestData)
	if err != nil {
		return nil, err
	}
	// Client resolution for a CIMD client may have repaired metadata and revoked sessions; recheck the persisted session before returning it
	if client, ok := request.GetClient().(Client); ok && client.IsMetadataDocument() {
		session, err = s.getSession(ctx, sessionKindRefreshToken, signature)
		if err != nil {
			return nil, err
		}
	}
	// Allow retries briefly after rotation while keeping explicitly revoked tokens inactive
	if !session.Active && (session.RotatedAt == nil || !time.Now().UTC().Before(session.RotatedAt.ToTime().Add(refreshTokenGracePeriod))) {
		return request, fosite.ErrInactiveToken
	}
	return request, nil
}

func (s *Store) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	return s.deleteSession(ctx, sessionKindRefreshToken, signature)
}

func (s *Store) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	// If the caller already locked a CIMD client row for this exchange, proceed directly to the session update
	if _, alreadyLocked := ctx.Value(lockedResolvedClientKey{}).(Client); alreadyLocked {
		return s.rotateRefreshTokenLocked(ctx, requestID, refreshTokenSignature)
	}
	// If the caller has not already locked a CIMD client row for this exchange, keep the row lock, policy check, and session updates in one transaction
	return withTx(ctx, s.db, func(txCtx context.Context) error {
		var lockedClient model.OidcClient
		lockErr := withClientNoKeyUpdateLock(s.dbFor(txCtx)).
			Select("id", "metadata_expires_at", "disabled", "is_public", "pkce_enabled", "credentials").
			Where("client_type = ? AND id = (SELECT client_id FROM oauth2_sessions WHERE kind = ? AND key = ? AND request_id = ? LIMIT 1)",
				model.OidcClientTypeCIMD, sessionKindRefreshToken, refreshTokenSignature, requestID).
			First(&lockedClient).Error
		if lockErr != nil && !errors.Is(lockErr, gorm.ErrRecordNotFound) {
			return lockErr
		}
		if lockErr == nil {
			if lockedClient.Disabled || !isCIMDMetadataActive(lockedClient, time.Now()) || s.isCIMDInvalidatedForContext(txCtx, lockedClient.ID) {
				return fosite.ErrInactiveToken
			}
			if resolver, ok := s.clientResolver.(*cimdClientResolver); ok {
				if err := resolver.policy.AllowCIMDClient(txCtx, lockedClient.ID); err != nil {
					return fosite.ErrInactiveToken
				}
			}
		}
		return s.rotateRefreshTokenLocked(txCtx, requestID, refreshTokenSignature)
	})
}

func (s *Store) rotateRefreshTokenLocked(ctx context.Context, requestID string, refreshTokenSignature string) error {
	// Atomically start the grace period once so parallel refreshes cannot extend it or revive revoked tokens
	now := time.Now().UTC()
	result := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind = ? AND key = ? AND request_id = ?", sessionKindRefreshToken, refreshTokenSignature, requestID).
		Where("active = ? OR rotated_at > ?", true, datatype.DateTime(now.Add(-refreshTokenGracePeriod))).
		Updates(map[string]any{
			"active":     false,
			"rotated_at": gorm.Expr("COALESCE(rotated_at, ?)", datatype.DateTime(now)),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fosite.ErrInactiveToken
	}

	// Revoke only the access token paired with this refresh token so concurrent refreshes keep their access tokens
	session, err := s.getSession(ctx, sessionKindRefreshToken, refreshTokenSignature)
	if err != nil {
		return err
	}
	return s.DeleteAccessTokenSession(ctx, session.AccessTokenSignature)
}

// Satisfies fositeoauth2.TokenRevocationStorage

func (s *Store) RevokeRefreshToken(ctx context.Context, requestID string) error {
	return s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind = ? AND request_id = ?", sessionKindRefreshToken, requestID).
		Updates(map[string]any{"active": false, "rotated_at": nil}).
		Error
}

func (s *Store) RevokeAccessToken(ctx context.Context, requestID string) error {
	return s.dbFor(ctx).
		Where("kind = ? AND request_id = ?", sessionKindAccessToken, requestID).
		Delete(&OAuth2Session{}).
		Error
}

func (s *Store) RevokeSessionsByIDTokenHint(ctx context.Context, userID, clientID, idTokenJTI string) error {
	_, jtiMatches, err := s.findGrantRequestIDs(ctx, userID, clientID, idTokenJTI)
	if err != nil {
		return err
	}

	return s.revokeRequestIDs(ctx, jtiMatches)
}

func RevokeUserClientSessions(ctx context.Context, db *gorm.DB, userID, clientID string) error {
	s := NewStore(db, nil)
	requestIDs, _, err := s.findGrantRequestIDs(ctx, userID, clientID, "")
	if err != nil {
		return err
	}
	return s.revokeRequestIDs(ctx, requestIDs)
}

// RevokeUserGrants revokes every grant of the user at every client, offline_access ones included
func RevokeUserGrants(ctx context.Context, db *gorm.DB, userID string) error {
	s := NewStore(db, nil)
	requestIDs, _, err := s.findGrantRequestIDs(ctx, userID, "", "")
	if err != nil {
		return err
	}
	return s.revokeRequestIDs(ctx, requestIDs)
}

// findGrantRequestIDs returns the request IDs of the usable grants of the user and the optional client, plus the subset matching the optional ID token JTI
// Grants are found through their refresh and access tokens and through codes that weren't exchanged yet, which become grants once they are
// Access tokens are searched as well because clients without the refresh_token grant never get a refresh token session
// The subset leaves out grants that requested offline_access, because those outlive the logout the ID token is passed to
func (s *Store) findGrantRequestIDs(ctx context.Context, userID, clientID, idTokenJTI string) (candidates []string, jtiMatches []string, err error) {
	var sessions []OAuth2Session
	query := s.dbFor(ctx).
		Select("request_id", "request_data").
		Where("kind = ? OR (kind IN ? AND (active = ? OR rotated_at > ?))", sessionKindAccessToken, []string{sessionKindAuthorizeCode, sessionKindDeviceCode, sessionKindRefreshToken}, true, datatype.DateTime(time.Now().UTC().Add(-refreshTokenGracePeriod)))
	if clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}

	subjectField, err := requestDataField(query, "subject")
	if err != nil {
		return nil, nil, err
	}
	err = query.
		Where(subjectField+" = ?", userID).
		Find(&sessions).
		Error
	if err != nil {
		return nil, nil, err
	}

	candidateRequestIDs := map[string]struct{}{}
	matchingRequestIDs := map[string]struct{}{}
	for _, session := range sessions {
		// Add all sessions that match the user and client
		candidateRequestIDs[session.RequestID] = struct{}{}
		// Only add sessions that also match the ID token hint JTI
		if idTokenJTI == "" {
			continue
		}
		var stored storedRequester
		if err := json.Unmarshal([]byte(session.RequestData), &stored); err != nil {
			return nil, nil, err
		}
		if stored.Session != nil && stored.Session.Claims != nil && stored.Session.Claims.JTI == idTokenJTI && !stored.GrantedScope.Has("offline_access") {
			matchingRequestIDs[session.RequestID] = struct{}{}
		}
	}

	return mapKeys(candidateRequestIDs), mapKeys(matchingRequestIDs), nil
}

// EndBrowserSession revokes the grants issued from the user's Pocket ID session and returns every client the session authorized, so they can be told it ended
// Grants that requested offline_access are kept, because Back-Channel Logout 1.0 lets them outlive the session they were issued in
// They lose the session ID, so tokens refreshed from them, and the ID token a code not yet exchanged still mints, no longer name the ended session
// Revoked grants are read too, so the client that started an RP-initiated logout is notified although its grant was revoked just before
func EndBrowserSession(ctx context.Context, db *gorm.DB, userID, sessionID string) (clientIDs []string, err error) {
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		clientIDs, err = endBrowserSession(ctx, tx, userID, sessionID)
		return err
	})
	return clientIDs, err
}

func endBrowserSession(ctx context.Context, tx *gorm.DB, userID, sessionID string) ([]string, error) {
	subjectField, err := requestDataField(tx, "subject")
	if err != nil {
		return nil, err
	}
	sessionIDField, err := requestDataField(tx, "sid")
	if err != nil {
		return nil, err
	}

	var sessions []OAuth2Session
	err = tx.
		Select("id", "request_id", "client_id", "request_data").
		Where("kind IN ?", []string{sessionKindAuthorizeCode, sessionKindOpenID, sessionKindAccessToken, sessionKindRefreshToken}).
		Where(subjectField+" = ? AND "+sessionIDField+" = ?", userID, sessionID).
		Find(&sessions).
		Error
	if err != nil {
		return nil, err
	}

	clientIDs := map[string]struct{}{}
	revokedRequestIDs := map[string]struct{}{}
	for _, session := range sessions {
		clientIDs[session.ClientID] = struct{}{}
		var stored storedRequester
		if err := json.Unmarshal([]byte(session.RequestData), &stored); err != nil {
			return nil, err
		}
		if !stored.GrantedScope.Has("offline_access") {
			revokedRequestIDs[session.RequestID] = struct{}{}
			continue
		}

		stored.Session.SessionID = ""
		if stored.Session.Claims != nil {
			delete(stored.Session.Claims.Extra, common.SessionIDClaim)
		}
		requestData, err := json.Marshal(stored)
		if err != nil {
			return nil, err
		}
		err = tx.Model(&OAuth2Session{}).Where("id = ?", session.ID).Update("request_data", string(requestData)).Error
		if err != nil {
			return nil, err
		}
	}

	err = NewStore(tx, nil).revokeRequestIDs(ctx, mapKeys(revokedRequestIDs))
	if err != nil {
		return nil, err
	}
	return mapKeys(clientIDs), nil
}

// GrantSession is a Pocket ID session in which a user authorized a client
type GrantSession struct {
	UserID    string
	ClientID  string
	SessionID string
}

// GrantSessions returns the Pocket ID sessions in which the given users authorized the given clients, read from the refresh tokens of their grants
func GrantSessions(ctx context.Context, db *gorm.DB, userIDs, clientIDs []string) ([]GrantSession, error) {
	if len(userIDs) == 0 || len(clientIDs) == 0 {
		return nil, nil
	}

	subjectField, err := requestDataField(db, "subject")
	if err != nil {
		return nil, err
	}
	sessionIDField, err := requestDataField(db, "sid")
	if err != nil {
		return nil, err
	}

	var grantSessions []GrantSession
	err = db.
		WithContext(ctx).
		Model(&OAuth2Session{}).
		Select("DISTINCT client_id, "+subjectField+" AS user_id, "+sessionIDField+" AS session_id").
		Where("kind = ? AND client_id IN ?", sessionKindRefreshToken, clientIDs).
		Where(subjectField+" IN ?", userIDs).
		Where(sessionIDField + " <> ''").
		Scan(&grantSessions).
		Error
	if err != nil {
		return nil, err
	}
	return grantSessions, nil
}

// requestDataField returns the SQL expression that reads a field of the stored session from the JSON request data
func requestDataField(db *gorm.DB, field string) (string, error) {
	switch db.Name() {
	case "sqlite":
		return "json_extract(CAST(request_data AS TEXT), '$.session." + field + "')", nil
	case "postgres":
		return "request_data #>> '{session," + field + "}'", nil
	default:
		return "", fmt.Errorf("unsupported database dialect: %s", db.Name())
	}
}

func (s *Store) revokeRequestIDs(ctx context.Context, requestIDs []string) error {
	if len(requestIDs) == 0 {
		return nil
	}

	if err := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind = ? AND request_id IN ?", sessionKindRefreshToken, requestIDs).
		Updates(map[string]any{"active": false, "rotated_at": nil}).
		Error; err != nil {
		return err
	}

	// A code that is not exchanged yet must not mint tokens for a revoked grant
	if err := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind IN ? AND request_id IN ?", []string{sessionKindAuthorizeCode, sessionKindDeviceCode}, requestIDs).
		Update("active", false).
		Error; err != nil {
		return err
	}

	return s.dbFor(ctx).
		Where("kind = ? AND request_id IN ?", sessionKindAccessToken, requestIDs).
		Delete(&OAuth2Session{}).
		Error
}

func mapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// Satisfies pkce.PKCERequestStorage

func (s *Store) CreatePKCERequestSession(ctx context.Context, signature string, requester fosite.Requester) error {
	// PKCE sessions share the authorize code lifespan so abandoned ones expire and get cleaned up
	return s.upsertSession(ctx, sessionKindPKCE, signature, requester, "", true, fosite.AuthorizeCode)
}

func (s *Store) GetPKCERequestSession(ctx context.Context, _ string, signature string, _ fosite.Session) (fosite.Requester, error) {
	request, _, err := s.getRequesterSession(ctx, sessionKindPKCE, signature)
	return request, err
}

func (s *Store) DeletePKCERequestSession(ctx context.Context, _ string, signature string) error {
	return s.deleteSession(ctx, sessionKindPKCE, signature)
}

// Satisfies openid.OpenIDConnectRequestStorage

func (s *Store) CreateOpenIDConnectSession(ctx context.Context, authorizeCode string, requester fosite.Requester) error {
	// Sessions of the device authorization grant are keyed by the device code signature and expire together with the device code
	// An expiry ensures the cleanup job removes sessions whose code is never redeemed
	expiresAtKey := fosite.AuthorizeCode
	if session := requester.GetSession(); session != nil && session.GetExpiresAt(fosite.AuthorizeCode).IsZero() {
		expiresAtKey = fosite.DeviceCode
	}
	return s.upsertSession(ctx, sessionKindOpenID, authorizeCode, requester, "", true, expiresAtKey)
}

func (s *Store) GetOpenIDConnectSession(ctx context.Context, authorizeCode string, requester fosite.Requester) (fosite.Requester, error) {
	request, _, err := s.getRequesterSession(ctx, sessionKindOpenID, authorizeCode)
	if errors.Is(err, fosite.ErrNotFound) {
		return nil, openid.ErrNoSessionFound
	}
	if err != nil {
		return nil, err
	}

	// The token endpoint rebuilt the claims from the current user data, so the ID token must not carry the ones stored at approval
	if current, ok := requester.GetSession().(*Session); ok {
		request.GetSession().(*Session).IDTokenClaims().Extra = current.IDTokenClaims().Extra
	}
	return request, nil
}

func (s *Store) DeleteOpenIDConnectSession(ctx context.Context, authorizeCode string) error {
	return s.deleteSession(ctx, sessionKindOpenID, authorizeCode)
}

// Satisfies fosite.PARStorage

func (s *Store) CreatePARSession(ctx context.Context, requestURI string, request fosite.AuthorizeRequester) error {
	return s.upsertAuthorizeSession(ctx, sessionKindPAR, requestURI, request, true, fosite.PushedAuthorizeRequestContext)
}

func (s *Store) GetPARSession(ctx context.Context, requestURI string) (fosite.AuthorizeRequester, error) {
	session, err := s.getSession(ctx, sessionKindPAR, requestURI)
	if err != nil {
		return nil, err
	}
	if !session.Active || session.ExpiresAt == nil || session.ExpiresAt.ToTime().Before(time.Now()) {
		return nil, fosite.ErrNotFound
	}

	result := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind = ? AND key = ? AND active = ?", sessionKindPAR, requestURI, true).
		Update("active", false)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fosite.ErrNotFound
	}

	requester, err := s.decodeAuthorizeRequester(ctx, session.RequestData)
	if err != nil {
		return nil, err
	}
	// Client resolution for a CIMD client may have repaired metadata and deleted oauth2_sessions; verify the PAR session still exists
	if client, ok := requester.GetClient().(Client); ok && client.IsMetadataDocument() {
		if _, err := s.getSession(ctx, sessionKindPAR, requestURI); err != nil {
			return nil, err
		}
	}
	return requester, nil
}

func (s *Store) DeletePARSession(ctx context.Context, requestURI string) error {
	return s.deleteSession(ctx, sessionKindPAR, requestURI)
}

// Satisfies rfc8628.DeviceAuthStorage

func (s *Store) CreateDeviceAuthSession(ctx context.Context, deviceCodeSignature, userCodeSignature string, request fosite.DeviceRequester) error {
	requestData, err := s.encodeDeviceRequester(request)
	if err != nil {
		return err
	}

	if _, err := s.getSession(ctx, sessionKindUserCode, userCodeSignature); err == nil {
		return fosite.ErrExistingUserCodeSignature
	} else if !errors.Is(err, fosite.ErrNotFound) {
		return err
	}

	expDeviceCode := expiresAt(request.GetSession(), fosite.DeviceCode)
	expUserCode := expiresAt(request.GetSession(), fosite.UserCode)
	if err := s.storeSession(ctx, sessionKindDeviceCode, deviceCodeSignature, request.GetID(), "", true, requestData, expDeviceCode); err != nil {
		return err
	}
	return s.storeSession(ctx, sessionKindUserCode, userCodeSignature, request.GetID(), "", true, requestData, expUserCode)
}

func (s *Store) GetDeviceCodeSession(ctx context.Context, signature string, _ fosite.Session) (fosite.DeviceRequester, error) {
	request, active, err := s.getDeviceRequesterSession(ctx, sessionKindDeviceCode, signature)
	if err != nil {
		return nil, err
	}
	if !active {
		return request, fosite.ErrInvalidatedDeviceCode
	}
	return request, nil
}

func (s *Store) InvalidateDeviceCodeSession(ctx context.Context, signature string) error {
	session, err := s.getSession(ctx, sessionKindDeviceCode, signature)
	if err != nil {
		return err
	}

	// Only flip rows that are still active so two concurrent token requests for the same
	// device code cannot both pass the single-use check and each mint a token set.
	result := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind IN ? AND request_id = ? AND active = ?", []string{sessionKindDeviceCode, sessionKindUserCode}, session.RequestID, true).
		Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// The device code was active when the token request read it, so a concurrent request redeemed it first
		return fosite.ErrInvalidGrant.WithHint("The device code has already been used.")
	}
	return nil
}

func (s *Store) GetDeviceCodeSessionByUserCodeSignature(ctx context.Context, signature string) (fosite.DeviceRequester, error) {
	request, active, err := s.getDeviceRequesterSession(ctx, sessionKindUserCode, signature)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, fosite.ErrNotFound
	}
	return request, nil
}

func (s *Store) AcceptDeviceCodeSessionByUserCodeSignature(ctx context.Context, signature string, request fosite.DeviceRequester) (string, error) {
	var deviceCodeSignature string
	err := withTx(ctx, s.db, func(ctx context.Context) error {
		// Verify the currently persisted user-code state because the caller may hold a stale unused request
		userCodeSession, err := s.getSession(ctx, sessionKindUserCode, signature)
		if err != nil {
			return err
		}
		// Decoding the full requester would resolve the client, which for a metadata document client can be a fetch that must not hold the write lock
		var storedRequest storedRequester
		err = json.Unmarshal([]byte(userCodeSession.RequestData), &storedRequest)
		if err != nil {
			return err
		}
		if !userCodeSession.Active || storedRequest.UserCodeState != fosite.UserCodeUnused {
			return fosite.ErrNotFound
		}

		// Prepare the accepted request before claiming the persisted unused state
		request.SetUserCodeState(fosite.UserCodeAccepted)
		requestData, err := s.encodeDeviceRequester(request)
		if err != nil {
			return err
		}
		deviceCodeSession, err := s.getSessionByRequestID(ctx, sessionKindDeviceCode, userCodeSession.RequestID)
		if err != nil {
			return err
		}

		// Claim the user code by deactivating it only if no competing acceptance already did so
		result := s.dbFor(ctx).
			Model(&OAuth2Session{}).
			Where("kind = ? AND key = ? AND request_id = ? AND active = ?", sessionKindUserCode, signature, userCodeSession.RequestID, true).
			Updates(map[string]any{
				"active":       false,
				"request_data": requestData,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fosite.ErrNotFound
		}

		// Update the paired device-code session only after the user-code claim succeeds
		result = s.dbFor(ctx).
			Model(&OAuth2Session{}).
			Where("kind = ? AND request_id = ? AND active = ?", sessionKindDeviceCode, userCodeSession.RequestID, true).
			Update("request_data", requestData)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fosite.ErrNotFound
		}

		deviceCodeSignature = deviceCodeSession.Key
		return nil
	})
	return deviceCodeSignature, err
}

// Satisfies fosite.Transactional
func (s *Store) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return withTx(ctx, s.db, fn)
}

func (s *Store) upsertSession(ctx context.Context, kind string, key string, requester fosite.Requester, accessTokenSignature string, active bool, expiresAtKey fosite.TokenType) error {
	requestData, err := s.encodeRequester(requester)
	if err != nil {
		return err
	}

	return s.storeSession(ctx, kind, key, requester.GetID(), accessTokenSignature, active, requestData, expiresAt(requester.GetSession(), expiresAtKey))
}

func (s *Store) upsertAuthorizeSession(ctx context.Context, kind string, key string, requester fosite.AuthorizeRequester, active bool, expiresAtKey fosite.TokenType) error {
	requestData, err := s.encodeAuthorizeRequester(requester)
	if err != nil {
		return err
	}

	return s.storeSession(ctx, kind, key, requester.GetID(), "", active, requestData, expiresAt(requester.GetSession(), expiresAtKey))
}

func (s *Store) storeSession(ctx context.Context, kind string, key string, requestID string, accessTokenSignature string, active bool, requestData string, exp *datatype.DateTime) error {
	clientID, err := sessionClientID(requestData)
	if err != nil {
		return err
	}

	session := OAuth2Session{
		Kind:                 kind,
		Key:                  key,
		RequestID:            requestID,
		ClientID:             clientID,
		AccessTokenSignature: accessTokenSignature,
		Active:               active,
		RequestData:          requestData,
		ExpiresAt:            exp,
	}

	return s.dbFor(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "kind"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"request_id",
				"client_id",
				"access_token_signature",
				"active",
				"request_data",
				"expires_at",
			}),
		}).
		Create(&session).
		Error
}

func sessionClientID(requestData string) (string, error) {
	var stored storedRequester
	if err := json.Unmarshal([]byte(requestData), &stored); err != nil {
		return "", err
	}

	return stored.ClientID, nil
}

func (s *Store) getRequesterSession(ctx context.Context, kind string, key string) (fosite.Requester, bool, error) {
	session, err := s.getSession(ctx, kind, key)
	if err != nil {
		return nil, false, err
	}

	requester, err := s.decodeRequester(ctx, session.RequestData)
	if err != nil {
		return nil, false, err
	}

	// Client resolution for a CIMD client may have repaired metadata and deleted oauth2_sessions; recheck the persisted session before returning it
	if client, ok := requester.GetClient().(Client); ok && client.IsMetadataDocument() {
		session, err = s.getSession(ctx, kind, key)
		if err != nil {
			return nil, false, err
		}
	}

	return requester, session.Active, nil
}

func (s *Store) getDeviceRequesterSession(ctx context.Context, kind string, key string) (fosite.DeviceRequester, bool, error) {
	session, err := s.getSession(ctx, kind, key)
	if err != nil {
		return nil, false, err
	}

	requester, err := s.decodeDeviceRequester(ctx, session.RequestData)
	if err != nil {
		return nil, false, err
	}

	if client, ok := requester.GetClient().(Client); ok && client.IsMetadataDocument() {
		session, err = s.getSession(ctx, kind, key)
		if err != nil {
			return nil, false, err
		}
	}

	return requester, session.Active, nil
}

func (s *Store) getSession(ctx context.Context, kind string, key string) (session OAuth2Session, err error) {
	err = s.dbFor(ctx).
		Where("kind = ? AND key = ?", kind, key).
		First(&session).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return session, fosite.ErrNotFound
	}
	if err != nil {
		return session, err
	}

	return session, nil
}

func (s *Store) getSessionByRequestID(ctx context.Context, kind string, requestID string) (session OAuth2Session, err error) {
	err = s.dbFor(ctx).
		Where("kind = ? AND request_id = ?", kind, requestID).
		First(&session).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return session, fosite.ErrNotFound
	}
	if err != nil {
		return session, err
	}

	return session, nil
}

func (s *Store) deleteSession(ctx context.Context, kind string, key string) error {
	return s.dbFor(ctx).
		Where("kind = ? AND key = ?", kind, key).
		Delete(&OAuth2Session{}).
		Error
}

func (s *Store) deactivateSession(ctx context.Context, kind string, key string) error {
	result := s.dbFor(ctx).
		Model(&OAuth2Session{}).
		Where("kind = ? AND key = ? AND active = ?", kind, key, true).
		Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fosite.ErrNotFound
	}
	return nil
}

func (s *Store) encodeRequester(requester fosite.Requester) (string, error) {
	stored, err := s.storedRequesterFromRequester(requester)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Store) encodeAuthorizeRequester(requester fosite.AuthorizeRequester) (string, error) {
	stored, err := s.storedRequesterFromRequester(requester)
	if err != nil {
		return "", err
	}

	stored.Authorize = true
	stored.ResponseTypes = cloneArguments(requester.GetResponseTypes())
	if redirectURI := requester.GetRedirectURI(); redirectURI != nil {
		stored.RedirectURI = redirectURI.String()
	}
	stored.State = requester.GetState()
	stored.ResponseMode = requester.GetResponseMode()
	stored.DefaultResponseMode = requester.GetDefaultResponseMode()

	if ar, ok := requester.(*fosite.AuthorizeRequest); ok {
		stored.HandledResponseTypes = cloneArguments(ar.HandledResponseTypes)
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Store) encodeDeviceRequester(requester fosite.DeviceRequester) (string, error) {
	stored, err := s.storedRequesterFromRequester(requester)
	if err != nil {
		return "", err
	}

	stored.Device = true
	stored.UserCodeState = requester.GetUserCodeState()

	data, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Store) storedRequesterFromRequester(requester fosite.Requester) (storedRequester, error) {
	if requester == nil {
		return storedRequester{}, fosite.ErrServerError.WithHint("requester must not be nil")
	}

	return storedRequester{
		ID:                requester.GetID(),
		RequestedAt:       requester.GetRequestedAt(),
		ClientID:          requester.GetClient().GetID(),
		RequestedScope:    cloneArguments(requester.GetRequestedScopes()),
		GrantedScope:      cloneArguments(requester.GetGrantedScopes()),
		Form:              sanitizeStoredForm(requester.GetRequestForm()),
		Session:           cloneSession(requester.GetSession()),
		RequestedAudience: cloneArguments(requester.GetRequestedAudience()),
		GrantedAudience:   cloneArguments(requester.GetGrantedAudience()),
	}, nil
}

func (s *Store) decodeRequester(ctx context.Context, data string) (fosite.Requester, error) {
	var stored storedRequester
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		return nil, err
	}
	if stored.Authorize {
		return s.requesterFromStoredAuthorize(ctx, stored)
	}
	return s.requesterFromStored(ctx, stored)
}

func (s *Store) decodeAuthorizeRequester(ctx context.Context, data string) (fosite.AuthorizeRequester, error) {
	var stored storedRequester
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		return nil, err
	}
	stored.Authorize = true
	return s.requesterFromStoredAuthorize(ctx, stored)
}

func (s *Store) decodeDeviceRequester(ctx context.Context, data string) (fosite.DeviceRequester, error) {
	var stored storedRequester
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		return nil, err
	}
	stored.Device = true
	return s.requesterFromStoredDevice(ctx, stored)
}

func (s *Store) requesterFromStored(ctx context.Context, stored storedRequester) (fosite.Requester, error) {
	client, err := s.resolvePersistedClient(ctx, stored.ClientID)
	if err != nil {
		return nil, err
	}

	request := fosite.NewRequest()
	request.ID = stored.ID
	request.RequestedAt = stored.RequestedAt
	request.Client = client
	request.RequestedScope = cloneArguments(stored.RequestedScope)
	request.GrantedScope = cloneArguments(stored.GrantedScope)
	request.Form = cloneValues(stored.Form)
	request.Session = stored.Session
	request.RequestedAudience = cloneArguments(stored.RequestedAudience)
	request.GrantedAudience = cloneArguments(stored.GrantedAudience)
	return request, nil
}

func (s *Store) requesterFromStoredDevice(ctx context.Context, stored storedRequester) (fosite.DeviceRequester, error) {
	requester, err := s.requesterFromStored(ctx, stored)
	if err != nil {
		return nil, err
	}

	base := requester.(*fosite.Request)
	request := fosite.NewDeviceRequest()
	request.Request = *base
	request.UserCodeState = stored.UserCodeState
	return request, nil
}

func (s *Store) requesterFromStoredAuthorize(ctx context.Context, stored storedRequester) (fosite.AuthorizeRequester, error) {
	requester, err := s.requesterFromStored(ctx, stored)
	if err != nil {
		return nil, err
	}

	base := requester.(*fosite.Request)
	request := fosite.NewAuthorizeRequest()
	request.Request = *base
	request.ResponseTypes = cloneArguments(stored.ResponseTypes)
	request.State = stored.State
	request.HandledResponseTypes = cloneArguments(stored.HandledResponseTypes)
	request.ResponseMode = stored.ResponseMode
	request.DefaultResponseMode = stored.DefaultResponseMode

	if stored.RedirectURI != "" {
		redirectURI, err := url.Parse(stored.RedirectURI)
		if err != nil {
			return nil, err
		}
		request.RedirectURI = redirectURI
	}

	return request, nil
}

func cloneSession(session fosite.Session) *Session {
	if session == nil {
		return nil
	}

	if s, ok := session.(*Session); ok {
		cloned := s.Clone()
		if typed, ok := cloned.(*Session); ok {
			return typed
		}
	}

	cloned := NewEmptySession()
	cloned.Subject = session.GetSubject()
	for _, tokenType := range []fosite.TokenType{
		fosite.AccessToken,
		fosite.RefreshToken,
		fosite.AuthorizeCode,
		fosite.IDToken,
		fosite.PushedAuthorizeRequestContext,
		fosite.DeviceCode,
		fosite.UserCode,
	} {
		if exp := session.GetExpiresAt(tokenType); !exp.IsZero() {
			cloned.SetExpiresAt(tokenType, exp)
		}
	}
	return cloned
}

func cloneArguments(arguments fosite.Arguments) fosite.Arguments {
	if len(arguments) == 0 {
		return fosite.Arguments{}
	}
	cloned := make(fosite.Arguments, len(arguments))
	copy(cloned, arguments)
	return cloned
}

func sanitizeStoredForm(values url.Values) url.Values {
	cloned := cloneValues(values)
	cloned.Del("client_secret")
	cloned.Del("client_assertion")
	return cloned
}

func cloneValues(values url.Values) url.Values {
	if len(values) == 0 {
		return url.Values{}
	}
	cloned := make(url.Values, len(values))
	for key, value := range values {
		cloned[key] = append([]string(nil), value...)
	}
	return cloned
}

func expiresAt(session fosite.Session, tokenType fosite.TokenType) *datatype.DateTime {
	if session == nil || tokenType == "" {
		return nil
	}
	exp := session.GetExpiresAt(tokenType)
	if exp.IsZero() {
		return nil
	}
	return new(datatype.DateTime(exp))
}

func (s *Store) dbFor(ctx context.Context) *gorm.DB {
	return dbFromContext(ctx, s.db)
}

// Fosite reaches every storage through a provider accessor, and Store implements all of them itself

func (s *Store) FositeClientManager() fosite.ClientManager {
	return s
}

func (s *Store) AuthorizeCodeStorage() fositeoauth2.AuthorizeCodeStorage {
	return s
}

func (s *Store) AccessTokenStorage() fositeoauth2.AccessTokenStorage {
	return s
}

func (s *Store) RefreshTokenStorage() fositeoauth2.RefreshTokenStorage {
	return s
}

func (s *Store) TokenRevocationStorage() fositeoauth2.TokenRevocationStorage {
	return s
}

func (s *Store) OpenIDConnectRequestStorage() openid.OpenIDConnectRequestStorage {
	return s
}

func (s *Store) PKCERequestStorage() pkce.PKCERequestStorage {
	return s
}

func (s *Store) DeviceAuthStorage() rfc8628.DeviceAuthStorage {
	return s
}

func (s *Store) PARStorage() fosite.PARStorage {
	return s
}
