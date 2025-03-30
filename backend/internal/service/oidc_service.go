package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type OidcService struct {
	db                 *gorm.DB
	jwtService         *JwtService
	appConfigService   *AppConfigService
	auditLogService    *AuditLogService
	customClaimService *CustomClaimService
}

func NewOidcService(db *gorm.DB, jwtService *JwtService, appConfigService *AppConfigService, auditLogService *AuditLogService, customClaimService *CustomClaimService) *OidcService {
	return &OidcService{
		db:                 db,
		jwtService:         jwtService,
		appConfigService:   appConfigService,
		auditLogService:    auditLogService,
		customClaimService: customClaimService,
	}
}

func (s *OidcService) Authorize(input dto.AuthorizeOidcClientRequestDto, userID, ipAddress, userAgent string) (string, string, error) {
	var client model.OidcClient
	if err := s.db.Preload("AllowedUserGroups").First(&client, "id = ?", input.ClientID).Error; err != nil {
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
	if err := s.db.Preload("UserGroups").First(&user, "id = ?", userID).Error; err != nil {
		return "", "", err
	}

	if !s.IsUserGroupAllowedToAuthorize(user, client) {
		return "", "", &common.OidcAccessDeniedError{}
	}

	// Check if the user has already authorized the client with the given scope
	hasAuthorizedClient, err := s.HasAuthorizedClient(input.ClientID, userID, input.Scope)
	if err != nil {
		return "", "", err
	}

	// If the user has not authorized the client, create a new authorization in the database
	if !hasAuthorizedClient {
		userAuthorizedClient := model.UserAuthorizedOidcClient{
			UserID:   userID,
			ClientID: input.ClientID,
			Scope:    input.Scope,
		}

		if err := s.db.Create(&userAuthorizedClient).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				// The client has already been authorized but with a different scope so we need to update the scope
				if err := s.db.Model(&userAuthorizedClient).Update("scope", input.Scope).Error; err != nil {
					return "", "", err
				}
			} else {
				return "", "", err
			}
		}
	}

	// Create the authorization code
	code, err := s.createAuthorizationCode(input.ClientID, userID, input.Scope, input.Nonce, input.CodeChallenge, input.CodeChallengeMethod)
	if err != nil {
		return "", "", err
	}

	// Log the authorization event
	if hasAuthorizedClient {
		s.auditLogService.Create(model.AuditLogEventClientAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": client.Name})
	} else {
		s.auditLogService.Create(model.AuditLogEventNewClientAuthorization, ipAddress, userAgent, userID, model.AuditLogData{"clientName": client.Name})

	}

	return code, callbackURL, nil
}

// HasAuthorizedClient checks if the user has already authorized the client with the given scope
func (s *OidcService) HasAuthorizedClient(clientID, userID, scope string) (bool, error) {
	var userAuthorizedOidcClient model.UserAuthorizedOidcClient
	if err := s.db.First(&userAuthorizedOidcClient, "client_id = ? AND user_id = ?", clientID, userID).Error; err != nil {
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

func (s *OidcService) CreateTokens(code, grantType, clientID, clientSecret, codeVerifier, refreshToken string) (idToken string, accessToken string, newRefreshToken string, exp int, err error) {
	switch grantType {
	case "authorization_code":
		return s.createTokenFromAuthorizationCode(code, clientID, clientSecret, codeVerifier)
	case "refresh_token":
		accessToken, newRefreshToken, exp, err = s.createTokenFromRefreshToken(refreshToken, clientID, clientSecret)
		return "", accessToken, newRefreshToken, exp, err
	default:
		return "", "", "", 0, &common.OidcGrantTypeNotSupportedError{}
	}
}

func (s *OidcService) createTokenFromAuthorizationCode(code, clientID, clientSecret, codeVerifier string) (idToken string, accessToken string, refreshToken string, exp int, err error) {
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return "", "", "", 0, err
	}

	// Verify the client secret if the client is not public
	if !client.IsPublic {
		if clientID == "" || clientSecret == "" {
			return "", "", "", 0, &common.OidcMissingClientCredentialsError{}
		}

		err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret))
		if err != nil {
			return "", "", "", 0, &common.OidcClientSecretInvalidError{}
		}
	}

	var authorizationCodeMetaData model.OidcAuthorizationCode
	err = s.db.Preload("User").First(&authorizationCodeMetaData, "code = ?", code).Error
	if err != nil {
		return "", "", "", 0, &common.OidcInvalidAuthorizationCodeError{}
	}

	// If the client is public or PKCE is enabled, the code verifier must match the code challenge
	if client.IsPublic || client.PkceEnabled {
		if !s.validateCodeVerifier(codeVerifier, *authorizationCodeMetaData.CodeChallenge, *authorizationCodeMetaData.CodeChallengeMethodSha256) {
			return "", "", "", 0, &common.OidcInvalidCodeVerifierError{}
		}
	}

	if authorizationCodeMetaData.ClientID != clientID && authorizationCodeMetaData.ExpiresAt.ToTime().Before(time.Now()) {
		return "", "", "", 0, &common.OidcInvalidAuthorizationCodeError{}
	}

	userClaims, err := s.GetUserClaimsForClient(authorizationCodeMetaData.UserID, clientID)
	if err != nil {
		return "", "", "", 0, err
	}

	idToken, err = s.jwtService.GenerateIDToken(userClaims, clientID, authorizationCodeMetaData.Nonce)
	if err != nil {
		return "", "", "", 0, err
	}

	// Generate a refresh token
	refreshToken, err = s.createRefreshToken(clientID, authorizationCodeMetaData.UserID, authorizationCodeMetaData.Scope)
	if err != nil {
		return "", "", "", 0, err
	}

	accessToken, err = s.jwtService.GenerateOauthAccessToken(authorizationCodeMetaData.User, clientID)
	if err != nil {
		return "", "", "", 0, err
	}

	s.db.Delete(&authorizationCodeMetaData)

	return idToken, accessToken, refreshToken, 3600, nil
}

func (s *OidcService) createTokenFromRefreshToken(refreshToken, clientID, clientSecret string) (accessToken string, newRefreshToken string, exp int, err error) {
	if refreshToken == "" {
		return "", "", 0, &common.OidcMissingRefreshTokenError{}
	}

	// Get the client to check if it's public
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return "", "", 0, err
	}

	// Verify the client secret if the client is not public
	if !client.IsPublic {
		if clientID == "" || clientSecret == "" {
			return "", "", 0, &common.OidcMissingClientCredentialsError{}
		}

		err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret))
		if err != nil {
			return "", "", 0, &common.OidcClientSecretInvalidError{}
		}
	}

	// Verify refresh token
	var storedRefreshToken model.OidcRefreshToken
	err = s.db.Preload("User").
		Where("token = ? AND expires_at > ?", utils.CreateSha256Hash(refreshToken), datatype.DateTime(time.Now())).
		First(&storedRefreshToken).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", 0, &common.OidcInvalidRefreshTokenError{}
		}
		return "", "", 0, err
	}

	// Verify that the refresh token belongs to the provided client
	if storedRefreshToken.ClientID != clientID {
		return "", "", 0, &common.OidcInvalidRefreshTokenError{}
	}

	// Generate a new access token
	accessToken, err = s.jwtService.GenerateOauthAccessToken(storedRefreshToken.User, clientID)
	if err != nil {
		return "", "", 0, err
	}

	// Generate a new refresh token and invalidate the old one
	newRefreshToken, err = s.createRefreshToken(clientID, storedRefreshToken.UserID, storedRefreshToken.Scope)
	if err != nil {
		return "", "", 0, err
	}

	// Delete the used refresh token
	s.db.Delete(&storedRefreshToken)

	return accessToken, newRefreshToken, 3600, nil
}

func (s *OidcService) IntrospectToken(clientID, clientSecret, tokenString string) (introspectDto dto.OidcIntrospectionResponseDto, err error) {
	if clientID == "" || clientSecret == "" {
		return introspectDto, &common.OidcMissingClientCredentialsError{}
	}

	// Get the client to check if we are authorized.
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return introspectDto, &common.OidcClientSecretInvalidError{}
	}

	// Verify the client secret. This endpoint may not be used by public clients.
	if client.IsPublic {
		return introspectDto, &common.OidcClientSecretInvalidError{}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret)); err != nil {
		return introspectDto, &common.OidcClientSecretInvalidError{}
	}

	token, err := s.jwtService.VerifyIdToken(tokenString, false)
	if err != nil {
		introspectDto.Active = false
		return introspectDto, nil
	}

	introspectDto.Active = true
	if token.Has("scope") {
		var asString string
		var asStrings []string
		if err := token.Get("scope", &asString); err == nil {
			introspectDto.Scope = asString
		} else if err := token.Get("scope", &asStrings); err == nil {
			introspectDto.Scope = strings.Join(asStrings, " ")
		}
	}
	if expiration, hasExpiration := token.Expiration(); hasExpiration {
		introspectDto.Expiration = expiration.Unix()
	}
	if issuedAt, hasIssuedAt := token.IssuedAt(); hasIssuedAt {
		introspectDto.IssuedAt = issuedAt.Unix()
	}
	if notBefore, hasNotBefore := token.NotBefore(); hasNotBefore {
		introspectDto.NotBefore = notBefore.Unix()
	}
	if subject, hasSubject := token.Subject(); hasSubject {
		introspectDto.Subject = subject
	}
	if audience, hasAudience := token.Audience(); hasAudience {
		introspectDto.Audience = audience
	}
	if issuer, hasIssuer := token.Issuer(); hasIssuer {
		introspectDto.Issuer = issuer
	}
	if identifier, hasIdentifier := token.JwtID(); hasIdentifier {
		introspectDto.Identifier = identifier
	}

	return introspectDto, nil
}

func (s *OidcService) GetClient(clientID string) (model.OidcClient, error) {
	var client model.OidcClient
	if err := s.db.Preload("CreatedBy").Preload("AllowedUserGroups").First(&client, "id = ?", clientID).Error; err != nil {
		return model.OidcClient{}, err
	}
	return client, nil
}

func (s *OidcService) ListClients(searchTerm string, sortedPaginationRequest utils.SortedPaginationRequest) ([]model.OidcClient, utils.PaginationResponse, error) {
	var clients []model.OidcClient

	query := s.db.Preload("CreatedBy").Model(&model.OidcClient{})
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

func (s *OidcService) CreateClient(input dto.OidcClientCreateDto, userID string) (model.OidcClient, error) {
	client := model.OidcClient{
		Name:               input.Name,
		CallbackURLs:       input.CallbackURLs,
		LogoutCallbackURLs: input.LogoutCallbackURLs,
		CreatedByID:        userID,
		IsPublic:           input.IsPublic,
		PkceEnabled:        input.IsPublic || input.PkceEnabled,
	}

	if err := s.db.Create(&client).Error; err != nil {
		return model.OidcClient{}, err
	}

	return client, nil
}

func (s *OidcService) UpdateClient(clientID string, input dto.OidcClientCreateDto) (model.OidcClient, error) {
	var client model.OidcClient
	if err := s.db.Preload("CreatedBy").First(&client, "id = ?", clientID).Error; err != nil {
		return model.OidcClient{}, err
	}

	client.Name = input.Name
	client.CallbackURLs = input.CallbackURLs
	client.LogoutCallbackURLs = input.LogoutCallbackURLs
	client.IsPublic = input.IsPublic
	client.PkceEnabled = input.IsPublic || input.PkceEnabled

	if err := s.db.Save(&client).Error; err != nil {
		return model.OidcClient{}, err
	}

	return client, nil
}

func (s *OidcService) DeleteClient(clientID string) error {
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return err
	}

	if err := s.db.Delete(&client).Error; err != nil {
		return err
	}

	return nil
}

func (s *OidcService) CreateClientSecret(clientID string) (string, error) {
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
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
	if err := s.db.Save(&client).Error; err != nil {
		return "", err
	}

	return clientSecret, nil
}

func (s *OidcService) GetClientLogo(clientID string) (string, string, error) {
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return "", "", err
	}

	if client.ImageType == nil {
		return "", "", errors.New("image not found")
	}

	imageType := *client.ImageType
	imagePath := fmt.Sprintf("%s/oidc-client-images/%s.%s", common.EnvConfig.UploadPath, client.ID, imageType)
	mimeType := utils.GetImageMimeType(imageType)

	return imagePath, mimeType, nil
}

func (s *OidcService) UpdateClientLogo(clientID string, file *multipart.FileHeader) error {
	fileType := utils.GetFileExtension(file.Filename)
	if mimeType := utils.GetImageMimeType(fileType); mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	imagePath := fmt.Sprintf("%s/oidc-client-images/%s.%s", common.EnvConfig.UploadPath, clientID, fileType)
	if err := utils.SaveFile(file, imagePath); err != nil {
		return err
	}

	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return err
	}

	if client.ImageType != nil && fileType != *client.ImageType {
		oldImagePath := fmt.Sprintf("%s/oidc-client-images/%s.%s", common.EnvConfig.UploadPath, client.ID, *client.ImageType)
		if err := os.Remove(oldImagePath); err != nil {
			return err
		}
	}

	client.ImageType = &fileType
	if err := s.db.Save(&client).Error; err != nil {
		return err
	}

	return nil
}

func (s *OidcService) DeleteClientLogo(clientID string) error {
	var client model.OidcClient
	if err := s.db.First(&client, "id = ?", clientID).Error; err != nil {
		return err
	}

	if client.ImageType == nil {
		return errors.New("image not found")
	}

	imagePath := fmt.Sprintf("%s/oidc-client-images/%s.%s", common.EnvConfig.UploadPath, client.ID, *client.ImageType)
	if err := os.Remove(imagePath); err != nil {
		return err
	}

	client.ImageType = nil
	if err := s.db.Save(&client).Error; err != nil {
		return err
	}

	return nil
}

func (s *OidcService) GetUserClaimsForClient(userID string, clientID string) (map[string]interface{}, error) {
	var authorizedOidcClient model.UserAuthorizedOidcClient
	if err := s.db.Preload("User.UserGroups").First(&authorizedOidcClient, "user_id = ? AND client_id = ?", userID, clientID).Error; err != nil {
		return nil, err
	}

	user := authorizedOidcClient.User
	scope := authorizedOidcClient.Scope

	claims := map[string]interface{}{
		"sub": user.ID,
	}

	if strings.Contains(scope, "email") {
		claims["email"] = user.Email
		claims["email_verified"] = s.appConfigService.DbConfig.EmailsVerified.IsTrue()
	}

	if strings.Contains(scope, "groups") {
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
		"picture":            fmt.Sprintf("%s/api/users/%s/profile-picture.png", common.EnvConfig.AppURL, user.ID),
	}

	if strings.Contains(scope, "profile") {
		// Add profile claims
		for k, v := range profileClaims {
			claims[k] = v
		}

		// Add custom claims
		customClaims, err := s.customClaimService.GetCustomClaimsForUserWithUserGroups(userID)
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
	if strings.Contains(scope, "email") {
		claims["email"] = user.Email
	}

	return claims, nil
}

func (s *OidcService) UpdateAllowedUserGroups(id string, input dto.OidcUpdateAllowedUserGroupsDto) (client model.OidcClient, err error) {
	client, err = s.GetClient(id)
	if err != nil {
		return model.OidcClient{}, err
	}

	// Fetch the user groups based on UserGroupIDs in input
	var groups []model.UserGroup
	if len(input.UserGroupIDs) > 0 {
		if err := s.db.Where("id IN (?)", input.UserGroupIDs).Find(&groups).Error; err != nil {
			return model.OidcClient{}, err
		}
	}

	// Replace the current user groups with the new set of user groups
	if err := s.db.Model(&client).Association("AllowedUserGroups").Replace(groups); err != nil {
		return model.OidcClient{}, err
	}

	// Save the updated client
	if err := s.db.Save(&client).Error; err != nil {
		return model.OidcClient{}, err
	}

	return client, nil
}

// ValidateEndSession returns the logout callback URL for the client if all the validations pass
func (s *OidcService) ValidateEndSession(input dto.OidcLogoutDto, userID string) (string, error) {
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
	if err := s.db.Preload("Client").First(&userAuthorizedOIDCClient, "client_id = ? AND user_id = ?", clientID[0], userID).Error; err != nil {
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

func (s *OidcService) createAuthorizationCode(clientID string, userID string, scope string, nonce string, codeChallenge string, codeChallengeMethod string) (string, error) {
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

	if err := s.db.Create(&oidcAuthorizationCode).Error; err != nil {
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
		regexPattern := strings.ReplaceAll(regexp.QuoteMeta(callbackPattern), `\*`, ".*") + "$"
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

func (s *OidcService) createRefreshToken(clientID string, userID string, scope string) (string, error) {
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

	if err := s.db.Create(&m).Error; err != nil {
		return "", err
	}

	return refreshToken, nil
}
