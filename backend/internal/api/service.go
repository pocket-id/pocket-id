package api

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
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
	return fosite.IsValidScopeToken(key)
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return API{}, apperror.NotFound("API")
	}
	return api, err
}

func (s *Service) Create(ctx context.Context, input apiCreateDto) (api API, err error) {
	// Reject the issuer as an audience so a custom API cannot impersonate Pocket ID's own identity tokens
	if isIssuerAudience(input.Resource, s.issuer) {
		return API{}, apperror.InvalidField("resource", "reserved", "is reserved by Pocket ID and cannot be used for a custom API")
	}

	api = API{
		Name:     input.Name,
		Audience: input.Resource,
	}

	err = s.db.WithContext(ctx).Create(&api).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return API{}, apperror.AlreadyInUse("resource")
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

	// Reject keys with invalid characters, that collide with Pocket ID's reserved scopes and claims, or that repeat within the request before persisting anything
	// A duplicate key would otherwise be silently coalesced last-wins into the map below, dropping a row behind a 200
	seen := make(map[string]struct{}, len(input.Permissions))
	for index, permission := range input.Permissions {
		field := fmt.Sprintf("permissions[%d].key", index)
		if !isValidPermissionKey(permission.Key) {
			return API{}, apperror.InvalidField(field, "invalid_format", "contains characters that are not valid in an OAuth scope")
		}
		if isPermissionKeyReserved(permission.Key) {
			return API{}, apperror.InvalidField(field, "reserved", "is reserved by Pocket ID")
		}
		_, ok := seen[permission.Key]
		if ok {
			return API{}, apperror.InvalidField(field, "duplicate", "is listed more than once")
		}
		seen[permission.Key] = struct{}{}
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

// SetCIMDAccess toggles whether an API is open to CIMD clients and stores which permissions they may request
// The selection is kept while access is off, so re-enabling restores the previous choice
func (s *Service) SetCIMDAccess(ctx context.Context, id string, input apiCimdAccessUpdateDto) (api API, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err = s.Get(ctx, tx, id)
	if err != nil {
		return API{}, err
	}

	// Keep only permission IDs that belong to this API; ignore anything else
	selected := make([]string, 0, len(input.PermissionIDs))
	for _, permission := range api.Permissions {
		if slices.Contains(input.PermissionIDs, permission.ID) {
			selected = append(selected, permission.ID)
		}
	}

	// Clear the flag on all permissions first, then set it on the selected ones
	err = tx.WithContext(ctx).
		Model(&Permission{}).
		Where("api_id = ?", api.ID).
		Update("allowed_for_cimd_clients", false).
		Error
	if err != nil {
		return API{}, err
	}

	if len(selected) > 0 {
		err = tx.WithContext(ctx).
			Model(&Permission{}).
			Where("api_id = ? AND id IN ?", api.ID, selected).
			Update("allowed_for_cimd_clients", true).
			Error
		if err != nil {
			return API{}, err
		}
	}

	err = tx.WithContext(ctx).
		Model(&API{}).
		Where("id = ?", api.ID).
		Updates(map[string]any{"allow_cimd_clients": input.Enabled, "updated_at": new(datatype.DateTime(time.Now()))}).
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

// ClientAPIAccess is what a client is allowed to do with the custom APIs, split by the subject the resulting tokens act for
type ClientAPIAccess struct {
	UserDelegatedAPIIDs        []string
	ClientAPIIDs               []string
	UserDelegatedPermissionIDs []string
	ClientPermissionIDs        []string
}

// ClientAPIGrant is one API a client may reach, together with the grants that apply to it
type ClientAPIGrant struct {
	API API
	APIClientGrant
	CIMDGrantedAccess        bool
	CIMDGrantedPermissionIDs []string
}

// ListClientAPIs returns every API the client may request tokens for, ordered by name, including APIs reached only through a CIMD opt-in
func (s *Service) ListClientAPIs(ctx context.Context, clientID string) ([]ClientAPIGrant, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	if err := ensureOIDCClientExists(ctx, tx, clientID); err != nil {
		return nil, err
	}

	grants := make(map[string]*ClientAPIGrant)
	grantFor := func(apiID string) *ClientAPIGrant {
		entry, ok := grants[apiID]
		if !ok {
			entry = &ClientAPIGrant{}
			grants[apiID] = entry
		}
		return entry
	}

	// Get the APIs the client may reach
	var apiRows []OidcClientAllowedAPI
	err := tx.WithContext(ctx).
		Where("oidc_client_id = ?", clientID).
		Find(&apiRows).
		Error
	if err != nil {
		return nil, err
	}

	// Collect the access flags for each API
	for _, row := range apiRows {
		switch row.SubjectType {
		case oidc.SubjectTypeClient:
			grantFor(row.APIID).ClientAccess = true
		case oidc.SubjectTypeUser:
			grantFor(row.APIID).UserDelegatedAccess = true
		}
	}

	var permissionRows []struct {
		APIID           string
		APIPermissionID string
		SubjectType     oidc.SubjectType
	}

	// Get the permissions the client may request
	err = tx.WithContext(ctx).
		Table("oidc_clients_allowed_api_permissions AS g").
		Select("api_permissions.api_id AS api_id, g.api_permission_id AS api_permission_id, g.subject_type AS subject_type").
		Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
		Where("g.oidc_client_id = ?", clientID).
		Scan(&permissionRows).
		Error
	if err != nil {
		return nil, err
	}

	// Collect the permission IDs for each API and subject type
	for _, row := range permissionRows {
		entry := grantFor(row.APIID)
		switch row.SubjectType {
		case oidc.SubjectTypeClient:
			entry.ClientPermissionIDs = append(entry.ClientPermissionIDs, row.APIPermissionID)
		case oidc.SubjectTypeUser:
			entry.UserDelegatedPermissionIDs = append(entry.UserDelegatedPermissionIDs, row.APIPermissionID)
		}
	}

	// Now the same again for the CIMD
	cimdGranted, err := s.cimdGrantedAccess(ctx, tx, clientID, "")
	if err != nil {
		return nil, err
	}
	for _, api := range cimdGranted.APIs {
		grantFor(api.ID).CIMDGrantedAccess = true
	}
	for _, permission := range cimdGranted.Permissions {
		entry, ok := grants[permission.APIID]
		if !ok {
			continue
		}
		entry.CIMDGrantedPermissionIDs = append(entry.CIMDGrantedPermissionIDs, permission.ID)
	}

	if len(grants) == 0 {
		return []ClientAPIGrant{}, nil
	}

	// Load the API rows for the grants we collected, so the caller can see the API's name and audience
	var apis []API
	err = tx.WithContext(ctx).
		Preload("Permissions").
		Where("id IN ?", slices.Collect(maps.Keys(grants))).
		Order("name").
		Find(&apis).
		Error
	if err != nil {
		return nil, err
	}

	result := make([]ClientAPIGrant, 0, len(apis))
	for _, api := range apis {
		entry := *grants[api.ID]
		entry.API = api
		entry.UserDelegatedPermissionIDs = orEmptyIDs(entry.UserDelegatedPermissionIDs)
		entry.ClientPermissionIDs = orEmptyIDs(entry.ClientPermissionIDs)
		entry.CIMDGrantedPermissionIDs = orEmptyIDs(entry.CIMDGrantedPermissionIDs)
		result = append(result, entry)
	}

	return result, nil
}

// insertClientGrants writes the API and permission grants of one client for both subject types
func insertClientGrants(ctx context.Context, tx *gorm.DB, clientID string, access ClientAPIAccess) error {
	apiRows := make([]OidcClientAllowedAPI, 0, len(access.UserDelegatedAPIIDs)+len(access.ClientAPIIDs))
	for _, apiID := range access.UserDelegatedAPIIDs {
		apiRows = append(apiRows, OidcClientAllowedAPI{OidcClientID: clientID, APIID: apiID, SubjectType: oidc.SubjectTypeUser})
	}
	for _, apiID := range access.ClientAPIIDs {
		apiRows = append(apiRows, OidcClientAllowedAPI{OidcClientID: clientID, APIID: apiID, SubjectType: oidc.SubjectTypeClient})
	}
	if len(apiRows) > 0 {
		if err := tx.WithContext(ctx).Create(&apiRows).Error; err != nil {
			return err
		}
	}

	permissionRows := make([]OidcClientAllowedAPIPermission, 0, len(access.UserDelegatedPermissionIDs)+len(access.ClientPermissionIDs))
	for _, permissionID := range access.UserDelegatedPermissionIDs {
		permissionRows = append(permissionRows, OidcClientAllowedAPIPermission{OidcClientID: clientID, APIPermissionID: permissionID, SubjectType: oidc.SubjectTypeUser})
	}
	for _, permissionID := range access.ClientPermissionIDs {
		permissionRows = append(permissionRows, OidcClientAllowedAPIPermission{OidcClientID: clientID, APIPermissionID: permissionID, SubjectType: oidc.SubjectTypeClient})
	}
	if len(permissionRows) > 0 {
		if err := tx.WithContext(ctx).Create(&permissionRows).Error; err != nil {
			return err
		}
	}

	return nil
}

// APIClientGrant is what one client may do with a single API
// Access and permissions are separate: access without any permission yields a token that carries no scope
type APIClientGrant struct {
	UserDelegatedAccess        bool
	ClientAccess               bool
	UserDelegatedPermissionIDs []string
	ClientPermissionIDs        []string
}

// APIClientAccess is a single client's grants on one API, as shown on the API's detail page
// The CIMD fields are read-only here: that access is managed on the API, not per client
type APIClientAccess struct {
	Client model.OidcClient
	APIClientGrant
	CIMDGrantedAccess        bool
	CIMDGrantedPermissionIDs []string
}

// ListAPIClients returns, one page at a time, every client that may reach the API, ordered by name, including clients covered by the CIMD opt-in (which are marked)
// The clients are paginated so an opted-in API does not load and return every metadata document client in the instance at once
func (s *Service) ListAPIClients(ctx context.Context, apiID, search string, listRequestOptions utils.ListRequestOptions) (result []APIClientAccess, response utils.PaginationResponse, err error) {
	api, err := s.Get(ctx, nil, apiID)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	access := make(map[string]*APIClientGrant)
	grantFor := func(clientID string) *APIClientGrant {
		entry, ok := access[clientID]
		if !ok {
			entry = &APIClientGrant{}
			access[clientID] = entry
		}
		return entry
	}

	// Access rows drive the list: a client with access but no permission still belongs in it
	var apiRows []OidcClientAllowedAPI
	err = s.db.WithContext(ctx).
		Where("api_id = ?", api.ID).
		Find(&apiRows).
		Error
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}
	for _, row := range apiRows {
		switch row.SubjectType {
		case oidc.SubjectTypeClient:
			grantFor(row.OidcClientID).ClientAccess = true
		case oidc.SubjectTypeUser:
			grantFor(row.OidcClientID).UserDelegatedAccess = true
		}
	}

	var permissionRows []struct {
		OidcClientID    string
		APIPermissionID string
		SubjectType     oidc.SubjectType
	}
	err = s.db.WithContext(ctx).
		Table("oidc_clients_allowed_api_permissions AS g").
		Select("g.oidc_client_id AS oidc_client_id, g.api_permission_id AS api_permission_id, g.subject_type AS subject_type").
		Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
		Where("api_permissions.api_id = ?", api.ID).
		Scan(&permissionRows).
		Error
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}
	for _, row := range permissionRows {
		entry := grantFor(row.OidcClientID)
		switch row.SubjectType {
		case oidc.SubjectTypeClient:
			entry.ClientPermissionIDs = append(entry.ClientPermissionIDs, row.APIPermissionID)
		case oidc.SubjectTypeUser:
			entry.UserDelegatedPermissionIDs = append(entry.UserDelegatedPermissionIDs, row.APIPermissionID)
		}
	}

	// Permissions a CIMD client receives come from the API's opt-in selection, the same for every such client
	var cimdPermissionIDs []string
	if api.AllowCIMDClients {
		err = s.db.WithContext(ctx).
			Model(&Permission{}).
			Where("api_id = ? AND allowed_for_cimd_clients = ?", api.ID, true).
			Pluck("id", &cimdPermissionIDs).
			Error
		if err != nil {
			return nil, utils.PaginationResponse{}, err
		}
	}

	// Load the matching clients one page at a time: those with an explicit grant, plus every CIMD client when the API opted in
	explicitIDs := slices.Collect(maps.Keys(access))
	query := s.db.WithContext(ctx).Model(&model.OidcClient{})
	switch {
	case api.AllowCIMDClients && len(explicitIDs) > 0:
		query = query.Where("id IN ? OR client_type = ?", explicitIDs, model.OidcClientTypeCIMD)
	case api.AllowCIMDClients:
		query = query.Where("client_type = ?", model.OidcClientTypeCIMD)
	case len(explicitIDs) > 0:
		query = query.Where("id IN ?", explicitIDs)
	default:
		// No client can reach the API, so match nothing while still returning a well-formed page
		query = query.Where("1 = 0")
	}
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	// Keep the list stably ordered by name when the caller does not ask for a specific sort
	if listRequestOptions.Sort.Column == "" {
		listRequestOptions.Sort.Column = "name"
		listRequestOptions.Sort.Direction = "asc"
	}

	var clients []model.OidcClient
	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	result = make([]APIClientAccess, 0, len(clients))
	for _, client := range clients {
		grant := APIClientGrant{}
		if existing, ok := access[client.ID]; ok {
			grant = *existing
		}
		grant.UserDelegatedPermissionIDs = orEmptyIDs(grant.UserDelegatedPermissionIDs)
		grant.ClientPermissionIDs = orEmptyIDs(grant.ClientPermissionIDs)

		entry := APIClientAccess{Client: client, APIClientGrant: grant, CIMDGrantedPermissionIDs: []string{}}
		if api.AllowCIMDClients && client.ClientType == model.OidcClientTypeCIMD {
			entry.CIMDGrantedAccess = true
			entry.CIMDGrantedPermissionIDs = cimdPermissionIDs
		}
		result = append(result, entry)
	}

	return result, response, nil
}

// ListAssignableClients returns the clients that hold no grant on the API yet, so only what can still be added is offered
func (s *Service) ListAssignableClients(ctx context.Context, apiID, search string, listRequestOptions utils.ListRequestOptions) (clients []model.OidcClient, response utils.PaginationResponse, err error) {
	api, err := s.Get(ctx, nil, apiID)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	granted := s.db.
		Model(&OidcClientAllowedAPI{}).
		Select("oidc_client_id").
		Where("api_id = ?", api.ID)

	query := s.db.
		WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id NOT IN (?)", granted)

	// CIMD clients already reach an opted-in API and are listed for it, so don't offer them again
	if api.AllowCIMDClients {
		query = query.Where("client_type <> ?", model.OidcClientTypeCIMD)
	}

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	return clients, response, err
}

// ListAssignableAPIs returns the APIs the client cannot already reach, which are the ones missing from its access list
func (s *Service) ListAssignableAPIs(ctx context.Context, clientID, search string, listRequestOptions utils.ListRequestOptions) (apis []API, response utils.PaginationResponse, err error) {
	if err = ensureOIDCClientExists(ctx, s.db, clientID); err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	granted := s.db.
		Model(&OidcClientAllowedAPI{}).
		Select("api_id").
		Where("oidc_client_id = ?", clientID)

	query := s.db.
		WithContext(ctx).
		Preload("Permissions").
		Model(&API{}).
		Where("id NOT IN (?)", granted)

	// APIs reached through the CIMD opt-in are already listed for the client, so don't offer them again
	cimdGranted, err := s.cimdGrantedAccess(ctx, s.db, clientID, "")
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}
	if cimdGranted.grantsAccess() {
		query = query.Where("id NOT IN ?", cimdGranted.apiIDs())
	}

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR audience LIKE ?", like, like)
	}

	if listRequestOptions.Sort.Column == "resource" {
		listRequestOptions.Sort.Column = "audience"
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &apis)
	return apis, response, err
}

// SetAPIClientAccess replaces a single client's grants on one API, leaving whatever it was granted on other APIs untouched
// Permission IDs that do not belong to this API are ignored, and a permission implies access for its subject type
func (s *Service) SetAPIClientAccess(ctx context.Context, apiID, clientID string, grant APIClientGrant) (applied APIClientGrant, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err := s.Get(ctx, tx, apiID)
	if err != nil {
		return APIClientGrant{}, err
	}
	client, err := loadOIDCClient(ctx, tx, clientID)
	if err != nil {
		return APIClientGrant{}, err
	}

	permissionIDs := collectIDs(api.Permissions)
	applied.UserDelegatedPermissionIDs = intersectIDs(permissionIDs, grant.UserDelegatedPermissionIDs)
	applied.ClientPermissionIDs = intersectIDs(permissionIDs, grant.ClientPermissionIDs)
	// A public client can't use the client credentials grant, so drop any client-subject grant that could never produce a token
	if client.IsPublic {
		grant.ClientAccess = false
		applied.ClientPermissionIDs = nil
	}
	applied.UserDelegatedAccess = grant.UserDelegatedAccess || len(applied.UserDelegatedPermissionIDs) > 0
	applied.ClientAccess = grant.ClientAccess || len(applied.ClientPermissionIDs) > 0

	if err = deleteAPIClientGrants(ctx, tx, api.ID, permissionIDs, clientID); err != nil {
		return APIClientGrant{}, err
	}

	access := ClientAPIAccess{
		UserDelegatedPermissionIDs: applied.UserDelegatedPermissionIDs,
		ClientPermissionIDs:        applied.ClientPermissionIDs,
	}
	if applied.UserDelegatedAccess {
		access.UserDelegatedAPIIDs = []string{api.ID}
	}
	if applied.ClientAccess {
		access.ClientAPIIDs = []string{api.ID}
	}
	if err = insertClientGrants(ctx, tx, clientID, access); err != nil {
		return APIClientGrant{}, err
	}

	if err = tx.Commit().Error; err != nil {
		return APIClientGrant{}, err
	}

	return applied, nil
}

// RemoveAPIClientAccess revokes every grant a client holds on one API
func (s *Service) RemoveAPIClientAccess(ctx context.Context, apiID, clientID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	api, err := s.Get(ctx, tx, apiID)
	if err != nil {
		return err
	}
	// Reject an unknown client id so it can't delete nothing and still report success
	if err = ensureOIDCClientExists(ctx, tx, clientID); err != nil {
		return err
	}

	if err = deleteAPIClientGrants(ctx, tx, api.ID, collectIDs(api.Permissions), clientID); err != nil {
		return err
	}

	return tx.Commit().Error
}

// deleteAPIClientGrants removes a client's access and permission grants for one API, leaving its grants on other APIs untouched
func deleteAPIClientGrants(ctx context.Context, tx *gorm.DB, apiID string, permissionIDs []string, clientID string) error {
	err := tx.WithContext(ctx).
		Where("oidc_client_id = ? AND api_id = ?", clientID, apiID).
		Delete(&OidcClientAllowedAPI{}).
		Error
	if err != nil || len(permissionIDs) == 0 {
		return err
	}

	return tx.WithContext(ctx).
		Where("oidc_client_id = ? AND api_permission_id IN ?", clientID, permissionIDs).
		Delete(&OidcClientAllowedAPIPermission{}).
		Error
}

// orEmptyIDs keeps an ID list non-nil so it serializes as [] rather than null, which the admin UI indexes into directly
func orEmptyIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// intersectIDs returns the requested IDs that are part of available, preserving the order of available
func intersectIDs(available, requested []string) []string {
	if len(requested) == 0 {
		return nil
	}

	result := make([]string, 0, len(requested))
	for _, id := range available {
		if slices.Contains(requested, id) {
			result = append(result, id)
		}
	}
	return result
}

func ensureOIDCClientExists(ctx context.Context, db *gorm.DB, clientID string) error {
	_, err := loadOIDCClient(ctx, db, clientID)
	return err
}

// loadOIDCClient reads the few client fields the grant paths need to decide what a client may be granted
func loadOIDCClient(ctx context.Context, db *gorm.DB, clientID string) (model.OidcClient, error) {
	var client model.OidcClient
	err := db.WithContext(ctx).
		Select("id", "is_public", "client_type").
		Where("id = ?", clientID).
		First(&client).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.OidcClient{}, apperror.NotFound("OIDC client")
	}

	return client, err
}

// ClientAPIScopesAndAudiences returns the permission keys a client may request and the distinct audiences of every custom API it may reach, across both subject types
// The OIDC module uses this to widen fosite's scope and audience validation; per-flow subject-type enforcement happens when the resource is resolved
// An accessible API contributes its audience even without any permission, so a scopeless resource request isn't rejected before it is resolved
func (s *Service) ClientAPIScopesAndAudiences(ctx context.Context, tx *gorm.DB, clientID string, isCIMDClient bool) (scopes []string, audiences []string, err error) {
	if tx == nil {
		tx = s.db
	}

	var audienceRows []string
	err = tx.WithContext(ctx).
		Table("oidc_clients_allowed_apis AS g").
		Joins("JOIN apis ON apis.id = g.api_id").
		Where("g.oidc_client_id = ?", clientID).
		Pluck("apis.audience", &audienceRows).
		Error
	if err != nil {
		return nil, nil, err
	}

	var scopeRows []string
	err = tx.WithContext(ctx).
		Table("oidc_clients_allowed_api_permissions AS g").
		Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
		Where("g.oidc_client_id = ?", clientID).
		Pluck("api_permissions.key", &scopeRows).
		Error
	if err != nil {
		return nil, nil, err
	}

	// CIMD clients additionally reach every API that opted in to them
	if isCIMDClient {
		cimdGranted, err := s.cimdGrantedAccess(ctx, tx, clientID, "")
		if err != nil {
			return nil, nil, err
		}
		audienceRows = append(audienceRows, cimdGranted.audiences()...)
		scopeRows = append(scopeRows, cimdGranted.scopes()...)
	}

	return distinct(scopeRows), distinct(audienceRows), nil
}

// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience and subject type, whether such an API exists, and whether the client may reach it at all
func (s *Service) AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType oidc.SubjectType) (scopes []string, apiExists bool, hasAccess bool, err error) {
	if tx == nil {
		tx = s.db
	}

	var api API
	err = tx.WithContext(ctx).
		Select("id", "allow_cimd_clients").
		Where("audience = ?", audience).
		First(&api).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, false, nil
	}
	if err != nil {
		return nil, false, false, err
	}

	var grantCount int64
	err = tx.WithContext(ctx).
		Model(&OidcClientAllowedAPI{}).
		Where("oidc_client_id = ? AND api_id = ? AND subject_type = ?", clientID, api.ID, subjectType).
		Count(&grantCount).
		Error
	if err != nil {
		return nil, true, false, err
	}
	hasAccess = grantCount > 0

	err = tx.WithContext(ctx).
		Table("api_permissions").
		Select("api_permissions.key").
		Joins("JOIN oidc_clients_allowed_api_permissions g ON g.api_permission_id = api_permissions.id AND g.oidc_client_id = ? AND g.subject_type = ?", clientID, subjectType).
		Where("api_permissions.api_id = ?", api.ID).
		Pluck("api_permissions.key", &scopes).
		Error
	if err != nil {
		return nil, true, hasAccess, err
	}

	// The opt-in only covers user-delegated access: CIMD clients are public and can't use the client credentials grant
	if api.AllowCIMDClients && subjectType == oidc.SubjectTypeUser {
		cimdGranted, err := s.cimdGrantedAccess(ctx, tx, clientID, api.ID)
		if err != nil {
			return nil, true, hasAccess, err
		}
		if cimdGranted.grantsAccess() {
			hasAccess = true
		}
		for _, scope := range cimdGranted.scopes() {
			if !slices.Contains(scopes, scope) {
				scopes = append(scopes, scope)
			}
		}
	}

	return scopes, true, hasAccess, nil
}

func distinct(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// cimdGrantedAccessRows is what the API-wide CIMD opt-in gives one client
// APIs stand on their own: one can opt in without selecting any permission, granting access with no scope
type cimdGrantedAccessRows struct {
	APIs []struct {
		ID       string
		Audience string
	}
	Permissions []struct {
		ID    string
		Key   string
		APIID string
	}
}

func (r cimdGrantedAccessRows) grantsAccess() bool { return len(r.APIs) > 0 }

func (r cimdGrantedAccessRows) apiIDs() []string {
	ids := make([]string, len(r.APIs))
	for i, api := range r.APIs {
		ids[i] = api.ID
	}
	return ids
}

func (r cimdGrantedAccessRows) audiences() []string {
	audiences := make([]string, len(r.APIs))
	for i, api := range r.APIs {
		audiences[i] = api.Audience
	}
	return audiences
}

func (r cimdGrantedAccessRows) scopes() []string {
	keys := make([]string, len(r.Permissions))
	for i, permission := range r.Permissions {
		keys[i] = permission.Key
	}
	return keys
}

// cimdGrantedAccess returns the APIs a client may reach and the permissions it may request through the CIMD opt-in, optionally narrowed to a single API
// The CIMD check is a join, so a client registered the regular way simply matches no rows
func (s *Service) cimdGrantedAccess(ctx context.Context, tx *gorm.DB, clientID, apiID string) (rows cimdGrantedAccessRows, err error) {
	if tx == nil {
		tx = s.db
	}

	apiQuery := tx.WithContext(ctx).
		Table("apis").
		Select("apis.id AS id, apis.audience AS audience").
		Joins("JOIN oidc_clients ON oidc_clients.id = ? AND oidc_clients.client_type = ?", clientID, model.OidcClientTypeCIMD).
		Where("apis.allow_cimd_clients = ?", true)
	if apiID != "" {
		apiQuery = apiQuery.Where("apis.id = ?", apiID)
	}
	if err = apiQuery.Scan(&rows.APIs).Error; err != nil {
		return cimdGrantedAccessRows{}, err
	}
	if len(rows.APIs) == 0 {
		return rows, nil
	}

	err = tx.WithContext(ctx).
		Table("api_permissions").
		Select("api_permissions.id AS id, api_permissions.key AS key, api_permissions.api_id AS api_id").
		Where("api_permissions.api_id IN ? AND api_permissions.allowed_for_cimd_clients = ?", rows.apiIDs(), true).
		Scan(&rows.Permissions).
		Error
	if err != nil {
		return cimdGrantedAccessRows{}, err
	}

	return rows, nil
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

// deletePermissions removes permissions by ID; deleting them cascades to any client permission grants that reference them (oidc_clients_allowed_api_permissions.api_permission_id ON DELETE CASCADE)
// A client's access to an API is dropped together with the last permission it held there, so removing a permission never leaves a client with lingering scopeless access it never asked for; access granted without any permission is untouched because such a client holds none of the deleted permissions
func (s *Service) deletePermissions(ctx context.Context, tx *gorm.DB, permissionIDs []string) error {
	if tx == nil {
		tx = s.db
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	// Record which clients held these permissions, and on which API and subject type, before the grants are gone
	var affected []struct {
		OidcClientID string
		APIID        string `gorm:"column:api_id"`
		SubjectType  oidc.SubjectType
	}
	err := tx.WithContext(ctx).
		Table("oidc_clients_allowed_api_permissions AS g").
		Select("DISTINCT g.oidc_client_id AS oidc_client_id, api_permissions.api_id AS api_id, g.subject_type AS subject_type").
		Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
		Where("g.api_permission_id IN ?", permissionIDs).
		Scan(&affected).
		Error
	if err != nil {
		return err
	}

	if err = tx.WithContext(ctx).
		Where("id IN ?", permissionIDs).
		Delete(&Permission{}).
		Error; err != nil {
		return err
	}

	// Drop the access a deleted permission implied, unless the client still holds another permission of the same API for that subject type
	for _, row := range affected {
		var remaining int64
		err = tx.WithContext(ctx).
			Table("oidc_clients_allowed_api_permissions AS g").
			Joins("JOIN api_permissions ON api_permissions.id = g.api_permission_id").
			Where("g.oidc_client_id = ? AND api_permissions.api_id = ? AND g.subject_type = ?", row.OidcClientID, row.APIID, row.SubjectType).
			Count(&remaining).
			Error
		if err != nil {
			return err
		}
		if remaining > 0 {
			continue
		}

		err = tx.WithContext(ctx).
			Where("oidc_client_id = ? AND api_id = ? AND subject_type = ?", row.OidcClientID, row.APIID, row.SubjectType).
			Delete(&OidcClientAllowedAPI{}).
			Error
		if err != nil {
			return err
		}
	}

	return nil
}

func collectIDs(permissions []Permission) []string {
	ids := make([]string, len(permissions))
	for i, p := range permissions {
		ids[i] = p.ID
	}
	return ids
}
