package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// isPermissionKeyReserved reports whether the key is a scope or claim name owned by Pocket ID's built-in identity layer
// A custom API permission must not reuse one, otherwise its scope string would collide with a standard OIDC scope or claim
func isPermissionKeyReserved(key string) bool {
	switch strings.ToLower(key) {
	case "openid", "profile", "email", "email_verified", "groups", "offline_access":
		return true
	default:
		return false
	}
}

// isValidPermissionKey reports whether the key consists only of RFC 6749 scope-token characters, which are printable ASCII without space, double-quote or backslash
// This keeps a key safe as a space-delimited value in the token scope claim and free of the control character used to qualify consent records
func isValidPermissionKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		// Allow 0x21, 0x23-0x5B and 0x5D-0x7E, which excludes space (0x20), double-quote (0x22) and backslash (0x5C)
		if c == 0x21 || (c >= 0x23 && c <= 0x5B) || (c >= 0x5D && c <= 0x7E) {
			continue
		}
		return false
	}
	return true
}

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
	return issuer != "" && strings.ToLower(strings.TrimRight(audience, "/")) == issuer
}

func (s *Service) List(ctx context.Context, search string, listRequestOptions utils.ListRequestOptions) (apis []API, response utils.PaginationResponse, err error) {
	query := s.db.
		WithContext(ctx).
		Preload("Permissions").
		Model(&API{})

	if listRequestOptions.Sort.Column == "resource" {
		listRequestOptions.Sort.Column = "audience"
	}

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR audience LIKE ?", like, like)
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &apis)
	return apis, response, err
}

// Get loads an API and its permissions
func (s *Service) Get(ctx context.Context, tx *gorm.DB, id string) (api API, err error) {
	query := s.db.WithContext(ctx)
	if tx != nil {
		query = tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"})
	}

	err = query.
		Preload("Permissions").
		Where("id = ?", id).
		First(&api).
		Error
	return api, err
}

func (s *Service) Create(ctx context.Context, input apiCreateDto) (api API, err error) {
	// Reject the issuer as an audience so a custom API cannot impersonate Pocket ID's own identity tokens
	if isIssuerAudience(input.Resource, s.issuer) {
		return API{}, &common.ValidationError{Message: "the resource is reserved by Pocket ID and cannot be used for a custom API"}
	}

	api = API{
		Name:     input.Name,
		Audience: input.Resource,
	}

	err = s.db.WithContext(ctx).Create(&api).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return API{}, &common.AlreadyInUseError{Property: "resource"}
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

	api, err = s.Get(ctx, tx, id)
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

	api, err := s.Get(ctx, tx, id)
	if err != nil {
		return err
	}

	if err = s.deletePermissions(ctx, tx, collectIDs(api.Permissions)); err != nil {
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

	api, err = s.Get(ctx, tx, id)
	if err != nil {
		return API{}, err
	}

	// Reject keys with invalid characters or that collide with Pocket ID's reserved scopes and claims before persisting anything
	for _, permission := range input.Permissions {
		if !isValidPermissionKey(permission.Key) {
			return API{}, &common.ValidationError{Message: fmt.Sprintf("the permission key %q contains invalid characters", permission.Key)}
		}
		if isPermissionKeyReserved(permission.Key) {
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
	if err = s.deletePermissions(ctx, tx, removedIDs); err != nil {
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

	api, err = s.Get(ctx, tx, id)
	if err != nil {
		return API{}, err
	}

	if err = tx.Commit().Error; err != nil {
		return API{}, err
	}

	return api, nil
}

// ClientAPIAccess is the set of API permissions granted to a client, split by the subject the resulting tokens act for
// User-delegated permissions may be requested on behalf of a signed-in user, client permissions may be obtained by the client itself through the client credentials grant
type ClientAPIAccess struct {
	UserDelegatedPermissionIDs []string
	ClientPermissionIDs        []string
}

// GetClientAPIAccess returns the API permissions a client is allowed to request, split by subject type
// Only custom-API permissions are tracked here because the identity scopes are freely requestable by every client
func (s *Service) GetClientAPIAccess(ctx context.Context, clientID string) (access ClientAPIAccess, err error) {
	var rows []OidcClientAllowedAPIPermission
	err = s.db.WithContext(ctx).
		Where("oidc_client_id = ?", clientID).
		Find(&rows).
		Error
	if err != nil {
		return ClientAPIAccess{}, err
	}

	for _, row := range rows {
		switch row.SubjectType {
		case oidc.SubjectTypeClient:
			access.ClientPermissionIDs = append(access.ClientPermissionIDs, row.APIPermissionID)
		default:
			access.UserDelegatedPermissionIDs = append(access.UserDelegatedPermissionIDs, row.APIPermissionID)
		}
	}

	return access, nil
}

// SetClientAPIAccess replaces the client's API-access grants for both subject types with the given permission IDs
// Unknown permission IDs are ignored
// It returns the access that was actually applied
func (s *Service) SetClientAPIAccess(ctx context.Context, clientID string, access ClientAPIAccess) (applied ClientAPIAccess, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Ensure the client exists so callers get a 404 for an unknown client
	var client model.OidcClient
	if err = tx.WithContext(ctx).Select("id").Where("id = ?", clientID).First(&client).Error; err != nil {
		return ClientAPIAccess{}, err
	}

	applied.UserDelegatedPermissionIDs, err = s.filterAssignablePermissionIDs(ctx, tx, access.UserDelegatedPermissionIDs)
	if err != nil {
		return ClientAPIAccess{}, err
	}
	applied.ClientPermissionIDs, err = s.filterAssignablePermissionIDs(ctx, tx, access.ClientPermissionIDs)
	if err != nil {
		return ClientAPIAccess{}, err
	}

	// Replace the grants for this client
	err = tx.WithContext(ctx).
		Where("oidc_client_id = ?", clientID).
		Delete(&OidcClientAllowedAPIPermission{}).
		Error
	if err != nil {
		return ClientAPIAccess{}, err
	}

	rows := make([]OidcClientAllowedAPIPermission, 0, len(applied.UserDelegatedPermissionIDs)+len(applied.ClientPermissionIDs))
	for _, permissionID := range applied.UserDelegatedPermissionIDs {
		rows = append(rows, OidcClientAllowedAPIPermission{OidcClientID: clientID, APIPermissionID: permissionID, SubjectType: oidc.SubjectTypeUser})
	}
	for _, permissionID := range applied.ClientPermissionIDs {
		rows = append(rows, OidcClientAllowedAPIPermission{OidcClientID: clientID, APIPermissionID: permissionID, SubjectType: oidc.SubjectTypeClient})
	}
	if len(rows) > 0 {
		if err = tx.WithContext(ctx).Create(&rows).Error; err != nil {
			return ClientAPIAccess{}, err
		}
	}

	if err = tx.Commit().Error; err != nil {
		return ClientAPIAccess{}, err
	}

	return applied, nil
}

// ClientAPIScopesAndAudiences returns the permission keys a client may request and the distinct audiences of the custom APIs those permissions belong to, across both subject types
// The OIDC module uses this to widen fosite's scope and audience validation for the client; the per-flow subject-type enforcement happens when the resource is resolved
func (s *Service) ClientAPIScopesAndAudiences(ctx context.Context, tx *gorm.DB, clientID string) (scopes []string, audiences []string, err error) {
	if tx == nil {
		tx = s.db
	}

	var rows []struct {
		Key      string
		Audience string
	}
	err = tx.WithContext(ctx).
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
	scopes = make([]string, 0, len(rows))
	audiences = make([]string, 0, len(rows))
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

// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience and subject type, plus whether such an API exists
func (s *Service) AllowedScopesForAudience(ctx context.Context, clientID, audience string, subjectType oidc.SubjectType) (scopes []string, apiExists bool, err error) {
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
		Joins("JOIN oidc_clients_allowed_api_permissions g ON g.api_permission_id = api_permissions.id AND g.oidc_client_id = ? AND g.subject_type = ?", clientID, subjectType).
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

// deletePermissions removes permissions by ID along with any client allow-list grants that reference them
// The explicit grant delete keeps this correct even when the database does not enforce ON DELETE CASCADE at runtime
func (s *Service) deletePermissions(ctx context.Context, tx *gorm.DB, permissionIDs []string) error {
	if tx == nil {
		tx = s.db
	}
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
