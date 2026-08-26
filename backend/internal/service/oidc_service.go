package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	imageutil "github.com/pocket-id/pocket-id/backend/internal/utils/image"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeClientCredentials = "client_credentials"

	AccessTokenDuration  = time.Duration(model.DefaultAccessTokenDurationMinutes) * time.Minute
	RefreshTokenDuration = time.Duration(model.DefaultRefreshTokenDurationMinutes) * time.Minute
)

type OidcService struct {
	db                *gorm.DB
	jwtService        *JwtService
	previewBuilder    oidcClientPreviewBuilder
	metadataRefresher metadataRefresher
	scimSyncScheduler ScimSyncScheduler

	httpClient  *http.Client
	fileStorage storage.FileStorage
}

type oidcClientPreviewBuilder interface {
	BuildClientPreview(ctx context.Context, client model.OidcClient, userID string, scopes []string, authenticationMethod string) (*oidc.ClientPreview, error)
}

type metadataRefresher interface {
	RefreshClientMetadata(ctx context.Context, clientID string) (model.OidcClient, error)
}

func NewOidcService(
	db *gorm.DB,
	jwtService *JwtService,
	previewBuilder oidcClientPreviewBuilder,
	metadataRefresher metadataRefresher,
	scimSyncScheduler ScimSyncScheduler,
	httpClient *http.Client,
	fileStorage storage.FileStorage,
) (s *OidcService, err error) {
	s = &OidcService{
		db:                db,
		jwtService:        jwtService,
		previewBuilder:    previewBuilder,
		metadataRefresher: metadataRefresher,
		scimSyncScheduler: scimSyncScheduler,
		httpClient:        httpClient,
		fileStorage:       fileStorage,
	}

	return s, nil
}

func (s *OidcService) GetClient(ctx context.Context, clientID string) (model.OidcClient, error) {
	return s.getClientInternal(ctx, clientID, s.db, false)
}

// RefreshClientMetadata forces a re-fetch of the OAuth Client ID Metadata Document
// for a CIMD client, bypassing the cache TTL, and returns the refreshed client.
func (s *OidcService) RefreshClientMetadata(ctx context.Context, clientID string) (model.OidcClient, error) {
	if s.metadataRefresher == nil {
		return model.OidcClient{}, apperror.ValidationMessage("Client ID metadata documents are not enabled")
	}
	client, err := s.metadataRefresher.RefreshClientMetadata(ctx, clientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.OidcClient{}, err
		}
		return model.OidcClient{}, apperror.ValidationMessage(err.Error())
	}
	return client, nil
}

func (s *OidcService) getClientInternal(ctx context.Context, clientID string, tx *gorm.DB, forUpdate bool) (model.OidcClient, error) {
	var client model.OidcClient
	q := tx.
		WithContext(ctx).
		Preload("CreatedBy").
		Preload("AllowedUserGroups")
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	q = q.First(&client, "id = ?", clientID)
	if errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return model.OidcClient{}, apperror.NotFound("OIDC client")
	}
	if q.Error != nil {
		return model.OidcClient{}, q.Error
	}
	return client, nil
}

func (s *OidcService) ListClients(ctx context.Context, name string, listRequestOptions utils.ListRequestOptions) ([]model.OidcClient, utils.PaginationResponse, error) {
	var clients []model.OidcClient

	query := s.db.
		WithContext(ctx).
		Preload("CreatedBy").
		Preload("AllowedUserGroups").
		Model(&model.OidcClient{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// Sort the allowed user groups relation by its row count because it is not an OIDC client column
	if listRequestOptions.Sort.Column == "allowedUserGroups" && utils.IsValidSortDirection(listRequestOptions.Sort.Direction) {
		query = query.Select("oidc_clients.*, COUNT(oidc_clients_allowed_user_groups.oidc_client_id)").
			Joins("LEFT JOIN oidc_clients_allowed_user_groups ON oidc_clients.id = oidc_clients_allowed_user_groups.oidc_client_id").
			Group("oidc_clients.id").
			Order("COUNT(oidc_clients_allowed_user_groups.oidc_client_id) " + listRequestOptions.Sort.Direction)

		response, err := utils.Paginate(listRequestOptions.Pagination.Page, listRequestOptions.Pagination.Limit, query, &clients)
		return clients, response, err
	}

	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	return clients, response, err
}

func (s *OidcService) CreateClient(ctx context.Context, input dto.OidcClientCreateDto, userID string) (model.OidcClient, error) {
	client := model.OidcClient{
		Base: model.Base{
			ID: input.ID,
		},
		CreatedByID: new(userID),
	}
	err := updateOIDCClientModelFromDto(&client, &input.OidcClientUpdateDto)
	if err != nil {
		return model.OidcClient{}, err
	}

	err = s.db.
		WithContext(ctx).
		Create(&client).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.OidcClient{}, apperror.ClientIDAlreadyExists()
		}
		return model.OidcClient{}, err
	}

	// All storage operations must be executed outside of a transaction
	if input.LogoURL != nil {
		err = s.downloadAndSaveLogoFromURL(ctx, client.ID, *input.LogoURL, true)
		if err != nil {
			return model.OidcClient{}, fmt.Errorf("failed to download logo: %w", err)
		}
	}

	if input.DarkLogoURL != nil {
		err = s.downloadAndSaveLogoFromURL(ctx, client.ID, *input.DarkLogoURL, false)
		if err != nil {
			return model.OidcClient{}, fmt.Errorf("failed to download dark logo: %w", err)
		}
	}

	return client, nil
}

func (s *OidcService) UpdateClient(ctx context.Context, clientID string, input dto.OidcClientUpdateDto) (model.OidcClient, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err := s.getClientInternal(ctx, clientID, tx, true)
	if err != nil {
		return model.OidcClient{}, err
	}

	err = updateOIDCClientModelFromDto(&client, &input)
	if err != nil {
		return model.OidcClient{}, err
	}

	if !input.IsGroupRestricted {
		// Clear allowed user groups if the restriction is removed
		err = tx.Model(&client).Association("AllowedUserGroups").Clear()
		if err != nil {
			return model.OidcClient{}, err
		}
	}

	// Metadata refresh owns all other CIMD columns, so an admin update must never write back a stale metadata snapshot
	if client.IsMetadataDocument() {
		err = tx.WithContext(ctx).
			Model(&client).
			Select(
				"Description",
				"RequiresReauthentication",
				"RequiresPushedAuthorizationRequests",
				"SkipConsent",
				"LaunchURL",
				"IsGroupRestricted",
				"AccessTokenDurationMinutes",
				"RefreshTokenDurationMinutes",
			).
			Updates(&client).Error
	} else {
		err = tx.WithContext(ctx).Save(&client).Error
	}
	if err != nil {
		return model.OidcClient{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.OidcClient{}, err
	}

	// All storage operations must be executed outside of a transaction
	if input.LogoURL != nil {
		err = s.downloadAndSaveLogoFromURL(ctx, client.ID, *input.LogoURL, true)
		if err != nil {
			return model.OidcClient{}, fmt.Errorf("failed to download logo: %w", err)
		}
	}

	if input.DarkLogoURL != nil {
		err = s.downloadAndSaveLogoFromURL(ctx, client.ID, *input.DarkLogoURL, false)
		if err != nil {
			return model.OidcClient{}, fmt.Errorf("failed to download dark logo: %w", err)
		}
	}

	return client, nil
}

func updateOIDCClientModelFromDto(client *model.OidcClient, input *dto.OidcClientUpdateDto) error {
	// Update fields that remain locally managed for every client type
	client.Description = input.Description
	client.RequiresReauthentication = input.RequiresReauthentication
	client.RequiresPushedAuthorizationRequests = input.RequiresPushedAuthorizationRequests
	client.SkipConsent = input.SkipConsent
	client.LaunchURL = input.LaunchURL
	client.IsGroupRestricted = input.IsGroupRestricted

	// Token lifetimes are optional, so a zero value falls back to the default
	client.AccessTokenDurationMinutes = cmp.Or(input.AccessTokenDurationMinutes, model.DefaultAccessTokenDurationMinutes)
	client.RefreshTokenDurationMinutes = cmp.Or(input.RefreshTokenDurationMinutes, model.DefaultRefreshTokenDurationMinutes)

	// Preserve fields that are sourced from the client metadata document
	if client.IsMetadataDocument() {
		return nil
	}

	// Update registration fields for manually configured clients
	client.Name = input.Name
	client.CallbackURLs = input.CallbackURLs
	client.LogoutCallbackURLs = input.LogoutCallbackURLs
	client.IsPublic = input.IsPublic
	// PKCE is required for public clients
	client.PkceEnabled = input.IsPublic || input.PkceEnabled
	// Reset any PKCE support prompt if previously flagged
	if !input.PkceEnabled {
		client.PkceSupported = false
	}

	// Replace the federated credentials with the submitted configuration
	federatedIdentities := make([]model.OidcClientFederatedIdentity, len(input.Credentials.FederatedIdentities))
	for i, fi := range input.Credentials.FederatedIdentities {
		// Validate the public keys before storing them
		publicKeys, err := jwkutils.NormalizePublicKeys(fi.PublicKeys)
		if err != nil {
			return apperror.ValidationMessage(fmt.Sprintf("Federated client credential %d has an invalid public key: %v", i+1, err))
		}
		if len(publicKeys) > 0 && fi.JWKS != "" {
			return apperror.ValidationMessage(fmt.Sprintf("Federated client credential %d must use either a JWKS URL or public keys, but not both", i+1))
		}

		federatedIdentities[i] = model.OidcClientFederatedIdentity{
			Issuer:           fi.Issuer,
			Audience:         fi.Audience,
			Subject:          fi.Subject,
			JWKS:             fi.JWKS,
			PublicKeys:       publicKeys,
			ReplayProtection: fi.ReplayProtection,
		}
	}
	client.Credentials.FederatedIdentities = federatedIdentities

	return nil
}

func (s *OidcService) DeleteClient(ctx context.Context, clientID string) error {
	var client model.OidcClient
	result := s.db.
		WithContext(ctx).
		Where("id = ?", clientID).
		Clauses(clause.Returning{}).
		Delete(&client)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("OIDC client")
	}

	// Delete images if present
	// Note that storage operations must be done outside of a transaction
	if client.ImageType != nil && *client.ImageType != "" {
		old := oidcClientImagePath(client.ID, "", *client.ImageType)
		_ = s.fileStorage.Delete(ctx, old)
	}
	if client.DarkImageType != nil && *client.DarkImageType != "" {
		old := oidcClientImagePath(client.ID, "-dark", *client.DarkImageType)
		_ = s.fileStorage.Delete(ctx, old)
	}

	return nil
}

// ListClientSecrets returns all secrets configured for a client, including the expired ones
func (s *OidcService) ListClientSecrets(ctx context.Context, clientID string) ([]model.OidcClientSecret, error) {
	client, err := s.getClientInternal(ctx, clientID, s.db, false)
	if err != nil {
		return nil, err
	}

	return client.Credentials.Secrets, nil
}

// CreateClientSecret adds a new secret to a client and returns both the stored record and the secret's value, which is not recoverable afterwards
func (s *OidcService) CreateClientSecret(ctx context.Context, clientID string, input dto.OidcClientSecretCreateDto) (model.OidcClientSecret, string, error) {
	// An expiration date in the past would create a secret that can never be used
	if input.ExpiresAt != nil && !input.ExpiresAt.ToTime().After(time.Now()) {
		return model.OidcClientSecret{}, "", apperror.ValidationMessage("The expiration date of a client secret must be in the future")
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err := s.getClientInternal(ctx, clientID, tx, true)
	if err != nil {
		return model.OidcClientSecret{}, "", fmt.Errorf("error retrieving OIDC client: %w", err)
	}

	if client.IsPublic {
		return model.OidcClientSecret{}, "", apperror.ValidationMessage("Cannot create a secret for a public client")
	}

	if len(client.Credentials.Secrets) >= model.MaxOidcClientSecrets {
		return model.OidcClientSecret{}, "", apperror.ValidationMessage(fmt.Sprintf("A client cannot have more than %d secrets", model.MaxOidcClientSecrets))
	}

	// Callers may supply their own value, otherwise one with enough entropy is generated here
	clientSecret := input.Secret
	if clientSecret == "" {
		clientSecret, err = utils.GenerateRandomAlphanumericString(32)
		if err != nil {
			return model.OidcClientSecret{}, "", fmt.Errorf("failed to generate client secret: %w", err)
		}
	}

	// Only the hash and a short prefix are persisted, so this is the last time the value is available
	secret := model.OidcClientSecret{
		ID:        uuid.New().String(),
		Algorithm: model.OidcClientSecretHashSHA256,
		Hash:      utils.CreateSha256Hash(clientSecret),
		Prefix:    clientSecretPrefix(clientSecret),
		CreatedAt: datatype.DateTime(time.Now()),
		ExpiresAt: input.ExpiresAt,
	}
	client.Credentials.Secrets = append(client.Credentials.Secrets, secret)

	err = tx.
		WithContext(ctx).
		Model(&client).
		Select("Credentials").
		Updates(&client).
		Error
	if err != nil {
		return model.OidcClientSecret{}, "", fmt.Errorf("failed to update OIDC client: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.OidcClientSecret{}, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return secret, clientSecret, nil
}

// DeleteClientSecret removes a single secret from a client, making it immediately unusable
func (s *OidcService) DeleteClientSecret(ctx context.Context, clientID string, secretID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err := s.getClientInternal(ctx, clientID, tx, true)
	if err != nil {
		return fmt.Errorf("error retrieving OIDC client: %w", err)
	}

	countBefore := len(client.Credentials.Secrets)
	client.Credentials.Secrets = slices.DeleteFunc(client.Credentials.Secrets, func(secret model.OidcClientSecret) bool {
		return secret.ID == secretID
	})
	if len(client.Credentials.Secrets) == countBefore {
		return apperror.NotFound("Client secret")
	}

	err = tx.
		WithContext(ctx).
		Model(&client).
		Select("Credentials").
		Updates(&client).
		Error
	if err != nil {
		return fmt.Errorf("failed to update OIDC client: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// clientSecretPrefix returns the leading characters of a secret that are stored in clear text to help admins tell secrets apart
func clientSecretPrefix(clientSecret string) string {
	if len(clientSecret) <= model.OidcClientSecretPrefixLength {
		return ""
	}

	return clientSecret[:model.OidcClientSecretPrefixLength]
}

func (s *OidcService) GetClientLogo(ctx context.Context, clientID string, light bool) (io.ReadCloser, int64, string, error) {
	client, err := s.getClientInternal(ctx, clientID, s.db, false)
	if err != nil {
		return nil, 0, "", err
	}

	var suffix string
	var ext string
	switch {
	case !light && client.DarkImageType != nil:
		// Dark logo if requested and exists
		suffix = "-dark"
		ext = *client.DarkImageType
	case client.ImageType != nil:
		// Light logo if requested or no dark logo is available
		ext = *client.ImageType
	default:
		return nil, 0, "", apperror.ImageNotFound()
	}

	mimeType := utils.GetImageMimeType(ext)
	if mimeType == "" {
		return nil, 0, "", fmt.Errorf("unsupported image type '%s'", ext)
	}
	key := oidcClientImagePath(client.ID, suffix, ext)
	reader, size, err := s.fileStorage.Open(ctx, key)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, 0, "", apperror.ImageNotFound()
		}
		return nil, 0, "", err
	}

	return reader, size, mimeType, nil
}

func (s *OidcService) UpdateClientLogo(ctx context.Context, clientID string, file *multipart.FileHeader, light bool) error {
	fileType := strings.ToLower(utils.GetFileExtension(file.Filename))
	if mimeType := utils.GetImageMimeType(fileType); mimeType == "" {
		return apperror.UnsupportedFileType("")
	}

	var darkSuffix string
	if !light {
		darkSuffix = "-dark"
	}

	imagePath := oidcClientImagePath(clientID, darkSuffix, fileType)
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	strippedReader, err := imageutil.StripMetadata(reader, fileType)
	if err != nil {
		return err
	}

	err = s.fileStorage.Save(ctx, imagePath, strippedReader)
	if err != nil {
		return err
	}

	err = s.updateClientLogoType(ctx, clientID, fileType, light)
	if err != nil {
		return err
	}

	return nil
}

func (s *OidcService) DeleteClientLogo(ctx context.Context, clientID string) error {
	return s.deleteClientLogoInternal(ctx, clientID, "", func(client *model.OidcClient) (string, error) {
		if client.ImageType == nil {
			return "", apperror.ImageNotFound()
		}

		oldImageType := *client.ImageType
		client.ImageType = nil
		return oldImageType, nil
	})
}

func (s *OidcService) DeleteClientDarkLogo(ctx context.Context, clientID string) error {
	return s.deleteClientLogoInternal(ctx, clientID, "-dark", func(client *model.OidcClient) (string, error) {
		if client.DarkImageType == nil {
			return "", apperror.ImageNotFound()
		}

		oldImageType := *client.DarkImageType
		client.DarkImageType = nil
		return oldImageType, nil
	})
}

func (s *OidcService) deleteClientLogoInternal(ctx context.Context, clientID string, imagePathSuffix string, setClientImage func(*model.OidcClient) (string, error)) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err := s.getClientInternal(ctx, clientID, tx, true)
	if err != nil {
		return err
	}

	oldImageType, err := setClientImage(&client)
	if err != nil {
		return err
	}

	err = tx.
		WithContext(ctx).
		Save(&client).
		Error
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	// All storage operations must be performed outside of a database transaction
	imagePath := oidcClientImagePath(client.ID, imagePathSuffix, oldImageType)
	err = s.fileStorage.Delete(ctx, imagePath)
	if err != nil {
		return err
	}

	return nil
}

func (s *OidcService) UpdateAllowedUserGroups(ctx context.Context, id string, input dto.OidcUpdateAllowedUserGroupsDto) (client model.OidcClient, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err = s.getClientInternal(ctx, id, tx, true)
	if err != nil {
		return model.OidcClient{}, err
	}

	// Fetch the user groups based on UserGroupIDs in input
	var groups []model.UserGroup
	if len(input.UserGroupIDs) > 0 {
		err = tx.
			WithContext(ctx).
			Where("id IN (?)", input.UserGroupIDs).
			Find(&groups).
			Error
		if err != nil {
			return model.OidcClient{}, err
		}
	}

	// Replace the current user groups with the new set of user groups
	err = tx.
		WithContext(ctx).
		Model(&client).
		Association("AllowedUserGroups").
		Replace(groups)
	if err != nil {
		return model.OidcClient{}, err
	}

	// Save the updated client
	err = tx.
		WithContext(ctx).
		Save(&client).
		Error
	if err != nil {
		return model.OidcClient{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.OidcClient{}, err
	}

	if s.scimSyncScheduler != nil {
		s.scimSyncScheduler.ScheduleSync(ctx)
	}
	return client, nil
}

func (s *OidcService) ListAuthorizedClients(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]model.UserAuthorizedOidcClient, utils.PaginationResponse, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var user model.User
	err := tx.
		WithContext(ctx).
		Select("id").
		First(&user, "id = ?", userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, utils.PaginationResponse{}, apperror.UserNotFound()
	}
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	query := tx.
		WithContext(ctx).
		Model(&model.UserAuthorizedOidcClient{}).
		Preload("Client").
		Where("user_id = ?", userID)

	// Apply the launch URL filter before pagination so hidden authorizations have their own page count
	if hasLaunchURL, ok := getHasLaunchURLFilter(listRequestOptions); ok {
		query = query.Joins("JOIN oidc_clients ON oidc_clients.id = user_authorized_oidc_clients.client_id")
		if hasLaunchURL {
			query = query.Where("oidc_clients.launch_url IS NOT NULL AND oidc_clients.launch_url <> ''")
		} else {
			query = query.Where("oidc_clients.launch_url IS NULL OR oidc_clients.launch_url = ''")
		}
	}

	var authorizedClients []model.UserAuthorizedOidcClient
	response, err := utils.PaginateFilterAndSort(listRequestOptions, query, &authorizedClients)

	return authorizedClients, response, err
}

func (s *OidcService) RevokeAuthorizedClient(ctx context.Context, userID string, clientID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var authorizedClient model.UserAuthorizedOidcClient
	err := tx.
		WithContext(ctx).
		Where("user_id = ? AND client_id = ?", userID, clientID).
		First(&authorizedClient).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.NotFound("Client authorization")
	}
	if err != nil {
		return err
	}

	err = tx.WithContext(ctx).Delete(&authorizedClient).Error
	if err != nil {
		return err
	}

	if err = oidc.RevokeUserClientSessions(ctx, tx, userID, clientID); err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	return nil
}

func (s *OidcService) ListAccessibleOidcClients(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]dto.AccessibleOidcClientDto, utils.PaginationResponse, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var user model.User
	err := tx.
		WithContext(ctx).
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, utils.PaginationResponse{}, apperror.UserNotFound()
	}
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	userGroupIDs := make([]string, len(user.UserGroups))
	for i, group := range user.UserGroups {
		userGroupIDs[i] = group.ID
	}

	// Build the query for accessible clients
	query := tx.
		WithContext(ctx).
		Model(&model.OidcClient{}).
		Preload("UserAuthorizedOidcClients", "user_id = ?", userID).
		Where(`oidc_clients.is_group_restricted = ? OR EXISTS (
			SELECT 1 FROM oidc_clients_allowed_user_groups
			WHERE oidc_clients_allowed_user_groups.oidc_client_id = oidc_clients.id
			AND oidc_clients_allowed_user_groups.user_group_id IN (?))`, false, userGroupIDs)

	// Apply the launch URL filter before pagination so the app launcher never contains empty pages
	if hasLaunchURL, ok := getHasLaunchURLFilter(listRequestOptions); ok {
		if hasLaunchURL {
			query = query.Where("oidc_clients.launch_url IS NOT NULL AND oidc_clients.launch_url <> ''")
		} else {
			query = query.Where("oidc_clients.launch_url IS NULL OR oidc_clients.launch_url = ''")
		}
	}

	var clients []model.OidcClient

	// Handle custom sorting for lastUsedAt column
	var response utils.PaginationResponse
	if listRequestOptions.Sort.Column == "lastUsedAt" && utils.IsValidSortDirection(listRequestOptions.Sort.Direction) {
		query = query.
			Joins("LEFT JOIN user_authorized_oidc_clients ON oidc_clients.id = user_authorized_oidc_clients.client_id AND user_authorized_oidc_clients.user_id = ?", userID).
			Order("user_authorized_oidc_clients.last_used_at " + listRequestOptions.Sort.Direction + " NULLS LAST")
	}

	response, err = utils.PaginateFilterAndSort(listRequestOptions, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	dtos := make([]dto.AccessibleOidcClientDto, len(clients))
	for i, client := range clients {
		var lastUsedAt *datatype.DateTime
		if len(client.UserAuthorizedOidcClients) > 0 {
			lastUsedAt = &client.UserAuthorizedOidcClients[0].LastUsedAt
		}
		dtos[i] = dto.AccessibleOidcClientDto{
			OidcClientMetaDataDto: dto.OidcClientMetaDataDto{
				ID:          client.ID,
				Name:        client.Name,
				Description: client.Description,
				LaunchURL:   client.LaunchURL,
				HasLogo:     client.HasLogo(),
				HasDarkLogo: client.HasDarkLogo(),
				ClientType:  string(client.ClientType),
			},
			LastUsedAt: lastUsedAt,
		}
	}

	return dtos, response, err
}

func getHasLaunchURLFilter(listRequestOptions utils.ListRequestOptions) (bool, bool) {
	values := listRequestOptions.Filters["hasLaunchURL"]
	if len(values) == 0 {
		return false, false
	}

	hasLaunchURL, ok := values[0].(bool)
	return hasLaunchURL, ok
}

func (s *OidcService) GetClientPreview(ctx context.Context, clientID string, userID string, scopes []string, authenticationMethod string) (*dto.OidcClientPreviewDto, error) {
	client, err := s.getClientInternal(ctx, clientID, s.db, false)
	if err != nil {
		return nil, err
	}

	var user model.User
	err = s.db.
		WithContext(ctx).
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.UserNotFound()
	}
	if err != nil {
		return nil, err
	}

	if !oidc.IsUserGroupAllowedToAuthorize(user, client) {
		return nil, apperror.OidcAccessDenied()
	}

	preview, err := s.previewBuilder.BuildClientPreview(ctx, client, userID, scopes, authenticationMethod)
	if err != nil {
		return nil, err
	}
	return &dto.OidcClientPreviewDto{
		IdToken:     preview.IDToken,
		AccessToken: preview.AccessToken,
		UserInfo:    preview.UserInfo,
	}, nil
}

func httpClientWithCheckRedirect(source *http.Client, checkRedirect func(req *http.Request, via []*http.Request) error) *http.Client {
	if source == nil {
		source = http.DefaultClient
	}

	// Create a new client that clones the transport
	client := &http.Client{
		Transport: source.Transport,
	}

	// Assign the CheckRedirect function
	client.CheckRedirect = checkRedirect

	return client
}

func (s *OidcService) downloadAndSaveLogoFromURL(parentCtx context.Context, clientID string, raw string, light bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return apperror.InvalidLogoURL(err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return apperror.InvalidLogoURL(fmt.Errorf("URL must use HTTP or HTTPS and include a host"))
	}

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	// Prevents SSRF by allowing only public IPs
	ok, err := utils.IsURLPrivate(ctx, u)
	if err != nil {
		return apperror.LogoDownloadFailed(err)
	} else if ok {
		return apperror.InvalidLogoURL(errors.New("private IP addresses are not allowed"))
	}

	// We need to check this on redirects too
	client := httpClientWithCheckRedirect(s.httpClient, func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return apperror.InvalidLogoURL(errors.New("stopped after 10 redirects"))
		}

		ok, err := utils.IsURLPrivate(r.Context(), r.URL)
		if err != nil {
			return err
		} else if ok {
			return apperror.InvalidLogoURL(errors.New("private IP addresses are not allowed"))
		}

		return nil
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return apperror.InvalidLogoURL(err)
	}
	req.Header.Set("User-Agent", "pocket-id/oidc-logo-fetcher")
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		if appErr, ok := errors.AsType[*apperror.Error](err); ok {
			return appErr
		}
		return apperror.LogoDownloadFailed(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperror.LogoDownloadFailed(fmt.Errorf("logo server returned %s", resp.Status))
	}

	const maxLogoSize int64 = 2 * 1024 * 1024 // 2MB
	if resp.ContentLength > maxLogoSize {
		return apperror.LogoTooLarge("2 MB")
	}

	// Prefer extension in path if supported
	ext := utils.GetFileExtension(u.Path)
	if ext == "" || utils.GetImageMimeType(ext) == "" {
		// Otherwise, try to detect from content type
		ext = utils.GetImageExtensionFromMimeType(resp.Header.Get("Content-Type"))
	}

	if ext == "" {
		return apperror.LogoTypeNotSupported()
	}

	var darkSuffix string
	if !light {
		darkSuffix = "-dark"
	}

	limitReader := utils.NewLimitReader(resp.Body, maxLogoSize+1)
	strippedReader, err := imageutil.StripMetadata(limitReader, ext)
	if errors.Is(err, utils.ErrSizeExceeded) {
		return apperror.LogoTooLarge("2 MB")
	} else if err != nil {
		return apperror.LogoDownloadFailed(err)
	}

	imagePath := oidcClientImagePath(clientID, darkSuffix, ext)
	err = s.fileStorage.Save(ctx, imagePath, strippedReader)
	if errors.Is(err, utils.ErrSizeExceeded) {
		return apperror.LogoTooLarge("2 MB")
	} else if err != nil {
		return apperror.LogoDownloadFailed(err)
	}

	err = s.updateClientLogoType(ctx, clientID, ext, light)
	if err != nil {
		return err
	}

	return nil
}

func (s *OidcService) updateClientLogoType(ctx context.Context, clientID string, ext string, light bool) error {
	var darkSuffix string
	if !light {
		darkSuffix = "-dark"
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// We need to acquire an update lock for the row to be locked, since we'll update it later
	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("OIDC client")
		}
		return fmt.Errorf("failed to look up client: %w", err)
	}

	var currentType *string
	if light {
		currentType = client.ImageType
		client.ImageType = &ext
	} else {
		currentType = client.DarkImageType
		client.DarkImageType = &ext
	}

	err = tx.
		WithContext(ctx).
		Save(&client).
		Error
	if err != nil {
		return fmt.Errorf("failed to save updated client: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Storage operations must be executed outside of a transaction
	if currentType != nil && *currentType != ext {
		old := oidcClientImagePath(client.ID, darkSuffix, *currentType)
		_ = s.fileStorage.Delete(ctx, old)
	}

	return nil
}

func oidcClientImagePath(clientID string, suffix string, extension string) string {
	storageID := clientID
	if !dto.ValidateClientID(clientID) {
		storageID = "cimd-" + utils.CreateSha256Hash(clientID)
	}
	return path.Join("oidc-client-images", storageID+suffix+"."+extension)
}
