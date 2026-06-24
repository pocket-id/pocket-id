package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	imageutil "github.com/pocket-id/pocket-id/backend/internal/utils/image"
)

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeClientCredentials = "client_credentials"

	AccessTokenDuration  = time.Hour
	RefreshTokenDuration = 30 * 24 * time.Hour // 30 days
)

type OidcService struct {
	db               *gorm.DB
	jwtService       *JwtService
	appConfigService *AppConfigService
	previewBuilder   oidcClientPreviewBuilder
	scimService      *ScimService

	httpClient  *http.Client
	fileStorage storage.FileStorage
}

type oidcClientPreviewBuilder interface {
	BuildClientPreview(ctx context.Context, client model.OidcClient, userID string, scopes []string, authenticationMethod string) (*oidc.ClientPreview, error)
}

func NewOidcService(
	db *gorm.DB,
	jwtService *JwtService,
	appConfigService *AppConfigService,
	previewBuilder oidcClientPreviewBuilder,
	scimService *ScimService,
	httpClient *http.Client,
	fileStorage storage.FileStorage,
) (s *OidcService, err error) {
	s = &OidcService{
		db:               db,
		jwtService:       jwtService,
		appConfigService: appConfigService,
		previewBuilder:   previewBuilder,
		scimService:      scimService,
		httpClient:       httpClient,
		fileStorage:      fileStorage,
	}

	return s, nil
}

func (s *OidcService) GetClient(ctx context.Context, clientID string) (model.OidcClient, error) {
	return s.getClientInternal(ctx, clientID, s.db, false)
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
		Model(&model.OidcClient{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// As allowedUserGroupsCount is not a column, we need to manually sort it
	if listRequestOptions.Sort.Column == "allowedUserGroupsCount" && utils.IsValidSortDirection(listRequestOptions.Sort.Direction) {
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
	updateOIDCClientModelFromDto(&client, &input.OidcClientUpdateDto)

	err := s.db.
		WithContext(ctx).
		Create(&client).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.OidcClient{}, &common.ClientIdAlreadyExistsError{}
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

	var client model.OidcClient
	err := tx.WithContext(ctx).
		Preload("CreatedBy").
		First(&client, "id = ?", clientID).Error
	if err != nil {
		return model.OidcClient{}, err
	}

	updateOIDCClientModelFromDto(&client, &input)

	if !input.IsGroupRestricted {
		// Clear allowed user groups if the restriction is removed
		err = tx.Model(&client).Association("AllowedUserGroups").Clear()
		if err != nil {
			return model.OidcClient{}, err
		}
	}

	err = tx.WithContext(ctx).Save(&client).Error
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

func updateOIDCClientModelFromDto(client *model.OidcClient, input *dto.OidcClientUpdateDto) {
	// Base fields
	client.Name = input.Name
	client.Description = input.Description
	client.CallbackURLs = input.CallbackURLs
	client.LogoutCallbackURLs = input.LogoutCallbackURLs
	client.IsPublic = input.IsPublic
	// PKCE is required for public clients
	client.PkceEnabled = input.IsPublic || input.PkceEnabled
	client.RequiresReauthentication = input.RequiresReauthentication
	client.RequiresPushedAuthorizationRequests = input.RequiresPushedAuthorizationRequests
	client.LaunchURL = input.LaunchURL
	client.IsGroupRestricted = input.IsGroupRestricted

	// Credentials
	client.Credentials.FederatedIdentities = make([]model.OidcClientFederatedIdentity, len(input.Credentials.FederatedIdentities))
	for i, fi := range input.Credentials.FederatedIdentities {
		client.Credentials.FederatedIdentities[i] = model.OidcClientFederatedIdentity{
			Issuer:           fi.Issuer,
			Audience:         fi.Audience,
			Subject:          fi.Subject,
			JWKS:             fi.JWKS,
			ReplayProtection: fi.ReplayProtection,
		}
	}

}

func (s *OidcService) DeleteClient(ctx context.Context, clientID string) error {
	var client model.OidcClient
	err := s.db.
		WithContext(ctx).
		Where("id = ?", clientID).
		Clauses(clause.Returning{}).
		Delete(&client).
		Error
	if err != nil {
		return err
	}

	// Delete images if present
	// Note that storage operations must be done outside of a transaction
	if client.ImageType != nil && *client.ImageType != "" {
		old := path.Join("oidc-client-images", client.ID+"."+*client.ImageType)
		_ = s.fileStorage.Delete(ctx, old)
	}
	if client.DarkImageType != nil && *client.DarkImageType != "" {
		old := path.Join("oidc-client-images", client.ID+"-dark."+*client.DarkImageType)
		_ = s.fileStorage.Delete(ctx, old)
	}

	return nil
}

func (s *OidcService) CreateClientSecret(ctx context.Context, clientID string) (string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		return "", err
	}

	clientSecret, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", err
	}

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	client.Secret = string(hashedSecret)
	err = tx.
		WithContext(ctx).
		Save(&client).
		Error
	if err != nil {
		return "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return "", err
	}

	return clientSecret, nil
}

func (s *OidcService) GetClientLogo(ctx context.Context, clientID string, light bool) (io.ReadCloser, int64, string, error) {
	var client model.OidcClient
	err := s.db.
		WithContext(ctx).
		First(&client, "id = ?", clientID).
		Error
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
		return nil, 0, "", errors.New("image not found")
	}

	mimeType := utils.GetImageMimeType(ext)
	if mimeType == "" {
		return nil, 0, "", fmt.Errorf("unsupported image type '%s'", ext)
	}
	key := path.Join("oidc-client-images", client.ID+suffix+"."+ext)
	reader, size, err := s.fileStorage.Open(ctx, key)
	if err != nil {
		return nil, 0, "", err
	}

	return reader, size, mimeType, nil
}

func (s *OidcService) UpdateClientLogo(ctx context.Context, clientID string, file *multipart.FileHeader, light bool) error {
	fileType := strings.ToLower(utils.GetFileExtension(file.Filename))
	if mimeType := utils.GetImageMimeType(fileType); mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	var darkSuffix string
	if !light {
		darkSuffix = "-dark"
	}

	imagePath := path.Join("oidc-client-images", clientID+darkSuffix+"."+fileType)
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
			return "", errors.New("image not found")
		}

		oldImageType := *client.ImageType
		client.ImageType = nil
		return oldImageType, nil
	})
}

func (s *OidcService) DeleteClientDarkLogo(ctx context.Context, clientID string) error {
	return s.deleteClientLogoInternal(ctx, clientID, "-dark", func(client *model.OidcClient) (string, error) {
		if client.DarkImageType == nil {
			return "", errors.New("image not found")
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

	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		First(&client, "id = ?", clientID).
		Error
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
	imagePath := path.Join("oidc-client-images", client.ID+imagePathSuffix+"."+oldImageType)
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

	s.scimService.ScheduleSync()
	return client, nil
}

func (s *OidcService) GetAllowedGroupsCountOfClient(ctx context.Context, id string) (int64, error) {
	// We only perform select queries here, so we can rollback in all cases
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var client model.OidcClient
	err := tx.WithContext(ctx).Where("id = ?", id).First(&client).Error
	if err != nil {
		return 0, err
	}

	count := tx.WithContext(ctx).Model(&client).Association("AllowedUserGroups").Count()
	return count, nil
}

func (s *OidcService) ListAuthorizedClients(ctx context.Context, userID string, listRequestOptions utils.ListRequestOptions) ([]model.UserAuthorizedOidcClient, utils.PaginationResponse, error) {

	query := s.db.
		WithContext(ctx).
		Model(&model.UserAuthorizedOidcClient{}).
		Preload("Client").
		Where("user_id = ?", userID)

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
		Preload("UserAuthorizedOidcClients", "user_id = ?", userID)

	// If user has no groups, only return clients with no allowed user groups
	if len(userGroupIDs) == 0 {
		query = query.Where(`NOT EXISTS (
        SELECT 1 FROM oidc_clients_allowed_user_groups 
        WHERE oidc_clients_allowed_user_groups.oidc_client_id = oidc_clients.id)`)
	} else {
		query = query.Where(`
        NOT EXISTS (
            SELECT 1 FROM oidc_clients_allowed_user_groups 
            WHERE oidc_clients_allowed_user_groups.oidc_client_id = oidc_clients.id
        ) OR EXISTS (
            SELECT 1 FROM oidc_clients_allowed_user_groups 
            WHERE oidc_clients_allowed_user_groups.oidc_client_id = oidc_clients.id 
            AND oidc_clients_allowed_user_groups.user_group_id IN (?))`, userGroupIDs)
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
				LaunchURL:   client.LaunchURL,
				HasLogo:     client.HasLogo(),
				HasDarkLogo: client.HasDarkLogo(),
			},
			LastUsedAt: lastUsedAt,
		}
	}

	return dtos, response, err
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
	if err != nil {
		return nil, err
	}

	if !oidc.IsUserGroupAllowedToAuthorize(user, client) {
		return nil, &common.OidcAccessDeniedError{}
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

var errLogoTooLarge = errors.New("logo is too large")

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
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	// Prevents SSRF by allowing only public IPs
	ok, err := utils.IsURLPrivate(ctx, u)
	if err != nil {
		return err
	} else if ok {
		return errors.New("private IP addresses are not allowed")
	}

	// We need to check this on redirects too
	client := httpClientWithCheckRedirect(s.httpClient, func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		ok, err := utils.IsURLPrivate(r.Context(), r.URL)
		if err != nil {
			return err
		} else if ok {
			return errors.New("private IP addresses are not allowed")
		}

		return nil
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pocket-id/oidc-logo-fetcher")
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch logo: %s", resp.Status)
	}

	const maxLogoSize int64 = 2 * 1024 * 1024 // 2MB
	if resp.ContentLength > maxLogoSize {
		return errLogoTooLarge
	}

	// Prefer extension in path if supported
	ext := utils.GetFileExtension(u.Path)
	if ext == "" || utils.GetImageMimeType(ext) == "" {
		// Otherwise, try to detect from content type
		ext = utils.GetImageExtensionFromMimeType(resp.Header.Get("Content-Type"))
	}

	if ext == "" {
		return &common.FileTypeNotSupportedError{}
	}

	var darkSuffix string
	if !light {
		darkSuffix = "-dark"
	}

	limitReader := utils.NewLimitReader(resp.Body, maxLogoSize+1)
	strippedReader, err := imageutil.StripMetadata(limitReader, ext)
	if errors.Is(err, utils.ErrSizeExceeded) {
		return errLogoTooLarge
	} else if err != nil {
		return err
	}

	imagePath := path.Join("oidc-client-images", clientID+darkSuffix+"."+ext)
	err = s.fileStorage.Save(ctx, imagePath, strippedReader)
	if errors.Is(err, utils.ErrSizeExceeded) {
		return errLogoTooLarge
	} else if err != nil {
		return err
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
		old := path.Join("oidc-client-images", client.ID+darkSuffix+"."+*currentType)
		_ = s.fileStorage.Delete(ctx, old)
	}

	return nil
}

func (s *OidcService) GetClientScimServiceProvider(ctx context.Context, clientID string) (model.ScimServiceProvider, error) {
	var provider model.ScimServiceProvider
	err := s.db.
		WithContext(ctx).
		First(&provider, "oidc_client_id = ?", clientID).
		Error
	if err != nil {
		return model.ScimServiceProvider{}, err
	}

	return provider, nil
}
