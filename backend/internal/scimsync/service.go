package scimsync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	scimUserSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimGroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimContentType = "application/scim+json"
)

const (
	scimErrorBodyLimit      = 4 << 10 // 4KB
	syncProviderConcurrency = 4
)

type scimSyncAction int

const (
	scimActionNone scimSyncAction = iota
	scimActionCreated
	scimActionUpdated
	scimActionDeleted
)

type scimSyncStats struct {
	Created int
	Updated int
	Deleted int
}

// Service handles SCIM provisioning to external service providers
type Service struct {
	db         *gorm.DB
	httpClient *http.Client
}

func newService(db *gorm.DB, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Service{
		db:         db,
		httpClient: httpClient,
	}
}

func (s *Service) GetServiceProvider(ctx context.Context, serviceProviderID string) (ServiceProvider, error) {
	return getServiceProvider(ctx, s.db, serviceProviderID)
}

func getServiceProvider(ctx context.Context, db *gorm.DB, serviceProviderID string) (ServiceProvider, error) {
	var provider ServiceProvider
	err := db.WithContext(ctx).
		Preload("OidcClient").
		Preload("OidcClient.AllowedUserGroups").
		First(&provider, "id = ?", serviceProviderID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ServiceProvider{}, apperror.NotFound("SCIM service provider")
	} else if err != nil {
		return ServiceProvider{}, err
	}

	return provider, nil
}

func (s *Service) ListServiceProviders(ctx context.Context) ([]ServiceProvider, error) {
	var providers []ServiceProvider
	err := s.db.WithContext(ctx).
		Select("id").
		Find(&providers).
		Error
	if err != nil {
		return nil, err
	}

	return providers, nil
}

func (s *Service) GetServiceProviderByClient(ctx context.Context, clientID string) (ServiceProvider, error) {
	var provider ServiceProvider
	err := s.db.WithContext(ctx).
		First(&provider, "oidc_client_id = ?", clientID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ServiceProvider{}, apperror.NotFound("SCIM service provider")
	} else if err != nil {
		return ServiceProvider{}, err
	}

	return provider, nil
}

func (s *Service) CreateServiceProvider(ctx context.Context, input *ScimServiceProviderCreateDTO) (ServiceProvider, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	err := ensureScimOIDCClientExists(ctx, tx, input.OidcClientID)
	if err != nil {
		return ServiceProvider{}, err
	}

	provider := ServiceProvider{
		Endpoint:     input.Endpoint,
		Token:        datatype.EncryptedString(input.Token),
		OidcClientID: input.OidcClientID,
	}

	err = tx.WithContext(ctx).Create(&provider).Error
	if err != nil {
		return ServiceProvider{}, fmt.Errorf("error creating service provider: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return ServiceProvider{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return provider, nil
}

func (s *Service) UpdateServiceProvider(ctx context.Context, serviceProviderID string, input *ScimServiceProviderCreateDTO) (ServiceProvider, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var provider ServiceProvider
	err := tx.WithContext(ctx).
		First(&provider, "id = ?", serviceProviderID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ServiceProvider{}, apperror.NotFound("SCIM service provider")
	} else if err != nil {
		return ServiceProvider{}, fmt.Errorf("error loading SCIM service provider: %w", err)
	}

	err = ensureScimOIDCClientExists(ctx, tx, input.OidcClientID)
	if err != nil {
		return ServiceProvider{}, err
	}

	provider.Endpoint = input.Endpoint
	provider.Token = datatype.EncryptedString(input.Token)
	provider.OidcClientID = input.OidcClientID

	err = tx.WithContext(ctx).Save(&provider).Error
	if err != nil {
		return ServiceProvider{}, fmt.Errorf("error saving SCIM service provider: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return ServiceProvider{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return provider, nil
}

func ensureScimOIDCClientExists(ctx context.Context, db *gorm.DB, clientID string) error {
	var client model.OidcClient
	err := db.WithContext(ctx).
		Select("id").
		First(&client, "id = ?", clientID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.NotFound("OIDC client")
	}

	return err
}

func (s *Service) DeleteServiceProvider(ctx context.Context, serviceProviderID string) error {
	result := s.db.
		WithContext(ctx).
		Delete(&ServiceProvider{}, "id = ?", serviceProviderID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("SCIM service provider")
	}

	return nil
}

func (s *Service) SyncAll(ctx context.Context) error {
	providers, err := s.ListServiceProviders(ctx)
	if err != nil {
		return err
	}

	return syncServiceProviders(ctx, providers, s.SyncServiceProvider)
}

func (s *Service) SyncServiceProvider(ctx context.Context, serviceProviderID string) error {
	start := time.Now()

	// Load one consistent local snapshot and release the transaction before making remote requests
	snapshot, err := s.loadSyncSnapshot(ctx, serviceProviderID)
	if err != nil {
		return err
	}
	provider := snapshot.provider

	slog.InfoContext(ctx, "Syncing SCIM service provider",
		slog.String("provider_id", provider.ID),
		slog.String("oidc_client_id", provider.OidcClientID),
	)

	// Load users and groups that already exist in the SCIM provider
	userResources, err := listScimResources[ScimUser](s, ctx, provider, "/Users")
	if err != nil {
		return err
	}
	groupResources, err := listScimResources[ScimGroup](s, ctx, provider, "/Groups")
	if err != nil {
		return err
	}

	var errs []error

	// Sync users first, so that groups can reference them
	userStats, err := s.syncUsers(ctx, provider, snapshot.users, &userResources)
	if err != nil {
		errs = append(errs, err)
	}

	groupStats, err := s.syncGroups(ctx, provider, snapshot.groups, groupResources.Resources, userResources.Resources)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
		slog.WarnContext(ctx, "SCIM sync completed with errors",
			slog.String("provider_id", provider.ID),
			slog.Int("error_count", len(errs)),
			slog.Int("users_created", userStats.Created),
			slog.Int("users_updated", userStats.Updated),
			slog.Int("users_deleted", userStats.Deleted),
			slog.Int("groups_created", groupStats.Created),
			slog.Int("groups_updated", groupStats.Updated),
			slog.Int("groups_deleted", groupStats.Deleted),
			slog.Duration("duration", time.Since(start)),
			slog.Any("error", err),
		)
		return err
	}

	lastSyncedAt := datatype.DateTime(time.Now())
	result := s.db.WithContext(ctx).
		Model(&ServiceProvider{}).
		Where("id = ?", provider.ID).
		Update("last_synced_at", &lastSyncedAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("SCIM service provider")
	}

	slog.InfoContext(ctx, "SCIM sync completed",
		slog.String("provider_id", provider.ID),
		slog.Int("users_created", userStats.Created),
		slog.Int("users_updated", userStats.Updated),
		slog.Int("users_deleted", userStats.Deleted),
		slog.Int("groups_created", groupStats.Created),
		slog.Int("groups_updated", groupStats.Updated),
		slog.Int("groups_deleted", groupStats.Deleted),
		slog.Duration("duration", time.Since(start)),
	)

	return nil
}

type syncSnapshot struct {
	provider ServiceProvider
	users    []model.User
	groups   []model.UserGroup
}

// loadSyncSnapshot reads all local inputs from one point in time without holding the transaction across remote SCIM calls
func (s *Service) loadSyncSnapshot(ctx context.Context, serviceProviderID string) (snapshot syncSnapshot, oErr error) {
	oErr = s.db.
		WithContext(ctx).
		Transaction(
			func(tx *gorm.DB) (err error) {
				snapshot.provider, err = getServiceProvider(ctx, tx, serviceProviderID)
				if err != nil {
					return err
				}

				allowedGroupIDs := groupIDs(snapshot.provider.OidcClient.AllowedUserGroups)
				snapshot.groups, err = groupsForClient(ctx, tx, snapshot.provider.OidcClient, allowedGroupIDs)
				if err != nil {
					return err
				}

				snapshot.users, err = usersForClient(ctx, tx, snapshot.provider.OidcClient, allowedGroupIDs)
				return err
			},
			syncSnapshotTxOptions(s.db.Name()),
		)
	if oErr != nil {
		return syncSnapshot{}, oErr
	}

	return snapshot, nil
}

// syncSnapshotTxOptions pins a consistent read snapshot without taking SQLite's configured immediate write lock
func syncSnapshotTxOptions(provider string) *sql.TxOptions {
	opts := &sql.TxOptions{ReadOnly: true}
	if provider == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}

	return opts
}

func syncServiceProviders(ctx context.Context, providers []ServiceProvider, syncProvider func(context.Context, string) error) error {
	// Bound concurrency so several independent providers make progress without overwhelming the database or network
	semaphore := make(chan struct{}, syncProviderConcurrency)
	errs := make([]error, len(providers))
	var waitGroup sync.WaitGroup

	// Start each provider when a slot is available and retain its error in deterministic provider order
providerLoop:
	for i, provider := range providers {
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			errs[i] = ctx.Err()
			break providerLoop
		}

		waitGroup.Go(func() {
			defer func() {
				<-semaphore
			}()

			err := syncProvider(ctx, provider.ID)
			if err != nil {
				errs[i] = fmt.Errorf("failed to sync SCIM provider %s: %w", provider.ID, err)
			}
		})
	}

	// Wait for every started provider so one failure never prevents the remaining providers from synchronizing
	waitGroup.Wait()

	return errors.Join(errs...)
}

func (s *Service) syncUsers(ctx context.Context, provider ServiceProvider, users []model.User, resourceList *ScimListResponse[ScimUser]) (stats scimSyncStats, err error) {
	var errs []error

	// Update or create users
	for _, u := range users {
		existing := getResourceByExternalID(u.ID, resourceList.Resources)

		action, created, err := s.syncUser(ctx, provider, u, existing)
		if created != nil && existing == nil {
			resourceList.Resources = append(resourceList.Resources, *created)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Update stats based on action taken by syncUser
		switch action {
		case scimActionCreated:
			stats.Created++
		case scimActionUpdated:
			stats.Updated++
		case scimActionDeleted:
			stats.Deleted++
		case scimActionNone:
		}
	}

	// Delete users that are present in SCIM provider but not locally
	userSet := make(map[string]struct{})
	for _, u := range users {
		userSet[u.ID] = struct{}{}
	}

	for _, r := range resourceList.Resources {
		if _, ok := userSet[r.ExternalID]; !ok {
			if err := s.deleteScimResource(ctx, provider, "/Users/"+url.PathEscape(r.ID)); err != nil {
				errs = append(errs, err)
			} else {
				stats.Deleted++
			}
		}
	}

	return stats, errors.Join(errs...)
}

func (s *Service) syncGroups(ctx context.Context, provider ServiceProvider, groups []model.UserGroup, remoteGroups []ScimGroup, userResources []ScimUser) (stats scimSyncStats, err error) {
	var errs []error

	// Update or create groups
	for _, g := range groups {
		existing := getResourceByExternalID(g.ID, remoteGroups)

		action, err := s.syncGroup(ctx, provider, g, existing, userResources)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Update stats based on action taken by syncGroup
		switch action {
		case scimActionCreated:
			stats.Created++
		case scimActionUpdated:
			stats.Updated++
		case scimActionDeleted:
			stats.Deleted++
		case scimActionNone:
		}

	}

	// Delete groups that are present in SCIM provider but not locally
	groupSet := make(map[string]struct{})
	for _, g := range groups {
		groupSet[g.ID] = struct{}{}
	}

	for _, r := range remoteGroups {
		if _, ok := groupSet[r.ExternalID]; !ok {
			if err := s.deleteScimResource(ctx, provider, "/Groups/"+url.PathEscape(r.GetID())); err != nil {
				errs = append(errs, err)
			} else {
				stats.Deleted++
			}
		}
	}

	return stats, errors.Join(errs...)
}

func (s *Service) syncUser(ctx context.Context, provider ServiceProvider, user model.User, userResource *ScimUser) (scimSyncAction, *ScimUser, error) {
	// If user is not allowed for the client, delete it from SCIM provider
	if userResource != nil && !oidc.IsUserGroupAllowedToAuthorize(user, provider.OidcClient) {
		return scimActionDeleted, nil, s.deleteScimResource(ctx, provider, fmt.Sprintf("/Users/%s", url.PathEscape(userResource.ID)))
	}

	payload := ScimUser{
		ScimResourceData: ScimResourceData{
			Schemas:    []string{scimUserSchema},
			ExternalID: user.ID,
		},
		UserName: user.Username,
		Name: &ScimName{
			GivenName:  user.FirstName,
			FamilyName: user.LastName,
		},
		Display: user.DisplayName,
		Active:  !user.Disabled,
	}

	if user.Email != nil {
		payload.Emails = []ScimEmail{{
			Value:   *user.Email,
			Primary: true,
		}}
	}

	// If the user exists on the SCIM provider, and it has been modified, update it
	if userResource != nil {
		if user.LastModified().Before(userResource.GetMeta().LastModified) {
			return scimActionNone, nil, nil
		}
		path := fmt.Sprintf("/Users/%s", url.PathEscape(userResource.GetID()))
		userResource, err := updateScimResource(s, ctx, provider, path, payload)
		if err != nil {
			return scimActionNone, nil, err
		}
		return scimActionUpdated, userResource, nil
	}

	// Otherwise, create a new SCIM user
	userResource, err := createScimResource(s, ctx, provider, "/Users", payload)
	if err != nil {
		return scimActionNone, nil, err
	}

	return scimActionCreated, userResource, nil
}

func (s *Service) syncGroup(ctx context.Context, provider ServiceProvider, group model.UserGroup, groupResource *ScimGroup, userResources []ScimUser) (scimSyncAction, error) {
	// If group is not allowed for the client, delete it from SCIM provider
	if groupResource != nil && !groupAllowedForClient(group.ID, provider.OidcClient) {
		err := s.deleteScimResource(ctx, provider, fmt.Sprintf("/Groups/%s", url.PathEscape(groupResource.GetID())))
		if err != nil {
			return scimActionNone, err
		}
		return scimActionDeleted, nil
	}

	// Prepare group members
	members := make([]ScimGroupMember, len(group.Users))
	for i, user := range group.Users {
		userResource := getResourceByExternalID(user.ID, userResources)
		if userResource == nil {
			// Groups depend on user IDs already being provisioned
			return scimActionNone, fmt.Errorf("cannot sync group %s: user %s is not provisioned in SCIM provider", group.ID, user.ID)
		}

		members[i] = ScimGroupMember{
			Value: userResource.GetID(),
		}
	}

	groupPayload := ScimGroup{
		ScimResourceData: ScimResourceData{
			Schemas:    []string{scimGroupSchema},
			ExternalID: group.ID,
		},
		Display: group.FriendlyName,
		Members: members,
	}

	// If the group exists on the SCIM provider, and it has been modified, update it
	if groupResource != nil {
		if group.LastModified().Before(groupResource.GetMeta().LastModified) {
			return scimActionNone, nil
		}
		path := fmt.Sprintf("/Groups/%s", url.PathEscape(groupResource.GetID()))
		_, err := updateScimResource(s, ctx, provider, path, groupPayload)
		if err != nil {
			return scimActionNone, err
		}
		return scimActionUpdated, nil
	}

	// Otherwise, create a new SCIM group
	_, err := createScimResource(s, ctx, provider, "/Groups", groupPayload)
	if err != nil {
		return scimActionNone, err
	}

	return scimActionCreated, nil
}

func groupAllowedForClient(groupID string, client model.OidcClient) bool {
	if !client.IsGroupRestricted {
		return true
	}

	for _, allowedGroup := range client.AllowedUserGroups {
		if allowedGroup.ID == groupID {
			return true
		}
	}

	return false
}

func groupIDs(groups []model.UserGroup) []string {
	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	return ids
}

func groupsForClient(ctx context.Context, db *gorm.DB, client model.OidcClient, allowedGroupIDs []string) ([]model.UserGroup, error) {
	var groups []model.UserGroup

	query := db.WithContext(ctx).Preload("Users").Model(&model.UserGroup{})
	if client.IsGroupRestricted {
		if len(allowedGroupIDs) == 0 {
			return groups, nil
		}
		query = query.Where("id IN ?", allowedGroupIDs)
	}

	err := query.Find(&groups).Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func usersForClient(ctx context.Context, db *gorm.DB, client model.OidcClient, allowedGroupIDs []string) ([]model.User, error) {
	var users []model.User

	query := db.WithContext(ctx).Model(&model.User{})
	if client.IsGroupRestricted {
		if len(allowedGroupIDs) == 0 {
			return users, nil
		}
		query = query.
			Joins("JOIN user_groups_users ON users.id = user_groups_users.user_id").
			Where("user_groups_users.user_group_id IN ?", allowedGroupIDs).
			Select("users.*").
			Distinct()
	}

	query = query.Preload("UserGroups")

	err := query.Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func getResourceByExternalID[T ScimResource](externalID string, resource []T) *T {
	for i := range resource {
		if resource[i].GetExternalID() == externalID {
			return &resource[i]
		}
	}
	return nil
}

func listScimResources[T any](s *Service, ctx context.Context, provider ServiceProvider, path string) (result ScimListResponse[T], err error) {
	startIndex := 1
	count := 1000

	for {
		// Use SCIM pagination to avoid missing resources on large providers
		queryParams := map[string]string{
			"startIndex": strconv.Itoa(startIndex),
			"count":      strconv.Itoa(count),
		}

		resp, err := s.scimRequest(ctx, provider, http.MethodGet, path, nil, queryParams)
		if err != nil {
			return ScimListResponse[T]{}, err
		}

		err = ensureScimStatus(ctx, resp, provider, http.StatusOK)
		if err != nil {
			return ScimListResponse[T]{}, err
		}

		var page ScimListResponse[T]
		err = json.NewDecoder(resp.Body).Decode(&page)
		if err != nil {
			return ScimListResponse[T]{}, fmt.Errorf("failed to decode SCIM list response: %w", err)
		}

		resp.Body.Close()

		// Initialize metadata only once
		if result.TotalResults == 0 {
			result.TotalResults = page.TotalResults
		}

		result.Resources = append(result.Resources, page.Resources...)

		// If we've fetched everything, stop
		if len(result.Resources) >= page.TotalResults || len(page.Resources) == 0 {
			break
		}

		startIndex += page.ItemsPerPage
	}

	result.ItemsPerPage = len(result.Resources)
	return result, nil
}

func createScimResource[T ScimResource](s *Service, ctx context.Context, provider ServiceProvider, path string, payload T) (*T, error) {
	resp, err := s.scimRequest(ctx, provider, http.MethodPost, path, payload, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var resource T
	err = json.NewDecoder(resp.Body).Decode(&resource)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SCIM create response: %w", err)
	}

	return &resource, nil
}

func updateScimResource[T ScimResource](s *Service, ctx context.Context, provider ServiceProvider, path string, payload T) (*T, error) {
	resp, err := s.scimRequest(ctx, provider, http.MethodPut, path, payload, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var resource T
	err = json.NewDecoder(resp.Body).Decode(&resource)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SCIM update response: %w", err)
	}

	return &resource, nil
}

func (s *Service) deleteScimResource(ctx context.Context, provider ServiceProvider, path string) error {
	resp, err := s.scimRequest(ctx, provider, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	err = ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) scimRequest(ctx context.Context, provider ServiceProvider, method, path string, payload any, queryParams map[string]string) (*http.Response, error) {
	urlString, err := scimURL(provider.Endpoint, path, queryParams)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode SCIM payload: %w", err)
		}
		bodyBytes = encoded
	}

	retryAttempts := 3
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		var body io.Reader
		if bodyBytes != nil {
			body = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, urlString, body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", scimContentType)
		if payload != nil {
			req.Header.Set("Content-Type", scimContentType)
		}
		token := string(provider.Token)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		slog.Debug("Sending SCIM request",
			slog.String("method", method),
			slog.String("url", urlString),
			slog.String("provider_id", provider.ID),
		)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		// Only retry on 429 to avoid masking other errors
		if resp.StatusCode != http.StatusTooManyRequests || attempt == retryAttempts {
			return resp, nil
		}

		retryDelay := scimRetryDelay(resp.Header.Get("Retry-After"), attempt)
		slog.WarnContext(ctx, "SCIM provider rate-limited, retrying",
			slog.String("provider_id", provider.ID),
			slog.String("method", method),
			slog.String("url", urlString),
			slog.Int("attempt", attempt),
			slog.Duration("retry_after", retryDelay),
		)

		resp.Body.Close()
		err = utils.SleepWithContext(ctx, retryDelay)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("scim request retry attempts exceeded")
}

func scimRetryDelay(retryAfter string, attempt int) time.Duration {
	// Respect Retry-After when provided
	if retryAfter != "" {
		seconds, err := strconv.Atoi(retryAfter)
		if err == nil {
			return time.Duration(seconds) * time.Second
		}
		t, err := http.ParseTime(retryAfter)
		if err == nil {
			delay := time.Until(t)
			if delay > 0 {
				return delay
			}
		}
	}

	// Exponential backoff otherwise
	maxDelay := 10 * time.Second
	delay := 500 * time.Millisecond * (time.Duration(1) << (attempt - 1)) //nolint:gosec // attempt is bounded 1-3
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func scimURL(endpoint, p string, queryParams map[string]string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid scim endpoint: %w", err)
	}

	u.Path = path.Join(strings.TrimRight(u.Path, "/"), p)

	q := u.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func ensureScimStatus(ctx context.Context, resp *http.Response, provider ServiceProvider, allowedStatuses ...int) error {
	if slices.Contains(allowedStatuses, resp.StatusCode) {
		return nil
	}

	body := readScimErrorBody(resp.Body)

	slog.ErrorContext(ctx, "SCIM request failed",
		slog.String("provider_id", provider.ID),
		slog.String("method", resp.Request.Method),
		slog.String("url", resp.Request.URL.String()),
		slog.Int("status", resp.StatusCode),
		slog.String("response_body", body),
	)

	return fmt.Errorf("scim request failed with status %d: %s", resp.StatusCode, body)
}

func readScimErrorBody(body io.Reader) string {
	payload, err := io.ReadAll(io.LimitReader(body, scimErrorBodyLimit))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(payload))
}
