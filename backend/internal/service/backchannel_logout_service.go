package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

const (
	// backchannelLogoutRequestTimeout bounds each notification POST, so one unreachable client cannot stall the others
	backchannelLogoutRequestTimeout = 10 * time.Second

	// backchannelLogoutConcurrency caps the parallel notification POSTs, so revoking access for many users still finishes in reasonable time without flooding clients
	backchannelLogoutConcurrency = 4
)

// BackchannelLogoutService sends OIDC Back-Channel Logout 1.0 tokens to clients when a user's access is revoked
// Delivery is best effort: failures are logged and never retried
type BackchannelLogoutService struct {
	db         *gorm.DB
	jwtService *JwtService
	httpClient *http.Client
}

func NewBackchannelLogoutService(db *gorm.DB, jwtService *JwtService, httpClient *http.Client) *BackchannelLogoutService {
	// Refuse to follow redirects, as Go would turn the POST into a body-less GET and the logout token would be silently dropped
	// Returning the redirect response instead makes the delivery fail loudly on the status check
	httpClient = httpClientWithCheckRedirect(httpClient, func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	})

	return &BackchannelLogoutService{
		db:         db,
		jwtService: jwtService,
		httpClient: httpClient,
	}
}

// backchannelLogoutTarget is a single client to notify that a user's session should end
type backchannelLogoutTarget struct {
	UserID    string
	ClientID  string
	LogoutURL string
}

// targetsQuery selects the authorizations of clients that are registered for back-channel logout
// Callers narrow it down to the users or the client whose access was revoked
func (s *BackchannelLogoutService) targetsQuery(ctx context.Context, tx *gorm.DB) *gorm.DB {
	return tx.
		WithContext(ctx).
		Model(&model.UserAuthorizedOidcClient{}).
		Select("user_authorized_oidc_clients.user_id", "user_authorized_oidc_clients.client_id", "oidc_clients.backchannel_logout_url AS logout_url").
		Joins("JOIN oidc_clients ON oidc_clients.id = user_authorized_oidc_clients.client_id").
		Where("oidc_clients.backchannel_logout_url IS NOT NULL AND oidc_clients.backchannel_logout_url <> ''")
}

// targetsForUsers returns every client the given users have authorized that is registered for back-channel logout
func (s *BackchannelLogoutService) targetsForUsers(ctx context.Context, tx *gorm.DB, userIDs []string) ([]backchannelLogoutTarget, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var targets []backchannelLogoutTarget
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
func (s *BackchannelLogoutService) targetsForAuthorization(ctx context.Context, tx *gorm.DB, userID string, clientID string) ([]backchannelLogoutTarget, error) {
	var targets []backchannelLogoutTarget
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
func (s *BackchannelLogoutService) targetsForClient(ctx context.Context, tx *gorm.DB, clientID string) ([]backchannelLogoutTarget, error) {
	var targets []backchannelLogoutTarget
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
func (s *BackchannelLogoutService) targetsForLostGroupAccess(ctx context.Context, tx *gorm.DB, userIDs []string, clientID string) ([]backchannelLogoutTarget, error) {
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

	var targets []backchannelLogoutTarget
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
func (s *BackchannelLogoutService) PrepareUserNotifications(ctx context.Context, tx *gorm.DB, userIDs []string) (func(), error) {
	targets, err := s.targetsForUsers(ctx, tx, userIDs)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// PrepareAuthorizationNotification resolves, within the given transaction, the logout notification for a single authorization that is being revoked
// The returned function behaves like the one from PrepareUserNotifications
func (s *BackchannelLogoutService) PrepareAuthorizationNotification(ctx context.Context, tx *gorm.DB, userID string, clientID string) (func(), error) {
	targets, err := s.targetsForAuthorization(ctx, tx, userID, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// PrepareClientNotifications resolves, within the given transaction, the logout notifications for every user of a client that is being deleted
// The returned function behaves like the one from PrepareUserNotifications
func (s *BackchannelLogoutService) PrepareClientNotifications(ctx context.Context, tx *gorm.DB, clientID string) (func(), error) {
	targets, err := s.targetsForClient(ctx, tx, clientID)
	if err != nil {
		return func() {}, err
	}
	return func() { s.notifyClients(ctx, targets) }, nil
}

// NotifyUser delivers logout tokens to every client the user has authorized that is registered for back-channel logout
// It must be called after the change that revoked the user's access has been committed, and logs instead of failing because delivery is best effort
func (s *BackchannelLogoutService) NotifyUser(ctx context.Context, userID string) {
	targets, err := s.targetsForUsers(ctx, s.db, []string{userID})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.String("userId", userID), slog.Any("error", err))
		return
	}
	s.notifyClients(ctx, targets)
}

// NotifyUsersLostGroupAccess delivers logout tokens for group-restricted clients that the given users can no longer access
// It must be called after the group membership change has been committed, and logs instead of failing because delivery is best effort
func (s *BackchannelLogoutService) NotifyUsersLostGroupAccess(ctx context.Context, userIDs []string) {
	s.notifyLostGroupAccess(ctx, userIDs, "")
}

// NotifyClientLostGroupAccess delivers logout tokens for users who have authorized the given group-restricted client but are no longer in any of its allowed groups
// It must be called after the allowed-group change has been committed, and logs instead of failing because delivery is best effort
func (s *BackchannelLogoutService) NotifyClientLostGroupAccess(ctx context.Context, clientID string) {
	s.notifyLostGroupAccess(ctx, nil, clientID)
}

func (s *BackchannelLogoutService) notifyLostGroupAccess(ctx context.Context, userIDs []string, clientID string) {
	targets, err := s.targetsForLostGroupAccess(ctx, s.db, userIDs, clientID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to find clients to notify for back-channel logout", slog.Any("error", err))
		return
	}
	s.notifyClients(ctx, targets)
}

// notifyClients delivers logout tokens to the given clients in the background, so callers are never blocked on slow or unreachable clients
// It must be called after the change that revoked the user's access has been committed
func (s *BackchannelLogoutService) notifyClients(ctx context.Context, targets []backchannelLogoutTarget) {
	if len(targets) == 0 {
		return
	}

	// The caller's request context ends with its HTTP response, so delivery detaches from its cancellation while keeping its values
	go s.notifyClientsSync(context.WithoutCancel(ctx), targets)
}

func (s *BackchannelLogoutService) notifyClientsSync(ctx context.Context, targets []backchannelLogoutTarget) {
	semaphore := make(chan struct{}, backchannelLogoutConcurrency)
	var waitGroup sync.WaitGroup

	for _, target := range targets {
		semaphore <- struct{}{}

		waitGroup.Go(func() {
			defer func() {
				<-semaphore
			}()

			err := s.sendLogoutToken(ctx, target)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to deliver back-channel logout token",
					slog.String("clientId", target.ClientID),
					slog.String("userId", target.UserID),
					slog.String("logoutUrl", target.LogoutURL),
					slog.Any("error", err),
				)
			}
		})
	}

	waitGroup.Wait()
}

func (s *BackchannelLogoutService) sendLogoutToken(parentCtx context.Context, target backchannelLogoutTarget) error {
	logoutToken, err := s.jwtService.GenerateLogoutToken(target.UserID, target.ClientID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, backchannelLogoutRequestTimeout)
	defer cancel()

	body := url.Values{"logout_token": []string{logoutToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.LogoutURL, strings.NewReader(body.Encode()))
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
