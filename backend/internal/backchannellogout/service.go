package backchannellogout

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

	"github.com/italypaleale/francis/actor"
	francishost "github.com/italypaleale/francis/host"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/session"
)

// requestTimeout bounds each notification POST, so one unreachable client cannot stall the others
const requestTimeout = 10 * time.Second

// TokenSigner mints the logout tokens delivered to clients
type TokenSigner interface {
	GenerateLogoutToken(userID string, clientID string, sessionID string) (string, error)
}

// Service sends OIDC Back-Channel Logout 1.0 tokens to clients when a user's access is revoked
// Deliveries are scheduled as durable jobs, so they survive a restart and failed attempts are retried a capped number of times before giving up
type Service struct {
	db          *gorm.DB
	tokenSigner TokenSigner
	httpClient  *http.Client
	actors      *actor.Service
}

func NewService(db *gorm.DB, tokenSigner TokenSigner, httpClient *http.Client, actorsHost francishost.Host) (*Service, error) {
	s := &Service{
		db:          db,
		tokenSigner: tokenSigner,
		httpClient:  newHTTPClient(httpClient),
		actors:      actorsHost.Service(),
	}

	err := actorsHost.RegisterActor(
		ActorType,
		s.newNotifierActor,
		local.WithCapacityGroup(ActorType, deliveryConcurrency),
		local.WithMaxAttempts(deliveryMaxAttempts),
	)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", ActorType, err)
	}

	return s, nil
}

// newHTTPClient refuses to follow redirects, as Go would turn the POST into a body-less GET and the logout token would be silently dropped
// Returning the redirect response instead makes the delivery fail loudly on the status check
func newHTTPClient(source *http.Client) *http.Client {
	if source == nil {
		source = http.DefaultClient
	}

	return &http.Client{
		Transport: source.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// target is a single client to notify that a user's session should end
// SessionID is empty when the client must end every session of the user
type target struct {
	UserID    string
	ClientID  string
	LogoutURL string
	SessionID string
}

// authorization is a client a user authorized, which has to be told when the user's access ends
type authorization struct {
	UserID    string
	ClientID  string
	LogoutURL string
	// GrantSessionIDs are the Pocket ID sessions the client's grants were issued in
	GrantSessionIDs []string `gorm:"-"`
}

// authorizationsQuery selects the authorizations of clients that are registered for back-channel logout
// Callers narrow it down to the users or the client whose access was revoked
func (s *Service) authorizationsQuery(ctx context.Context, tx *gorm.DB) *gorm.DB {
	return tx.
		WithContext(ctx).
		Model(&model.UserAuthorizedOidcClient{}).
		Select("user_authorized_oidc_clients.user_id", "user_authorized_oidc_clients.client_id", "oidc_clients.backchannel_logout_url AS logout_url").
		Joins("JOIN oidc_clients ON oidc_clients.id = user_authorized_oidc_clients.client_id").
		Where("oidc_clients.backchannel_logout_url <> ''")
}

// authorizationsForUsers returns every client the given users have authorized that is registered for back-channel logout
func (s *Service) authorizationsForUsers(ctx context.Context, tx *gorm.DB, userIDs []string) ([]authorization, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	return s.scanAuthorizations(ctx, tx, s.authorizationsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.user_id IN (?)", userIDs))
}

// authorizationsForAuthorization returns the client to notify when a single authorization is revoked, and nothing when that client is not registered for back-channel logout
func (s *Service) authorizationsForAuthorization(ctx context.Context, tx *gorm.DB, userID string, clientID string) ([]authorization, error) {
	return s.scanAuthorizations(ctx, tx, s.authorizationsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.user_id = ?", userID).
		Where("user_authorized_oidc_clients.client_id = ?", clientID))
}

// authorizationsForClient returns every user who has authorized the given client, when that client is registered for back-channel logout
func (s *Service) authorizationsForClient(ctx context.Context, tx *gorm.DB, clientID string) ([]authorization, error) {
	return s.scanAuthorizations(ctx, tx, s.authorizationsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.client_id = ?", clientID))
}

// authorizationsForLostGroupAccess returns clients registered for back-channel logout that the matched users have authorized but can no longer access because of the client's group restriction
// It must run after the group membership or allowed-group change has been committed
func (s *Service) authorizationsForLostGroupAccess(ctx context.Context, tx *gorm.DB, userIDs []string, clientID string) ([]authorization, error) {
	// Require at least one filter, so a caller that computes an empty user list can never match every user of every client
	if len(userIDs) == 0 && clientID == "" {
		return nil, nil
	}

	query := s.authorizationsQuery(ctx, tx).
		Where("oidc_clients.is_group_restricted = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM oidc_clients_allowed_user_groups ag JOIN user_groups_users ugu ON ugu.user_group_id = ag.user_group_id WHERE ag.oidc_client_id = oidc_clients.id AND ugu.user_id = user_authorized_oidc_clients.user_id)")
	if len(userIDs) > 0 {
		query = query.Where("user_authorized_oidc_clients.user_id IN (?)", userIDs)
	}
	if clientID != "" {
		query = query.Where("user_authorized_oidc_clients.client_id = ?", clientID)
	}

	return s.scanAuthorizations(ctx, tx, query)
}

// scanAuthorizations returns the authorizations the query selects, each with the Pocket ID sessions its grants were issued in
func (s *Service) scanAuthorizations(ctx context.Context, tx *gorm.DB, query *gorm.DB) ([]authorization, error) {
	var authorizations []authorization
	err := query.Scan(&authorizations).Error
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(authorizations))
	clientIDs := make([]string, 0, len(authorizations))
	for _, a := range authorizations {
		userIDs = append(userIDs, a.UserID)
		clientIDs = append(clientIDs, a.ClientID)
	}
	grantSessions, err := oidc.GrantSessions(ctx, tx, userIDs, clientIDs)
	if err != nil {
		return nil, err
	}
	sessionIDs := make(map[[2]string][]string, len(grantSessions))
	for _, gs := range grantSessions {
		key := [2]string{gs.UserID, gs.ClientID}
		sessionIDs[key] = append(sessionIDs[key], gs.SessionID)
	}
	for i, a := range authorizations {
		authorizations[i].GrantSessionIDs = sessionIDs[[2]string{a.UserID, a.ClientID}]
	}
	return authorizations, nil
}

// notifyAuthorizations tells each client that the user's access ended, with one token without a sid that ends every session of the user at the client
// Clients that require a sid also get one token per session the user is still signed in to Pocket ID with and the client's grants were issued in
// It lists sessions through their actor, so it must be called after the transaction has committed
func (s *Service) notifyAuthorizations(ctx context.Context, authorizations []authorization) {
	liveSessionIDs := map[string][]string{}
	targets := make([]target, 0, len(authorizations))
	for _, a := range authorizations {
		targets = append(targets, target{UserID: a.UserID, ClientID: a.ClientID, LogoutURL: a.LogoutURL})

		live, ok := liveSessionIDs[a.UserID]
		if !ok {
			var err error
			live, err = session.ActiveSessionIDs(ctx, s.actors, a.UserID)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to list the sessions to notify for back-channel logout", slog.String("userId", a.UserID), slog.Any("error", err))
			}
			liveSessionIDs[a.UserID] = live
		}
		for _, sessionID := range a.GrantSessionIDs {
			if slices.Contains(live, sessionID) {
				targets = append(targets, target{UserID: a.UserID, ClientID: a.ClientID, LogoutURL: a.LogoutURL, SessionID: sessionID})
			}
		}
	}
	s.notifyClients(ctx, targets)
}

// PrepareUserNotifications resolves, within the given transaction, the logout notifications for users whose access is being revoked
// It exists for callers that delete the users or their authorizations, which are gone once the transaction commits
// The returned function delivers the notifications in the background and must only be called after the transaction has committed
// It is never nil, so callers that treat a failed lookup as non-fatal can call it unconditionally
func (s *Service) PrepareUserNotifications(ctx context.Context, tx *gorm.DB, userIDs []string) (func(), error) {
	authorizations, err := s.authorizationsForUsers(ctx, tx, userIDs)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyAuthorizations(ctx, authorizations) }, nil
}

// PrepareAuthorizationNotification resolves, within the given transaction, the logout notification for a single authorization that is being revoked
// The returned function behaves like the one from PrepareUserNotifications
func (s *Service) PrepareAuthorizationNotification(ctx context.Context, tx *gorm.DB, userID string, clientID string) (func(), error) {
	authorizations, err := s.authorizationsForAuthorization(ctx, tx, userID, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyAuthorizations(ctx, authorizations) }, nil
}

// PrepareClientNotifications resolves, within the given transaction, the logout notifications for every user of a client that is being deleted
// The returned function behaves like the one from PrepareUserNotifications
func (s *Service) PrepareClientNotifications(ctx context.Context, tx *gorm.DB, clientID string) (func(), error) {
	authorizations, err := s.authorizationsForClient(ctx, tx, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyAuthorizations(ctx, authorizations) }, nil
}

// NotifyUser delivers logout tokens to every client the user has authorized that is registered for back-channel logout
// It must be called after the change that revoked the user's access has been committed, and logs instead of failing because delivery is best effort
func (s *Service) NotifyUser(ctx context.Context, userID string) {
	authorizations, err := s.authorizationsForUsers(ctx, s.db, []string{userID})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.String("userId", userID), slog.Any("error", err))
		return
	}
	s.notifyAuthorizations(ctx, authorizations)
}

// NotifyLostGroupAccess delivers logout tokens for group-restricted clients that the matched users can no longer access
// Callers pass the users whose membership changed, the client whose allowed groups changed, or both to narrow the match
// It must be called after the group change has been committed, and logs instead of failing because delivery is best effort
func (s *Service) NotifyLostGroupAccess(ctx context.Context, userIDs []string, clientID string) {
	authorizations, err := s.authorizationsForLostGroupAccess(ctx, s.db, userIDs, clientID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.Any("error", err))
		return
	}
	s.notifyAuthorizations(ctx, authorizations)
}

// SessionsEnded revokes the grants issued from the user's Pocket ID sessions and tells every client authorized in them that the session ended
// It implements session.EndNotifier and logs instead of failing because delivery is best effort
func (s *Service) SessionsEnded(ctx context.Context, userID string, sessionIDs []string) {
	for _, sessionID := range sessionIDs {
		clientIDs, err := oidc.EndBrowserSession(ctx, s.db, userID, sessionID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to end the grants of an ended session", slog.String("userId", userID), slog.Any("error", err))
			continue
		}
		if len(clientIDs) == 0 {
			continue
		}

		var targets []target
		err = s.db.
			WithContext(ctx).
			Model(&model.OidcClient{}).
			Select("id AS client_id", "backchannel_logout_url AS logout_url").
			Where("id IN ? AND backchannel_logout_url <> ''", clientIDs).
			Scan(&targets).
			Error
		if err != nil {
			slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.String("userId", userID), slog.Any("error", err))
			continue
		}
		for i := range targets {
			targets[i].UserID = userID
			targets[i].SessionID = sessionID
		}
		s.notifyClients(ctx, targets)
	}
}

// SessionsExpired revokes the grants issued from the user's expired Pocket ID sessions without telling the clients, as Keycloak does
// It implements session.EndNotifier
func (s *Service) SessionsExpired(ctx context.Context, userID string, sessionIDs []string) error {
	var errs []error
	for _, sessionID := range sessionIDs {
		_, err := oidc.EndBrowserSession(ctx, s.db, userID, sessionID)
		if err != nil {
			errs = append(errs, fmt.Errorf("error ending the grants of an expired session: %w", err))
		}
	}
	return errors.Join(errs...)
}

// SignedOutEverywhere revokes the user's remaining grants, which are those that outlive sessions such as offline_access and device grants
// Like Keycloak's sign-out of a user, it notifies no clients for them
// It implements session.EndNotifier
func (s *Service) SignedOutEverywhere(ctx context.Context, userID string) error {
	err := oidc.RevokeUserGrants(ctx, s.db, userID)
	if err != nil {
		return fmt.Errorf("error revoking the user's grants: %w", err)
	}
	return nil
}

// notifyClients schedules a durable delivery job for each of the given clients, so callers are never blocked on slow or unreachable clients
// It must be called after the change that revoked the user's access has been committed
func (s *Service) notifyClients(ctx context.Context, targets []target) {
	for _, t := range targets {
		// One actor per authorization serializes its deliveries, while the capacity group limits running jobs without counting idle actors
		actorID := t.ClientID + ":" + t.UserID
		_, _, err := s.actors.Dispatch(ctx, ActorType, actorID, methodDeliver, t)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to schedule back-channel logout notification",
				slog.String("clientId", t.ClientID),
				slog.String("userId", t.UserID),
				slog.Any("error", err),
			)
		}
	}
}

func (s *Service) sendLogoutToken(parentCtx context.Context, t target) error {
	logoutToken, err := s.tokenSigner.GenerateLogoutToken(t.UserID, t.ClientID, t.SessionID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, requestTimeout)
	defer cancel()

	body := url.Values{"logout_token": []string{logoutToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.LogoutURL, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("%w: invalid logout request: %w", actor.ErrJobPermanentFailure, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// Only retry responses that indicate a potentially temporary failure at the client
		if res.StatusCode != http.StatusRequestTimeout && res.StatusCode != http.StatusTooManyRequests && res.StatusCode < http.StatusInternalServerError {
			return fmt.Errorf("%w: client responded with status %d", actor.ErrJobPermanentFailure, res.StatusCode)
		}
		return fmt.Errorf("client responded with status %d", res.StatusCode)
	}
	return nil
}
