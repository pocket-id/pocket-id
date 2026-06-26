package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// reservedPermissionKeys are the scope and claim names owned by Pocket ID's built-in identity layer
// A custom API permission must not reuse one, otherwise its scope string would collide with a standard OIDC scope or claim
var reservedPermissionKeys = map[string]struct{}{
	"openid":         {},
	"profile":        {},
	"email":          {},
	"email_verified": {},
	"groups":         {},
	"offline_access": {},
}

// permissionKeyPattern restricts permission keys to RFC 6749 scope-token characters, which are printable ASCII without space, double-quote or backslash
// This keeps a key safe as a space-delimited value in the token scope claim and free of the control character used to qualify consent records
var permissionKeyPattern = regexp.MustCompile(`^[\x21\x23-\x5B\x5D-\x7E]+$`)

// Service holds the business logic for managing APIs and their permissions
type Service struct {
	db     *gorm.DB
	issuer string
}

func newService(db *gorm.DB, issuer string) *Service {
	return &Service{db: db, issuer: issuer}
}

// isIssuerAudience reports whether the audience refers to Pocket ID itself (the issuer)
// A custom API must not claim the issuer as its audience, otherwise its tokens would be indistinguishable from Pocket ID's own identity tokens
func isIssuerAudience(audience, issuer string) bool {
	return issuer != "" && strings.EqualFold(strings.TrimRight(audience, "/"), strings.TrimRight(issuer, "/"))
}

func (s *Service) List(ctx context.Context, search string, listRequestOptions utils.ListRequestOptions) (apis []API, response utils.PaginationResponse, err error) {
	query := s.db.
		WithContext(ctx).
		Preload("Permissions").
		Model(&API{})

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR audience LIKE ?", like, like)
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &apis)
	return apis, response, err
}

func (s *Service) Get(ctx context.Context, id string) (api API, err error) {
	err = s.db.
		WithContext(ctx).
		Preload("Permissions").
		Where("id = ?", id).
		First(&api).
		Error
	return api, err
}

func (s *Service) Create(ctx context.Context, input apiCreateDto) (api API, err error) {
	// Reject the issuer as an audience so a custom API cannot impersonate Pocket ID's own identity tokens
	if isIssuerAudience(input.Audience, s.issuer) {
		return API{}, &common.ValidationError{Message: "the audience is reserved by Pocket ID and cannot be used for a custom API"}
	}

	api = API{
		Name:     input.Name,
		Audience: input.Audience,
	}

	err = s.db.WithContext(ctx).Create(&api).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return API{}, &common.AlreadyInUseError{Property: "audience"}
		}
		return API{}, err
	}

	return api, nil
}

func (s *Service) Update(ctx context.Context, id string, input apiUpdateDto) (api API, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err = s.getInternal(ctx, id, tx)
	if err != nil {
		return API{}, err
	}

	api.Name = input.Name
	api.UpdatedAt = new(datatype.DateTime(time.Now()))

	err = tx.WithContext(ctx).Save(&api).Error
	if err != nil {
		return API{}, err
	}

	if err = tx.Commit().Error; err != nil {
		return API{}, err
	}

	return api, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err := s.getInternal(ctx, id, tx)
	if err != nil {
		return err
	}

	if err = s.deletePermissionsInternal(ctx, tx, collectIDs(api.Permissions)); err != nil {
		return err
	}

	if err = tx.WithContext(ctx).Delete(&API{}, "id = ?", id).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

// UpdatePermissions replaces the full permission set of an API, matching existing permissions by key
// Unchanged keys keep their grants, removed keys and their client grants are deleted, and new keys are inserted
func (s *Service) UpdatePermissions(ctx context.Context, id string, input apiPermissionsUpdateDto) (api API, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err = s.getInternal(ctx, id, tx)
	if err != nil {
		return API{}, err
	}

	// Reject keys with invalid characters or that collide with Pocket ID's reserved scopes and claims before persisting anything
	for _, permission := range input.Permissions {
		if !permissionKeyPattern.MatchString(permission.Key) {
			return API{}, &common.ValidationError{Message: fmt.Sprintf("the permission key %q contains invalid characters", permission.Key)}
		}
		if _, reserved := reservedPermissionKeys[strings.ToLower(permission.Key)]; reserved {
			return API{}, &common.ValidationError{Message: fmt.Sprintf("the permission key %q is reserved by Pocket ID", permission.Key)}
		}
	}

	existing := make(map[string]Permission, len(api.Permissions))
	for _, p := range api.Permissions {
		existing[p.Key] = p
	}

	wanted := make(map[string]apiPermissionInputDto, len(input.Permissions))
	var removedIDs []string
	for _, in := range input.Permissions {
		wanted[in.Key] = in
	}

	// Delete permissions whose key is no longer wanted
	for key, p := range existing {
		if _, ok := wanted[key]; !ok {
			removedIDs = append(removedIDs, p.ID)
		}
	}
	if err = s.deletePermissionsInternal(ctx, tx, removedIDs); err != nil {
		return API{}, err
	}

	// Insert new keys and update the display fields of existing ones
	for key, in := range wanted {
		if cur, ok := existing[key]; ok {
			err = tx.WithContext(ctx).
				Model(&Permission{}).
				Where("id = ?", cur.ID).
				Updates(map[string]any{"name": in.Name, "description": in.Description}).
				Error
			if err != nil {
				return API{}, err
			}
			continue
		}

		newPermission := Permission{
			APIID:       api.ID,
			Key:         in.Key,
			Name:        in.Name,
			Description: in.Description,
		}
		if err = tx.WithContext(ctx).Create(&newPermission).Error; err != nil {
			return API{}, err
		}
	}

	err = tx.WithContext(ctx).
		Model(&API{}).
		Where("id = ?", api.ID).
		Update("updated_at", new(datatype.DateTime(time.Now()))).
		Error
	if err != nil {
		return API{}, err
	}

	if err = tx.Commit().Error; err != nil {
		return API{}, err
	}

	return s.Get(ctx, id)
}

// GetClientAllowedPermissionIDs returns the IDs of the API permissions a client is allowed to request
// Only custom-API permissions are tracked here because the identity scopes are freely requestable by every client
func (s *Service) GetClientAllowedPermissionIDs(ctx context.Context, clientID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).
		Model(&OidcClientAllowedAPIPermission{}).
		Where("oidc_client_id = ?", clientID).
		Pluck("api_permission_id", &ids).
		Error
	return ids, err
}

// SetClientAllowedPermissions replaces the client's API-access allow-list with the given permission IDs
// Unknown permission IDs are ignored
// It returns the IDs that were actually applied
func (s *Service) SetClientAllowedPermissions(ctx context.Context, clientID string, permissionIDs []string) (applied []string, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Ensure the client exists so callers get a 404 for an unknown client
	var client model.OidcClient
	if err = tx.WithContext(ctx).Select("id").Where("id = ?", clientID).First(&client).Error; err != nil {
		return nil, err
	}

	applied, err = s.filterAssignablePermissionIDs(ctx, tx, permissionIDs)
	if err != nil {
		return nil, err
	}

	// Replace the allow-list for this client
	err = tx.WithContext(ctx).
		Where("oidc_client_id = ?", clientID).
		Delete(&OidcClientAllowedAPIPermission{}).
		Error
	if err != nil {
		return nil, err
	}

	if len(applied) > 0 {
		rows := make([]OidcClientAllowedAPIPermission, len(applied))
		for i, permissionID := range applied {
			rows[i] = OidcClientAllowedAPIPermission{OidcClientID: clientID, APIPermissionID: permissionID}
		}
		if err = tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return nil, err
		}
	}

	if err = tx.Commit().Error; err != nil {
		return nil, err
	}

	return applied, nil
}

// ClientAPIScopesAndAudiences returns the permission keys a client may request and the distinct audiences of the custom APIs those permissions belong to
// The OIDC module uses this to widen fosite's scope and audience validation for the client
func (s *Service) ClientAPIScopesAndAudiences(ctx context.Context, clientID string) (scopes []string, audiences []string, err error) {
	var rows []struct {
		Key      string
		Audience string
	}
	err = s.db.WithContext(ctx).
		Table("oidc_clients_allowed_api_permissions AS g").
		Select("api_permissions.key AS key, apis.audience AS audience").
		Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
		Joins("JOIN apis ON apis.id = api_permissions.api_id").
		Where("g.oidc_client_id = ?", clientID).
		Scan(&rows).
		Error
	if err != nil {
		return nil, nil, err
	}

	scopeSeen := make(map[string]struct{}, len(rows))
	audienceSeen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := scopeSeen[row.Key]; !ok {
			scopeSeen[row.Key] = struct{}{}
			scopes = append(scopes, row.Key)
		}
		if _, ok := audienceSeen[row.Audience]; !ok {
			audienceSeen[row.Audience] = struct{}{}
			audiences = append(audiences, row.Audience)
		}
	}

	return scopes, audiences, nil
}

// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience, plus whether such an API exists
func (s *Service) AllowedScopesForAudience(ctx context.Context, clientID, audience string) (scopes []string, apiExists bool, err error) {
	var api API
	err = s.db.WithContext(ctx).
		Select("id").
		Where("audience = ?", audience).
		First(&api).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	err = s.db.WithContext(ctx).
		Table("api_permissions").
		Select("api_permissions.key").
		Joins("JOIN oidc_clients_allowed_api_permissions g ON g.api_permission_id = api_permissions.id AND g.oidc_client_id = ?", clientID).
		Where("api_permissions.api_id = ?", api.ID).
		Pluck("api_permissions.key", &scopes).
		Error
	if err != nil {
		return nil, true, err
	}

	return scopes, true, nil
}

// DescribePermissions returns the permission rows of the API identified by the given audience whose key is in keys
// The consent screen uses these to show friendly names instead of raw scope keys
func (s *Service) DescribePermissions(ctx context.Context, audience string, keys []string) ([]Permission, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	var permissions []Permission
	err := s.db.WithContext(ctx).
		Model(&Permission{}).
		Joins("JOIN apis ON apis.id = api_permissions.api_id").
		Where("apis.audience = ? AND api_permissions.key IN ?", audience, keys).
		Find(&permissions).
		Error
	if err != nil {
		return nil, err
	}

	return permissions, nil
}

// filterAssignablePermissionIDs returns the subset of the given permission IDs that exist
func (s *Service) filterAssignablePermissionIDs(ctx context.Context, tx *gorm.DB, permissionIDs []string) ([]string, error) {
	if len(permissionIDs) == 0 {
		return nil, nil
	}

	var valid []string
	err := tx.WithContext(ctx).
		Model(&Permission{}).
		Where("id IN ?", permissionIDs).
		Pluck("id", &valid).
		Error
	if err != nil {
		return nil, err
	}

	return valid, nil
}

func (s *Service) getInternal(ctx context.Context, id string, tx *gorm.DB) (api API, err error) {
	err = tx.
		WithContext(ctx).
		Preload("Permissions").
		Where("id = ?", id).
		First(&api).
		Error
	return api, err
}

// deletePermissionsInternal removes permissions by ID along with any client allow-list grants that reference them
// The explicit grant delete keeps this correct even when the database does not enforce ON DELETE CASCADE at runtime
func (s *Service) deletePermissionsInternal(ctx context.Context, tx *gorm.DB, permissionIDs []string) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	err := tx.WithContext(ctx).
		Where("api_permission_id IN ?", permissionIDs).
		Delete(&OidcClientAllowedAPIPermission{}).
		Error
	if err != nil {
		return err
	}

	return tx.WithContext(ctx).
		Where("id IN ?", permissionIDs).
		Delete(&Permission{}).
		Error
}

func collectIDs(permissions []Permission) []string {
	ids := make([]string, len(permissions))
	for i, p := range permissions {
		ids[i] = p.ID
	}
	return ids
}
