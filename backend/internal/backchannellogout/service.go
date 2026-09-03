package backchannellogout

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// requestTimeout bounds each notification POST, so one unreachable client cannot stall the others
const requestTimeout = 10 * time.Second

// TokenSigner mints the logout tokens delivered to clients
type TokenSigner interface {
	GenerateLogoutToken(userID string, clientID string) (string, error)
}

// Service sends OIDC Back-Channel Logout 1.0 tokens to clients when a user's access is revoked
// Deliveries are scheduled as durable jobs, so they survive a restart and failed attempts are retried a capped number of times before giving up
type Service struct {
	db          *gorm.DB
	tokenSigner TokenSigner
	httpClient  *http.Client
	actors      *actor.Service
}

func NewService(db *gorm.DB, tokenSigner TokenSigner, httpClient *http.Client, actorsHost *local.Host) (*Service, error) {
	s := &Service{
		db:          db,
		tokenSigner: tokenSigner,
		httpClient:  newHTTPClient(httpClient),
		actors:      actorsHost.Service(),
	}

	err := actorsHost.RegisterActor(
		ActorType,
		s.newNotifierActor,
		local.WithConcurrencyLimit(deliveryConcurrency),
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
type target struct {
	UserID    string
	ClientID  string
	LogoutURL string
}

// targetsQuery selects the authorizations of clients that are registered for back-channel logout
// Callers narrow it down to the users or the client whose access was revoked
func (s *Service) targetsQuery(ctx context.Context, tx *gorm.DB) *gorm.DB {
	return tx.
		WithContext(ctx).
		Model(&model.UserAuthorizedOidcClient{}).
		Select("user_authorized_oidc_clients.user_id", "user_authorized_oidc_clients.client_id", "oidc_clients.backchannel_logout_url AS logout_url").
		Joins("JOIN oidc_clients ON oidc_clients.id = user_authorized_oidc_clients.client_id").
		Where("oidc_clients.backchannel_logout_url <> ''")
}

// targetsForUsers returns every client the given users have authorized that is registered for back-channel logout
func (s *Service) targetsForUsers(ctx context.Context, tx *gorm.DB, userIDs []string) ([]target, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var targets []target
	err := s.targetsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.user_id IN (?)", userIDs).
		Scan(&targets).
		Error
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// targetsForAuthorization returns the client to notify when a single authorization is revoked, and nothing when that client is not registered for back-channel logout
func (s *Service) targetsForAuthorization(ctx context.Context, tx *gorm.DB, userID string, clientID string) ([]target, error) {
	var targets []target
	err := s.targetsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.user_id = ?", userID).
		Where("user_authorized_oidc_clients.client_id = ?", clientID).
		Scan(&targets).
		Error
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// targetsForClient returns every user who has authorized the given client, when that client is registered for back-channel logout
func (s *Service) targetsForClient(ctx context.Context, tx *gorm.DB, clientID string) ([]target, error) {
	var targets []target
	err := s.targetsQuery(ctx, tx).
		Where("user_authorized_oidc_clients.client_id = ?", clientID).
		Scan(&targets).
		Error
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// targetsForLostGroupAccess returns clients registered for back-channel logout that the matched users have authorized but can no longer access because of the client's group restriction
// It must run after the group membership or allowed-group change has been committed
func (s *Service) targetsForLostGroupAccess(ctx context.Context, tx *gorm.DB, userIDs []string, clientID string) ([]target, error) {
	// Require at least one filter, so a caller that computes an empty user list can never match every user of every client
	if len(userIDs) == 0 && clientID == "" {
		return nil, nil
	}

	query := s.targetsQuery(ctx, tx).
		Where("oidc_clients.is_group_restricted = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM oidc_clients_allowed_user_groups ag JOIN user_groups_users ugu ON ugu.user_group_id = ag.user_group_id WHERE ag.oidc_client_id = oidc_clients.id AND ugu.user_id = user_authorized_oidc_clients.user_id)")
	if len(userIDs) > 0 {
		query = query.Where("user_authorized_oidc_clients.user_id IN (?)", userIDs)
	}
	if clientID != "" {
		query = query.Where("user_authorized_oidc_clients.client_id = ?", clientID)
	}

	var targets []target
	err := query.Scan(&targets).Error
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// PrepareUserNotifications resolves, within the given transaction, the logout notifications for users whose access is being revoked
// It exists for callers that delete the users or their authorizations, which are gone once the transaction commits
// The returned function delivers the notifications in the background and must only be called after the transaction has committed
// It is never nil, so callers that treat a failed lookup as non-fatal can call it unconditionally
func (s *Service) PrepareUserNotifications(ctx context.Context, tx *gorm.DB, userIDs []string) (func(), error) {
	targets, err := s.targetsForUsers(ctx, tx, userIDs)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// PrepareAuthorizationNotification resolves, within the given transaction, the logout notification for a single authorization that is being revoked
// The returned function behaves like the one from PrepareUserNotifications
func (s *Service) PrepareAuthorizationNotification(ctx context.Context, tx *gorm.DB, userID string, clientID string) (func(), error) {
	targets, err := s.targetsForAuthorization(ctx, tx, userID, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// PrepareClientNotifications resolves, within the given transaction, the logout notifications for every user of a client that is being deleted
// The returned function behaves like the one from PrepareUserNotifications
func (s *Service) PrepareClientNotifications(ctx context.Context, tx *gorm.DB, clientID string) (func(), error) {
	targets, err := s.targetsForClient(ctx, tx, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// NotifyUser delivers logout tokens to every client the user has authorized that is registered for back-channel logout
// It must be called after the change that revoked the user's access has been committed, and logs instead of failing because delivery is best effort
func (s *Service) NotifyUser(ctx context.Context, userID string) {
	targets, err := s.targetsForUsers(ctx, s.db, []string{userID})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.String("userId", userID), slog.Any("error", err))
		return
	}
	s.notifyClients(ctx, targets)
}

// NotifyUsersLostGroupAccess delivers logout tokens for group-restricted clients that the given users can no longer access
// It must be called after the group membership change has been committed, and logs instead of failing because delivery is best effort
func (s *Service) NotifyUsersLostGroupAccess(ctx context.Context, userIDs []string) {
	s.notifyLostGroupAccess(ctx, userIDs, "")
}

// NotifyClientLostGroupAccess delivers logout tokens for users who have authorized the given group-restricted client but are no longer in any of its allowed groups
// It must be called after the allowed-group change has been committed, and logs instead of failing because delivery is best effort
func (s *Service) NotifyClientLostGroupAccess(ctx context.Context, clientID string) {
	s.notifyLostGroupAccess(ctx, nil, clientID)
}

func (s *Service) notifyLostGroupAccess(ctx context.Context, userIDs []string, clientID string) {
	targets, err := s.targetsForLostGroupAccess(ctx, s.db, userIDs, clientID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.Any("error", err))
		return
	}
	s.notifyClients(ctx, targets)
}

// notifyClients schedules a durable delivery job for each of the given clients, so callers are never blocked on slow or unreachable clients
// It must be called after the change that revoked the user's access has been committed
func (s *Service) notifyClients(ctx context.Context, targets []target) {
	for _, t := range targets {
		// One actor per authorization keeps deliveries for the same user and client serialized, while the actor type's concurrency limit caps the parallel POSTs
		actorID := t.ClientID + ":" + t.UserID
		_, err := s.actors.Dispatch(ctx, ActorType, actorID, methodDeliver, t)
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
	logoutToken, err := s.tokenSigner.GenerateLogoutToken(t.UserID, t.ClientID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, requestTimeout)
	defer cancel()

	body := url.Values{"logout_token": []string{logoutToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.LogoutURL, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("client responded with status %d", res.StatusCode)
	}
	return nil
}
