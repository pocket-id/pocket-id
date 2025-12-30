package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

const (
	scimUserSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimGroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimContentType = "application/scim+json"
)

const scimErrorBodyLimit = 4096

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

// ScimService handles SCIM provisioning to external service providers.
type ScimService struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewScimService(db *gorm.DB, httpClient *http.Client) *ScimService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	return &ScimService{db: db, httpClient: httpClient}
}

func (s *ScimService) GetServiceProvider(
	ctx context.Context,
	serviceProviderID string,
) (model.ScimServiceProvider, error) {
	var provider model.ScimServiceProvider
	err := s.db.WithContext(ctx).
		Preload("OidcClient").
		Preload("OidcClient.AllowedUserGroups").
		First(&provider, "id = ?", serviceProviderID).
		Error
	if err != nil {
		return model.ScimServiceProvider{}, err
	}
	return provider, nil
}

func (s *ScimService) ListServiceProviders(ctx context.Context) ([]model.ScimServiceProvider, error) {
	var providers []model.ScimServiceProvider
	err := s.db.WithContext(ctx).
		Preload("OidcClient").
		Find(&providers).
		Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *ScimService) CreateServiceProvider(
	ctx context.Context,
	input *dto.ScimServiceProviderCreateDTO) (model.ScimServiceProvider, error) {
	provider := model.ScimServiceProvider{
		Endpoint:     input.Endpoint,
		Token:        input.Token,
		OidcClientID: input.OidcClientID,
	}

	if err := s.db.WithContext(ctx).Create(&provider).Error; err != nil {
		return model.ScimServiceProvider{}, err
	}

	return provider, nil
}

func (s *ScimService) UpdateServiceProvider(ctx context.Context,
	serviceProviderID string,
	input *dto.ScimServiceProviderCreateDTO,
) (model.ScimServiceProvider, error) {
	var provider model.ScimServiceProvider
	err := s.db.WithContext(ctx).
		First(&provider, "id = ?", serviceProviderID).
		Error
	if err != nil {
		return model.ScimServiceProvider{}, err
	}

	provider.Endpoint = input.Endpoint
	provider.Token = input.Token
	provider.OidcClientID = input.OidcClientID

	if err := s.db.WithContext(ctx).Save(&provider).Error; err != nil {
		return model.ScimServiceProvider{}, err
	}

	return provider, nil
}

func (s *ScimService) DeleteServiceProvider(ctx context.Context, serviceProviderID string) error {
	return s.db.WithContext(ctx).
		Delete(&model.ScimServiceProvider{}, "id = ?", serviceProviderID).
		Error
}

func (s *ScimService) SyncServiceProvider(ctx context.Context, serviceProviderID string) error {
	start := time.Now()
	provider, err := s.GetServiceProvider(ctx, serviceProviderID)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Syncing SCIM service provider",
		slog.String("provider_id", provider.ID),
		slog.String("oidc_client_id", provider.OidcClientID),
	)

	allowedGroupIDs := groupIDs(provider.OidcClient.AllowedUserGroups)

	// Load users and groups that should be synced to the SCIM provider
	groups, err := s.groupsForClient(ctx, provider.OidcClient, allowedGroupIDs)
	if err != nil {
		return err
	}
	users, err := s.usersForClient(ctx, provider.OidcClient, allowedGroupIDs)
	if err != nil {
		return err
	}

	// Load users and groups that already exist in the SCIM provider
	userResources, err := listScimResources[dto.ScimUser](s, ctx, provider, "/Users")
	if err != nil {
		return err
	}
	groupResources, err := listScimResources[dto.ScimGroup](s, ctx, provider, "/Groups")
	if err != nil {
		return err
	}

	var errs []error
	var userStats scimSyncStats
	var groupStats scimSyncStats

	// Sync users first, so that groups can reference them
	if stats, err := s.syncUsers(ctx, provider, users, &userResources); err != nil {
		errs = append(errs, err)
		userStats = stats
	} else {
		userStats = stats
	}

	stats, err := s.syncGroups(ctx, provider, groups, groupResources.Resources, userResources.Resources)
	if err != nil {
		errs = append(errs, err)
		groupStats = stats
	} else {
		groupStats = stats
	}

	if len(errs) > 0 {
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
		)
		return errors.Join(errs...)
	}

	provider.LastSyncedAt = utils.Ptr(datatype.DateTime(time.Now()))
	if err := s.db.WithContext(ctx).Save(&provider).Error; err != nil {
		return err
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

func (s *ScimService) syncUsers(
	ctx context.Context,
	provider model.ScimServiceProvider,
	users []model.User,
	resourceList *dto.ScimListResponse[dto.ScimUser],
) (stats scimSyncStats, err error) {
	var errs []error

	// Update or create users
	for _, u := range users {
		existing := getResourceByExternalID[dto.ScimUser](u.ID, resourceList.Resources)

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

	// Delete users that are present in SCIM provider but not locally.
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

func (s *ScimService) syncGroups(
	ctx context.Context,
	provider model.ScimServiceProvider,
	groups []model.UserGroup,
	remoteGroups []dto.ScimGroup,
	userResources []dto.ScimUser,
) (stats scimSyncStats, err error) {
	var errs []error

	// Update or create groups
	for _, g := range groups {
		existing := getResourceByExternalID[dto.ScimGroup](g.ID, remoteGroups)

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

func (s *ScimService) syncUser(ctx context.Context,
	provider model.ScimServiceProvider,
	user model.User,
	userResource *dto.ScimUser,
) (scimSyncAction, *dto.ScimUser, error) {
	// If user is not allowed for the client, delete it from SCIM provider
	if userResource != nil && !IsUserGroupAllowedToAuthorize(user, provider.OidcClient) {
		return scimActionDeleted, nil, s.deleteScimResource(ctx, provider, fmt.Sprintf("/Users/%s", url.PathEscape(userResource.ID)))
	}

	payload := dto.ScimUser{
		ScimResourceData: dto.ScimResourceData{
			Schemas:    []string{scimUserSchema},
			ExternalID: user.ID,
		},
		UserName: user.Username,
		Name: &dto.ScimName{
			GivenName:  user.FirstName,
			FamilyName: user.LastName,
		},
		Display: user.DisplayName,
		Active:  !user.Disabled,
	}

	if user.Email != nil {
		payload.Emails = []dto.ScimEmail{{
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

func (s *ScimService) syncGroup(
	ctx context.Context,
	provider model.ScimServiceProvider,
	group model.UserGroup,
	groupResource *dto.ScimGroup,
	userResources []dto.ScimUser,
) (scimSyncAction, error) {
	// If group is not allowed for the client, delete it from SCIM provider
	if groupResource != nil && !groupAllowedForClient(group.ID, provider.OidcClient) {
		return scimActionDeleted, s.deleteScimResource(ctx, provider, fmt.Sprintf("/Groups/%s", url.PathEscape(groupResource.GetID())))
	}

	// Prepare group members
	members := make([]dto.ScimGroupMember, len(group.Users))
	for i, user := range group.Users {
		userResource := getResourceByExternalID[dto.ScimUser](user.ID, userResources)
		if userResource == nil {
			// Groups depend on user IDs already being provisioned
			return scimActionNone, fmt.Errorf("cannot sync group %s: user %s is not provisioned in SCIM provider", group.ID, user.ID)
		}

		members[i] = dto.ScimGroupMember{
			Value: userResource.GetID(),
		}
	}

	groupPayload := dto.ScimGroup{
		ScimResourceData: dto.ScimResourceData{
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

func (s *ScimService) groupsForClient(
	ctx context.Context,
	client model.OidcClient,
	allowedGroupIDs []string,
) ([]model.UserGroup, error) {
	var groups []model.UserGroup

	query := s.db.WithContext(ctx).Preload("Users").Model(&model.UserGroup{})
	if client.IsGroupRestricted {
		if len(allowedGroupIDs) == 0 {
			return groups, nil
		}
		query = query.Where("id IN ?", allowedGroupIDs)
	}

	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *ScimService) usersForClient(
	ctx context.Context,
	client model.OidcClient,
	allowedGroupIDs []string,
) ([]model.User, error) {
	var users []model.User

	query := s.db.WithContext(ctx).Model(&model.User{})
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

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func getResourceByExternalID[T dto.ScimResource](externalID string, resource []T) *T {
	for i := range resource {
		if resource[i].GetExternalID() == externalID {
			return &resource[i]
		}
	}
	return nil
}

func listScimResources[T any](
	s *ScimService,
	ctx context.Context,
	provider model.ScimServiceProvider,
	path string,
) (result dto.ScimListResponse[T], err error) {
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
			return dto.ScimListResponse[T]{}, err
		}

		if err := ensureScimStatus(ctx, resp, provider, http.StatusOK); err != nil {
			return dto.ScimListResponse[T]{}, err
		}

		var page dto.ScimListResponse[T]
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return dto.ScimListResponse[T]{}, fmt.Errorf("failed to decode SCIM list response: %w", err)
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

func createScimResource[T dto.ScimResource](
	s *ScimService,
	ctx context.Context,
	provider model.ScimServiceProvider,
	path string, payload T) (*T, error) {
	resp, err := s.scimRequest(ctx, provider, http.MethodPost, path, payload, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusCreated); err != nil {
		return nil, err
	}

	var resource T
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return nil, fmt.Errorf("failed to decode SCIM create response: %w", err)
	}

	return &resource, nil
}

func updateScimResource[T dto.ScimResource](
	s *ScimService,
	ctx context.Context,
	provider model.ScimServiceProvider,
	path string,
	payload T,
) (*T, error) {
	resp, err := s.scimRequest(ctx, provider, http.MethodPut, path, payload, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusCreated); err != nil {
		return nil, err
	}

	var resource T
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return nil, fmt.Errorf("failed to decode SCIM update response: %w", err)
	}

	return &resource, nil
}

func (s *ScimService) deleteScimResource(ctx context.Context, provider model.ScimServiceProvider, path string) error {
	resp, err := s.scimRequest(ctx, provider, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return ensureScimStatus(ctx, resp, provider, http.StatusOK, http.StatusNoContent)
}

func (s *ScimService) scimRequest(
	ctx context.Context,
	provider model.ScimServiceProvider,
	method,
	path string,
	payload any,
	queryParams map[string]string,
) (*http.Response, error) {
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
		if provider.Token != "" {
			req.Header.Set("Authorization", "Bearer "+provider.Token)
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
		if err := utils.SleepWithContext(ctx, retryDelay); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("scim request retry attempts exceeded")
}

func scimRetryDelay(retryAfter string, attempt int) time.Duration {
	// Respect Retry-After when provided
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
		if t, err := http.ParseTime(retryAfter); err == nil {
			if delay := time.Until(t); delay > 0 {
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

func ensureScimStatus(
	ctx context.Context,
	resp *http.Response,
	provider model.ScimServiceProvider,
	allowedStatuses ...int) error {
	for _, status := range allowedStatuses {
		if resp.StatusCode == status {
			return nil
		}
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
