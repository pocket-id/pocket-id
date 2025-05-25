package service

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/httprc/v3/errsink"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"

	ClientAssertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	TokenExchangeAudience = "api://PocketIDTokenExchange"
)

type OidcService struct {
	db                 *gorm.DB
	jwtService         *JwtService
	appConfigService   *AppConfigService
	auditLogService    *AuditLogService
	customClaimService *CustomClaimService

	httpClient *http.Client
	jwkCache   *jwk.Cache
}

func NewOidcService(
	ctx context.Context,
	db *gorm.DB,
	jwtService *JwtService,
	appConfigService *AppConfigService,
	auditLogService *AuditLogService,
	customClaimService *CustomClaimService,
) (s *OidcService, err error) {
	s = &OidcService{
		db:                 db,
		jwtService:         jwtService,
		appConfigService:   appConfigService,
		auditLogService:    auditLogService,
		customClaimService: customClaimService,
	}

	// Note: we don't pass the HTTP Client with OTel instrumented to this because requests are always made in background and not tied to a specific trace
	s.jwkCache, err = s.getJWKCache(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *OidcService) getJWKCache(ctx context.Context) (*jwk.Cache, error) {
	// We need to create a custom HTTP client to set a timeout.
	client := s.httpClient
	if client == nil {
		client = &http.Client{
			Timeout: 20 * time.Second,
		}

		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			// Indicates a development-time error
			panic("Default transport is not of type *http.Transport")
		}
		transport := defaultTransport.Clone()
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
		client.Transport = transport
	}

	// Create the JWKS cache
	return jwk.NewCache(ctx,
		httprc.NewClient(
			httprc.WithErrorSink(errsink.NewSlog(slog.Default())),
			httprc.WithHTTPClient(client),
		),
	)
}

func (s *OidcService) Authorize(ctx context.Context, input dto.AuthorizeOidcClientRequestDto, userID, ipAddress, userAgent string) (string, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		Preload("AllowedUserGroups").
		First(&client, "id = ?", input.ClientID).
		Error
	if err != nil {
		return "", "", err
	}

	// If the client is not public, the code challenge must be provided
	if client.IsPublic && input.CodeChallenge == "" {
		return "", "", &common.OidcMissingCodeChallengeError{}
	}

	// Get the callback URL of the client. Return an error if the provided callback URL is not allowed
	callbackURL, err := s.getCallbackURL(client.CallbackURLs, input.CallbackURL)
	if err != nil {
		return "", "", err
	}

	// Check if the user group is allowed to authorize the client
	var user model.User
	err = tx.
		WithContext(ctx).
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if err != nil {
		return "", "", err
	}

	if !s.IsUserGroupAllowedToAuthorize(user, client) {
		return "", "", &common.OidcAccessDeniedError{}
	}

	// Check if the user has already authorized the client with the given scope
	hasAuthorizedClient, err := s.hasAuthorizedClientInternal(ctx, input.ClientID, userID, input.Scope, tx)
	if err != nil {
		return "", "", err
	}

	// If the user has not authorized the client, create a new authorization in the database
	if !hasAuthorizedClient {
		err := s.createAuthorizedClientInternal(ctx, userID, input.ClientID, input.Scope, tx)
		if err != nil {
			return "", "", err
		}
	}

	// Create the authorization code
	code, err := s.createAuthorizationCode(ctx, input.ClientID, userID, input.Scope, input.Nonce, input.CodeChallenge, input.CodeChallengeMethod, tx)
	if err != nil {
		return "", "", err
	}

	// Log the authorization event
	if hasAuthorizedClient {
		s.auditLogService.Create(ctx, model.AuditLogEventClientAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": client.Name}, tx)
	} else {
		s.auditLogService.Create(ctx, model.AuditLogEventNewClientAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": client.Name}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return "", "", err
	}

	return code, callbackURL, nil
}

// HasAuthorizedClient checks if the user has already authorized the client with the given scope
func (s *OidcService) HasAuthorizedClient(ctx context.Context, clientID, userID, scope string) (bool, error) {
	return s.hasAuthorizedClientInternal(ctx, clientID, userID, scope, s.db)
}

func (s *OidcService) hasAuthorizedClientInternal(ctx context.Context, clientID, userID, scope string, tx *gorm.DB) (bool, error) {
	var userAuthorizedOidcClient model.UserAuthorizedOidcClient
	err := tx.
		WithContext(ctx).
		First(&userAuthorizedOidcClient, "client_id = ? AND user_id = ?", clientID, userID).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if userAuthorizedOidcClient.Scope != scope {
		return false, nil
	}

	return true, nil
}

// IsUserGroupAllowedToAuthorize checks if the user group of the user is allowed to authorize the client
func (s *OidcService) IsUserGroupAllowedToAuthorize(user model.User, client model.OidcClient) bool {
	if len(client.AllowedUserGroups) == 0 {
		return true
	}

	isAllowedToAuthorize := false
	for _, userGroup := range client.AllowedUserGroups {
		for _, userGroupUser := range user.UserGroups {
			if userGroup.ID == userGroupUser.ID {
				isAllowedToAuthorize = true
				break
			}
		}
	}

	return isAllowedToAuthorize
}

type CreatedTokens struct {
	IdToken      string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

func (s *OidcService) CreateTokens(ctx context.Context, input dto.OidcCreateTokensDto) (CreatedTokens, error) {
	switch input.GrantType {
	case GrantTypeAuthorizationCode:
		return s.createTokenFromAuthorizationCode(ctx, input)
	case GrantTypeRefreshToken:
		return s.createTokenFromRefreshToken(ctx, input)
	case GrantTypeDeviceCode:
		return s.createTokenFromDeviceCode(ctx, input)
	default:
		return CreatedTokens{}, &common.OidcGrantTypeNotSupportedError{}
	}
}

func (s *OidcService) createTokenFromDeviceCode(ctx context.Context, input dto.OidcCreateTokensDto) (CreatedTokens, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	_, err := s.verifyClientCredentialsInternal(ctx, tx, input)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Get the device authorization from database with explicit query conditions
	var deviceAuth model.OidcDeviceCode
	err = tx.
		WithContext(ctx).
		Preload("User").
		Where("device_code = ? AND client_id = ?", input.DeviceCode, input.ClientID).
		First(&deviceAuth).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreatedTokens{}, &common.OidcInvalidDeviceCodeError{}
		}
		return CreatedTokens{}, err
	}

	// Check if device code has expired
	if time.Now().After(deviceAuth.ExpiresAt.ToTime()) {
		return CreatedTokens{}, &common.OidcDeviceCodeExpiredError{}
	}

	// Check if device code has been authorized
	if !deviceAuth.IsAuthorized || deviceAuth.UserID == nil {
		return CreatedTokens{}, &common.OidcAuthorizationPendingError{}
	}

	// Get user claims for the ID token - ensure UserID is not nil
	if deviceAuth.UserID == nil {
		return CreatedTokens{}, &common.OidcAuthorizationPendingError{}
	}

	userClaims, err := s.getUserClaimsForClientInternal(ctx, *deviceAuth.UserID, input.ClientID, tx)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Explicitly use the input clientID for the audience claim to ensure consistency
	idToken, err := s.jwtService.GenerateIDToken(userClaims, input.ClientID, "")
	if err != nil {
		return CreatedTokens{}, err
	}

	refreshToken, err := s.createRefreshToken(ctx, input.ClientID, *deviceAuth.UserID, deviceAuth.Scope, tx)
	if err != nil {
		return CreatedTokens{}, err
	}

	accessToken, err := s.jwtService.GenerateOauthAccessToken(deviceAuth.User, input.ClientID)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Delete the used device code
	err = tx.WithContext(ctx).Delete(&deviceAuth).Error
	if err != nil {
		return CreatedTokens{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return CreatedTokens{}, err
	}

	return CreatedTokens{
		IdToken:      idToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Hour,
	}, nil
}

func (s *OidcService) createTokenFromAuthorizationCode(ctx context.Context, input dto.OidcCreateTokensDto) (CreatedTokens, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err := s.verifyClientCredentialsInternal(ctx, tx, input)
	if err != nil {
		return CreatedTokens{}, err
	}

	var authorizationCodeMetaData model.OidcAuthorizationCode
	err = tx.
		WithContext(ctx).
		Preload("User").
		First(&authorizationCodeMetaData, "code = ?", input.Code).
		Error
	if err != nil {
		return CreatedTokens{}, &common.OidcInvalidAuthorizationCodeError{}
	}

	// If the client is public or PKCE is enabled, the code verifier must match the code challenge
	if client.IsPublic || client.PkceEnabled {
		if !s.validateCodeVerifier(input.CodeVerifier, *authorizationCodeMetaData.CodeChallenge, *authorizationCodeMetaData.CodeChallengeMethodSha256) {
			return CreatedTokens{}, &common.OidcInvalidCodeVerifierError{}
		}
	}

	if authorizationCodeMetaData.ClientID != input.ClientID && authorizationCodeMetaData.ExpiresAt.ToTime().Before(time.Now()) {
		return CreatedTokens{}, &common.OidcInvalidAuthorizationCodeError{}
	}

	userClaims, err := s.getUserClaimsForClientInternal(ctx, authorizationCodeMetaData.UserID, input.ClientID, tx)
	if err != nil {
		return CreatedTokens{}, err
	}

	idToken, err := s.jwtService.GenerateIDToken(userClaims, input.ClientID, authorizationCodeMetaData.Nonce)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Generate a refresh token
	refreshToken, err := s.createRefreshToken(ctx, input.ClientID, authorizationCodeMetaData.UserID, authorizationCodeMetaData.Scope, tx)
	if err != nil {
		return CreatedTokens{}, err
	}

	accessToken, err := s.jwtService.GenerateOauthAccessToken(authorizationCodeMetaData.User, input.ClientID)
	if err != nil {
		return CreatedTokens{}, err
	}

	err = tx.
		WithContext(ctx).
		Delete(&authorizationCodeMetaData).
		Error
	if err != nil {
		return CreatedTokens{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return CreatedTokens{}, err
	}

	return CreatedTokens{
		IdToken:      idToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Hour,
	}, nil
}

func (s *OidcService) createTokenFromRefreshToken(ctx context.Context, input dto.OidcCreateTokensDto) (CreatedTokens, error) {
	if input.RefreshToken == "" {
		return CreatedTokens{}, &common.OidcMissingRefreshTokenError{}
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	_, err := s.verifyClientCredentialsInternal(ctx, tx, input)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Verify refresh token
	var storedRefreshToken model.OidcRefreshToken
	err = tx.
		WithContext(ctx).
		Preload("User").
		Where("token = ? AND expires_at > ?", utils.CreateSha256Hash(input.RefreshToken), datatype.DateTime(time.Now())).
		First(&storedRefreshToken).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreatedTokens{}, &common.OidcInvalidRefreshTokenError{}
		}
		return CreatedTokens{}, err
	}

	// Verify that the refresh token belongs to the provided client
	if storedRefreshToken.ClientID != input.ClientID {
		return CreatedTokens{}, &common.OidcInvalidRefreshTokenError{}
	}

	// Generate a new access token
	accessToken, err := s.jwtService.GenerateOauthAccessToken(storedRefreshToken.User, input.ClientID)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Generate a new refresh token and invalidate the old one
	newRefreshToken, err := s.createRefreshToken(ctx, input.ClientID, storedRefreshToken.UserID, storedRefreshToken.Scope, tx)
	if err != nil {
		return CreatedTokens{}, err
	}

	// Delete the used refresh token
	err = tx.
		WithContext(ctx).
		Delete(&storedRefreshToken).
		Error
	if err != nil {
		return CreatedTokens{}, err
	}

	err = tx.Commit().Error
	if err != nil {
		return CreatedTokens{}, err
	}

	return CreatedTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    time.Hour,
	}, nil
}

func (s *OidcService) IntrospectToken(ctx context.Context, clientID, clientSecret, tokenString string) (introspectDto dto.OidcIntrospectionResponseDto, err error) {
	if clientID == "" || clientSecret == "" {
		return introspectDto, &common.OidcMissingClientCredentialsError{}
	}

	_, err = s.verifyClientCredentialsInternal(ctx, s.db, dto.OidcCreateTokensDto{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return introspectDto, err
	}

	token, err := s.jwtService.VerifyOauthAccessToken(tokenString)
	if err != nil {
		if errors.Is(err, jwt.ParseError()) {
			// It's apparently not a valid JWT token, so we check if it's a valid refresh_token.
			return s.introspectRefreshToken(ctx, tokenString)
		}

		// Every failure we get means the token is invalid. Nothing more to do with the error.
		introspectDto.Active = false
		return introspectDto, nil
	}

	introspectDto.Active = true
	introspectDto.TokenType = "access_token"
	if token.Has("scope") {
		var (
			asString  string
			asStrings []string
		)
		if err := token.Get("scope", &asString); err == nil {
			introspectDto.Scope = asString
		} else if err := token.Get("scope", &asStrings); err == nil {
			introspectDto.Scope = strings.Join(asStrings, " ")
		}
	}
	if expiration, ok := token.Expiration(); ok {
		introspectDto.Expiration = expiration.Unix()
	}
	if issuedAt, ok := token.IssuedAt(); ok {
		introspectDto.IssuedAt = issuedAt.Unix()
	}
	if notBefore, ok := token.NotBefore(); ok {
		introspectDto.NotBefore = notBefore.Unix()
	}
	if subject, ok := token.Subject(); ok {
		introspectDto.Subject = subject
	}
	if audience, ok := token.Audience(); ok {
		introspectDto.Audience = audience
	}
	if issuer, ok := token.Issuer(); ok {
		introspectDto.Issuer = issuer
	}
	if identifier, ok := token.JwtID(); ok {
		introspectDto.Identifier = identifier
	}

	return introspectDto, nil
}

func (s *OidcService) introspectRefreshToken(ctx context.Context, refreshToken string) (introspectDto dto.OidcIntrospectionResponseDto, err error) {
	var storedRefreshToken model.OidcRefreshToken
	err = s.db.
		WithContext(ctx).
		Preload("User").
		Where("token = ? AND expires_at > ?", utils.CreateSha256Hash(refreshToken), datatype.DateTime(time.Now())).
		First(&storedRefreshToken).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			introspectDto.Active = false
			return introspectDto, nil
		}
		return introspectDto, err
	}

	introspectDto.Active = true
	introspectDto.TokenType = "refresh_token"
	return introspectDto, nil
}

func (s *OidcService) GetClient(ctx context.Context, clientID string) (model.OidcClient, error) {
	return s.getClientInternal(ctx, clientID, s.db)
}

func (s *OidcService) getClientInternal(ctx context.Context, clientID string, tx *gorm.DB) (model.OidcClient, error) {
	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		Preload("CreatedBy").
		Preload("AllowedUserGroups").
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		return model.OidcClient{}, err
	}
	return client, nil
}

func (s *OidcService) ListClients(ctx context.Context, searchTerm string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.OidcClient, utils.PaginationResponse, error) {
	var clients []model.OidcClient

	query := s.db.
		WithContext(ctx).
		Preload("CreatedBy").
		Model(&model.OidcClient{})
	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where("name LIKE ?", searchPattern)
	}

	pagination, err := utils.PaginateAndSort(sortedPaginationRequest, query, &clients)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	return clients, pagination, nil
}

func (s *OidcService) CreateClient(ctx context.Context, input dto.OidcClientCreateDto, userID string) (model.OidcClient, error) {
	client := model.OidcClient{
		Name:               input.Name,
		CallbackURLs:       input.CallbackURLs,
		LogoutCallbackURLs: input.LogoutCallbackURLs,
		CreatedByID:        userID,
		IsPublic:           input.IsPublic,
		PkceEnabled:        input.PkceEnabled,
	}

	err := s.db.
		WithContext(ctx).
		Create(&client).
		Error
	if err != nil {
		return model.OidcClient{}, err
	}

	return client, nil
}

func (s *OidcService) UpdateClient(ctx context.Context, clientID string, input dto.OidcClientCreateDto) (model.OidcClient, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		Preload("CreatedBy").
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		return model.OidcClient{}, err
	}

	client.Name = input.Name
	client.CallbackURLs = input.CallbackURLs
	client.LogoutCallbackURLs = input.LogoutCallbackURLs
	client.IsPublic = input.IsPublic
	client.PkceEnabled = input.IsPublic || input.PkceEnabled

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

	return client, nil
}

func (s *OidcService) DeleteClient(ctx context.Context, clientID string) error {
	var client model.OidcClient
	err := s.db.
		WithContext(ctx).
		Where("id = ?", clientID).
		Delete(&client).
		Error
	if err != nil {
		return err
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

func (s *OidcService) GetClientLogo(ctx context.Context, clientID string) (string, string, error) {
	var client model.OidcClient
	err := s.db.
		WithContext(ctx).
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		return "", "", err
	}

	if client.ImageType == nil {
		return "", "", errors.New("image not found")
	}

	imagePath := common.EnvConfig.UploadPath + "/oidc-client-images/" + client.ID + "." + *client.ImageType
	mimeType := utils.GetImageMimeType(*client.ImageType)

	return imagePath, mimeType, nil
}

func (s *OidcService) UpdateClientLogo(ctx context.Context, clientID string, file *multipart.FileHeader) error {
	fileType := utils.GetFileExtension(file.Filename)
	if mimeType := utils.GetImageMimeType(fileType); mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	imagePath := common.EnvConfig.UploadPath + "/oidc-client-images/" + clientID + "." + fileType
	err := utils.SaveFile(file, imagePath)
	if err != nil {
		return err
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var client model.OidcClient
	err = tx.
		WithContext(ctx).
		First(&client, "id = ?", clientID).
		Error
	if err != nil {
		return err
	}

	if client.ImageType != nil && fileType != *client.ImageType {
		oldImagePath := fmt.Sprintf("%s/oidc-client-images/%s.%s", common.EnvConfig.UploadPath, client.ID, *client.ImageType)
		if err := os.Remove(oldImagePath); err != nil {
			return err
		}
	}

	client.ImageType = &fileType
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

	return nil
}

func (s *OidcService) DeleteClientLogo(ctx context.Context, clientID string) error {
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

	if client.ImageType == nil {
		return errors.New("image not found")
	}

	client.ImageType = nil
	err = tx.
		WithContext(ctx).
		Save(&client).
		Error
	if err != nil {
		return err
	}

	imagePath := common.EnvConfig.UploadPath + "/oidc-client-images/" + client.ID + "." + *client.ImageType
	if err := os.Remove(imagePath); err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	return nil
}

func (s *OidcService) GetUserClaimsForClient(ctx context.Context, userID string, clientID string) (map[string]interface{}, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	claims, err := s.getUserClaimsForClientInternal(ctx, userID, clientID, s.db)
	if err != nil {
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (s *OidcService) getUserClaimsForClientInternal(ctx context.Context, userID string, clientID string, tx *gorm.DB) (map[string]interface{}, error) {
	var authorizedOidcClient model.UserAuthorizedOidcClient
	err := tx.
		WithContext(ctx).
		Preload("User.UserGroups").
		First(&authorizedOidcClient, "user_id = ? AND client_id = ?", userID, clientID).
		Error
	if err != nil {
		return nil, err
	}

	user := authorizedOidcClient.User
	scopes := strings.Split(authorizedOidcClient.Scope, " ")

	claims := map[string]interface{}{
		"sub": user.ID,
	}

	if slices.Contains(scopes, "email") {
		claims["email"] = user.Email
		claims["email_verified"] = s.appConfigService.GetDbConfig().EmailsVerified.IsTrue()
	}

	if slices.Contains(scopes, "groups") {
		userGroups := make([]string, len(user.UserGroups))
		for i, group := range user.UserGroups {
			userGroups[i] = group.Name
		}
		claims["groups"] = userGroups
	}

	profileClaims := map[string]interface{}{
		"given_name":         user.FirstName,
		"family_name":        user.LastName,
		"name":               user.FullName(),
		"preferred_username": user.Username,
		"picture":            common.EnvConfig.AppURL + "/api/users/" + user.ID + "/profile-picture.png",
	}

	if slices.Contains(scopes, "profile") {
		// Add profile claims
		for k, v := range profileClaims {
			claims[k] = v
		}

		// Add custom claims
		customClaims, err := s.customClaimService.GetCustomClaimsForUserWithUserGroups(ctx, userID, tx)
		if err != nil {
			return nil, err
		}

		for _, customClaim := range customClaims {
			// The value of the custom claim can be a JSON object or a string
			var jsonValue interface{}
			err := json.Unmarshal([]byte(customClaim.Value), &jsonValue)
			if err == nil {
				// It's JSON so we store it as an object
				claims[customClaim.Key] = jsonValue
			} else {
				// Marshalling failed, so we store it as a string
				claims[customClaim.Key] = customClaim.Value
			}
		}
	}

	if slices.Contains(scopes, "email") {
		claims["email"] = user.Email
	}

	return claims, nil
}

func (s *OidcService) UpdateAllowedUserGroups(ctx context.Context, id string, input dto.OidcUpdateAllowedUserGroupsDto) (client model.OidcClient, err error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	client, err = s.getClientInternal(ctx, id, tx)
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

	return client, nil
}

// ValidateEndSession returns the logout callback URL for the client if all the validations pass
func (s *OidcService) ValidateEndSession(ctx context.Context, input dto.OidcLogoutDto, userID string) (string, error) {
	// If no ID token hint is provided, return an error
	if input.IdTokenHint == "" {
		return "", &common.TokenInvalidError{}
	}

	// If the ID token hint is provided, verify the ID token
	// Here we also accept expired ID tokens, which are fine per spec
	token, err := s.jwtService.VerifyIdToken(input.IdTokenHint, true)
	if err != nil {
		return "", &common.TokenInvalidError{}
	}

	// If the client ID is provided check if the client ID in the ID token matches the client ID in the request
	clientID, ok := token.Audience()
	if !ok || len(clientID) == 0 {
		return "", &common.TokenInvalidError{}
	}
	if input.ClientId != "" && clientID[0] != input.ClientId {
		return "", &common.OidcClientIdNotMatchingError{}
	}

	// Check if the user has authorized the client before
	var userAuthorizedOIDCClient model.UserAuthorizedOidcClient
	err = s.db.
		WithContext(ctx).
		Preload("Client").
		First(&userAuthorizedOIDCClient, "client_id = ? AND user_id = ?", clientID[0], userID).
		Error
	if err != nil {
		return "", &common.OidcMissingAuthorizationError{}
	}

	// If the client has no logout callback URLs, return an error
	if len(userAuthorizedOIDCClient.Client.LogoutCallbackURLs) == 0 {
		return "", &common.OidcNoCallbackURLError{}
	}

	callbackURL, err := s.getCallbackURL(userAuthorizedOIDCClient.Client.LogoutCallbackURLs, input.PostLogoutRedirectUri)
	if err != nil {
		return "", err
	}

	return callbackURL, nil

}

func (s *OidcService) createAuthorizationCode(ctx context.Context, clientID string, userID string, scope string, nonce string, codeChallenge string, codeChallengeMethod string, tx *gorm.DB) (string, error) {
	randomString, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", err
	}

	codeChallengeMethodSha256 := strings.ToUpper(codeChallengeMethod) == "S256"

	oidcAuthorizationCode := model.OidcAuthorizationCode{
		ExpiresAt:                 datatype.DateTime(time.Now().Add(15 * time.Minute)),
		Code:                      randomString,
		ClientID:                  clientID,
		UserID:                    userID,
		Scope:                     scope,
		Nonce:                     nonce,
		CodeChallenge:             &codeChallenge,
		CodeChallengeMethodSha256: &codeChallengeMethodSha256,
	}

	err = tx.
		WithContext(ctx).
		Create(&oidcAuthorizationCode).
		Error
	if err != nil {
		return "", err
	}

	return randomString, nil
}

func (s *OidcService) validateCodeVerifier(codeVerifier, codeChallenge string, codeChallengeMethodSha256 bool) bool {
	if codeVerifier == "" || codeChallenge == "" {
		return false
	}

	if !codeChallengeMethodSha256 {
		return codeVerifier == codeChallenge
	}

	// Compute SHA-256 hash of the codeVerifier
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeVerifierHash := h.Sum(nil)

	// Base64 URL encode the verifier hash
	encodedVerifierHash := base64.RawURLEncoding.EncodeToString(codeVerifierHash)

	return encodedVerifierHash == codeChallenge
}

func (s *OidcService) getCallbackURL(urls []string, inputCallbackURL string) (callbackURL string, err error) {
	if inputCallbackURL == "" {
		return urls[0], nil
	}

	for _, callbackPattern := range urls {
		regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(callbackPattern), `\*`, ".*") + "$"
		matched, err := regexp.MatchString(regexPattern, inputCallbackURL)
		if err != nil {
			return "", err
		}
		if matched {
			return inputCallbackURL, nil
		}
	}

	return "", &common.OidcInvalidCallbackURLError{}
}

func (s *OidcService) CreateDeviceAuthorization(ctx context.Context, input dto.OidcDeviceAuthorizationRequestDto) (*dto.OidcDeviceAuthorizationResponseDto, error) {
	client, err := s.verifyClientCredentialsInternal(ctx, s.db, dto.OidcCreateTokensDto{
		ClientID:     input.ClientID,
		ClientSecret: input.ClientSecret,
	})
	if err != nil {
		return nil, err
	}

	// Generate codes
	deviceCode, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return nil, err
	}
	userCode, err := utils.GenerateRandomAlphanumericString(8)
	if err != nil {
		return nil, err
	}

	// Create device authorization
	deviceAuth := &model.OidcDeviceCode{
		DeviceCode:   deviceCode,
		UserCode:     userCode,
		Scope:        input.Scope,
		ExpiresAt:    datatype.DateTime(time.Now().Add(15 * time.Minute)),
		IsAuthorized: false,
		ClientID:     client.ID,
	}

	if err := s.db.Create(deviceAuth).Error; err != nil {
		return nil, err
	}

	return &dto.OidcDeviceAuthorizationResponseDto{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         common.EnvConfig.AppURL + "/device",
		VerificationURIComplete: common.EnvConfig.AppURL + "/device?code=" + userCode,
		ExpiresIn:               900, // 15 minutes
		Interval:                5,
	}, nil
}

func (s *OidcService) VerifyDeviceCode(ctx context.Context, userCode string, userID string, ipAddress string, userAgent string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var deviceAuth model.OidcDeviceCode
	if err := tx.WithContext(ctx).Preload("Client.AllowedUserGroups").First(&deviceAuth, "user_code = ?", userCode).Error; err != nil {
		log.Printf("Error finding device code with user_code %s: %v", userCode, err)
		return err
	}

	if time.Now().After(deviceAuth.ExpiresAt.ToTime()) {
		return &common.OidcDeviceCodeExpiredError{}
	}

	// Check if the user group is allowed to authorize the client
	var user model.User
	if err := tx.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	if !s.IsUserGroupAllowedToAuthorize(user, deviceAuth.Client) {
		return &common.OidcAccessDeniedError{}
	}

	if err := tx.WithContext(ctx).Preload("Client").First(&deviceAuth, "user_code = ?", userCode).Error; err != nil {
		log.Printf("Error finding device code with user_code %s: %v", userCode, err)
		return err
	}

	if time.Now().After(deviceAuth.ExpiresAt.ToTime()) {
		return &common.OidcDeviceCodeExpiredError{}
	}

	deviceAuth.UserID = &userID
	deviceAuth.IsAuthorized = true

	if err := tx.WithContext(ctx).Save(&deviceAuth).Error; err != nil {
		log.Printf("Error saving device auth: %v", err)
		return err
	}

	// Verify the update was successful
	var verifiedAuth model.OidcDeviceCode
	if err := tx.WithContext(ctx).First(&verifiedAuth, "device_code = ?", deviceAuth.DeviceCode).Error; err != nil {
		log.Printf("Error verifying update: %v", err)
		return err
	}

	// Create user authorization if needed
	hasAuthorizedClient, err := s.hasAuthorizedClientInternal(ctx, deviceAuth.ClientID, userID, deviceAuth.Scope, tx)
	if err != nil {
		return err
	}

	if !hasAuthorizedClient {
		err := s.createAuthorizedClientInternal(ctx, userID, deviceAuth.ClientID, deviceAuth.Scope, tx)
		if err != nil {
			return err
		}

		s.auditLogService.Create(ctx, model.AuditLogEventNewDeviceCodeAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": deviceAuth.Client.Name}, tx)
	} else {
		s.auditLogService.Create(ctx, model.AuditLogEventDeviceCodeAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": deviceAuth.Client.Name}, tx)
	}

	return tx.Commit().Error
}

func (s *OidcService) GetDeviceCodeInfo(ctx context.Context, userCode string, userID string) (*dto.DeviceCodeInfoDto, error) {
	var deviceAuth model.OidcDeviceCode
	err := s.db.
		WithContext(ctx).
		Preload("Client").
		First(&deviceAuth, "user_code = ?", userCode).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &common.OidcInvalidDeviceCodeError{}
		}
		return nil, err
	}

	if time.Now().After(deviceAuth.ExpiresAt.ToTime()) {
		return nil, &common.OidcDeviceCodeExpiredError{}
	}

	// Check if the user has already authorized this client with this scope
	hasAuthorizedClient := false
	if userID != "" {
		var err error
		hasAuthorizedClient, err = s.HasAuthorizedClient(ctx, deviceAuth.ClientID, userID, deviceAuth.Scope)
		if err != nil {
			return nil, err
		}
	}

	return &dto.DeviceCodeInfoDto{
		Client: dto.OidcClientMetaDataDto{
			ID:      deviceAuth.Client.ID,
			Name:    deviceAuth.Client.Name,
			HasLogo: deviceAuth.Client.HasLogo,
		},
		Scope:                 deviceAuth.Scope,
		AuthorizationRequired: !hasAuthorizedClient,
	}, nil
}

func (s *OidcService) createRefreshToken(ctx context.Context, clientID string, userID string, scope string, tx *gorm.DB) (string, error) {
	refreshToken, err := utils.GenerateRandomAlphanumericString(40)
	if err != nil {
		return "", err
	}

	// Compute the hash of the refresh token to store in the DB
	// Refresh tokens are pretty long already, so a "simple" SHA-256 hash is enough
	refreshTokenHash := utils.CreateSha256Hash(refreshToken)

	m := model.OidcRefreshToken{
		ExpiresAt: datatype.DateTime(time.Now().Add(30 * 24 * time.Hour)), // 30 days
		Token:     refreshTokenHash,
		ClientID:  clientID,
		UserID:    userID,
		Scope:     scope,
	}

	err = tx.
		WithContext(ctx).
		Create(&m).
		Error
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

func (s *OidcService) createAuthorizedClientInternal(ctx context.Context, userID string, clientID string, scope string, tx *gorm.DB) error {
	userAuthorizedClient := model.UserAuthorizedOidcClient{
		UserID:   userID,
		ClientID: clientID,
		Scope:    scope,
	}

	err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "client_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"scope"}),
		}).
		Create(&userAuthorizedClient).
		Error

	return err
}

func (s *OidcService) verifyClientCredentialsInternal(ctx context.Context, tx *gorm.DB, input dto.OidcCreateTokensDto) (*model.OidcClient, error) {
	// First, ensure we have a valid client ID
	if input.ClientID == "" {
		return nil, &common.OidcMissingClientCredentialsError{}
	}

	// Load the OIDC client's configuration
	var client model.OidcClient
	err := tx.
		WithContext(ctx).
		First(&client, "id = ?", input.ClientID).
		Error
	if err != nil {
		return nil, err
	}

	// We have 3 options
	// If credentials are provided, we validate them; otherwise, we can continue without credentials for public clients only
	switch {
	// First, if we have a client secret, we validate it
	case input.ClientSecret != "":
		err = bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(input.ClientSecret))
		if err != nil {
			return nil, &common.OidcClientSecretInvalidError{}
		}
		return &client, nil

	// Next, check if we want to use client assertions from federated identities
	case input.ClientAssertionType == ClientAssertionTypeJWTBearer && input.ClientAssertion != "":
		err = s.verifyClientAssertionFromFederatedIdentities(&client, input)
		if err != nil {
			log.Printf("Invalid assertion for client '%s': %v", client.ID, err)
			return nil, &common.OidcClientAssertionInvalidError{}
		}
		return &client, nil

	// There's no credentials
	// This is allowed only if the client is public
	case client.IsPublic:
		return &client, nil

	// If we're here, we have no credentials AND the client is not public, so credentials are required
	default:
		return nil, &common.OidcMissingClientCredentialsError{}
	}
}

func (s *OidcService) jwkSetForURL(url string) (set jwk.Set, err error) {
	// Check if we have already registered the URL
	// This is a background operation and uses a background context; the HTTP client has a timeout anyways
	registerCtx := context.Background()
	if !s.jwkCache.IsRegistered(registerCtx, url) {
		// We need to register the URL
		err = s.jwkCache.Register(
			registerCtx,
			url,
			jwk.WithMaxInterval(24*time.Hour),
			jwk.WithMinInterval(15*time.Minute),
		)
		// In case of race conditions (two goroutines calling jwkCache.Register at the same time), it's possible we can get a conflict anyways, so we ignore that error
		if err != nil && !errors.Is(err, httprc.ErrResourceAlreadyExists()) {
			return nil, fmt.Errorf("failed to register JWK set: %w", err)
		}
	}

	jwks, err := s.jwkCache.CachedSet(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached JWK set: %w", err)
	}

	return jwks, nil
}

func (s *OidcService) verifyClientAssertionFromFederatedIdentities(client *model.OidcClient, input dto.OidcCreateTokensDto) error {
	// First, parse the assertion JWT, without validating it, to check the issuer
	assertion := []byte(input.ClientAssertion)
	insecureToken, err := jwt.ParseInsecure(assertion)
	if err != nil {
		return fmt.Errorf("failed to parse client assertion JWT: %w", err)
	}

	issuer, _ := insecureToken.Issuer()
	if issuer == "" {
		return errors.New("client assertion does not contain an issuer claim")
	}

	// Ensure that this client is federated with the one that issued the token
	ocfi, ok := client.Credentials.FederatedIdentityForIssuer(issuer)
	if !ok {
		return fmt.Errorf("client assertion is not from an allowed issuer: %s", issuer)
	}

	// Get the JWK set for the issuer
	jwksURL := ocfi.JWKS
	if jwksURL == "" {
		// Default URL is from the issuer
		if strings.HasSuffix(issuer, "/") {
			jwksURL = issuer + ".well-known/jwks.json"
		} else {
			jwksURL = issuer + "/.well-known/jwks.json"
		}
	}
	jwks, err := s.jwkSetForURL(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to get JWK set for issuer '%s': %w", issuer, err)
	}

	// Set default audience and subject if missing
	audience := ocfi.Audience
	if audience == "" {
		audience = TokenExchangeAudience
	}
	subject := ocfi.Subject
	if subject == "" {
		// Default to the client ID, per RFC 7523
		subject = client.ID
	}

	// Now re-parse the token with proper validation
	// (Note: we don't use jwt.WithIssuer() because that would be redundant)
	_, err = jwt.Parse(assertion,
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithKeySet(jwks, jws.WithInferAlgorithmFromKey(true), jws.WithUseDefault(true)),
		jwt.WithAudience(audience),
		jwt.WithSubject(subject),
	)
	if err != nil {
		return fmt.Errorf("client assertion is not valid: %w", err)
	}

	// If we're here, the assertion is valid
	return nil
}
