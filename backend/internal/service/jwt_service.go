package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

const (
	// KeyUsageSigning is the usage for the private keys, for the "use" property
	KeyUsageSigning = "sig"

	// IsAdminClaim is a boolean claim used in access tokens for admin users
	// This may be omitted on non-admin tokens
	IsAdminClaim = "isAdmin"

	// TokenTypeClaim is the claim used to identify the type of token
	TokenTypeClaim = "type"

	// RefreshTokenClaim is the claim used for the refresh token's value
	RefreshTokenClaim = "rt"

	// OAuthAccessTokenJWTType identifies a JWT as an OAuth access token
	OAuthAccessTokenJWTType = "oauth-access-token" //nolint:gosec

	// OAuthRefreshTokenJWTType identifies a JWT as an OAuth refresh token
	OAuthRefreshTokenJWTType = "refresh-token"

	// AccessTokenJWTType identifies a JWT as an access token used by Pocket ID
	AccessTokenJWTType = "access-token"

	// IDTokenJWTType identifies a JWT as an ID token used by Pocket ID
	IDTokenJWTType = "id-token"

	// Acceptable clock skew for verifying tokens
	clockSkew = time.Minute
)

type JwtService struct {
	db               *gorm.DB
	envConfig        *common.EnvConfigSchema
	privateKey       jwk.Key
	keyId            string
	appConfigService *AppConfigService
	jwksEncoded      []byte
}

func NewJwtService(db *gorm.DB, appConfigService *AppConfigService) (*JwtService, error) {
	service := &JwtService{
		db: db,
	}

	// Ensure keys are generated or loaded
	err := service.init(appConfigService, &common.EnvConfig)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *JwtService) init(appConfigService *AppConfigService, envConfig *common.EnvConfigSchema) (err error) {
	s.appConfigService = appConfigService
	s.envConfig = envConfig

	// Ensure keys are generated or loaded
	return s.loadOrGenerateKey()
}

func (s *JwtService) loadOrGenerateKey() error {
	// Get the key provider
	keyProvider, err := jwkutils.GetKeyProvider(s.db, s.envConfig, s.appConfigService.GetDbConfig().InstanceID.Value)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	// Try loading a key
	key, err := keyProvider.LoadKey()
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}

	// If we have a key, store it in the object and we're done
	if key != nil {
		err = s.SetKey(key)
		if err != nil {
			return fmt.Errorf("failed to set private key: %w", err)
		}
		return nil
	}

	// If we are here, we need to generate a new key
	err = s.generateKey()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save the newly-generated key
	err = keyProvider.SaveKey(s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	return nil
}

// generateKey generates a new key and stores it in the object
func (s *JwtService) generateKey() error {
	// Default is to generate RS256 (RSA-2048) keys
	key, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	if err != nil {
		return fmt.Errorf("failed to generate new private key: %w", err)
	}

	// Set the key in the object, which also validates it
	err = s.SetKey(key)
	if err != nil {
		return fmt.Errorf("failed to set private key: %w", err)
	}

	return nil
}

func ValidateKey(privateKey jwk.Key) error {
	// Validate the loaded key
	err := privateKey.Validate()
	if err != nil {
		return fmt.Errorf("key object is invalid: %w", err)
	}
	keyID, ok := privateKey.KeyID()
	if !ok || keyID == "" {
		return errors.New("key object does not contain a key ID")
	}
	usage, ok := privateKey.KeyUsage()
	if !ok || usage != KeyUsageSigning {
		return errors.New("key object is not valid for signing")
	}
	ok, err = jwk.IsPrivateKey(privateKey)
	if err != nil || !ok {
		return errors.New("key object is not a private key")
	}

	return nil
}

func (s *JwtService) SetKey(privateKey jwk.Key) error {
	// Validate the loaded key
	err := ValidateKey(privateKey)
	if err != nil {
		return fmt.Errorf("private key is not valid: %w", err)
	}

	// Set the private key and key id in the object
	s.privateKey = privateKey

	keyId, ok := privateKey.KeyID()
	if !ok {
		return errors.New("key object does not contain a key ID")
	}
	s.keyId = keyId

	// Create and encode a JWKS containing the public key
	publicKey, err := s.GetPublicJWK()
	if err != nil {
		return fmt.Errorf("failed to get public JWK: %w", err)
	}
	jwks := jwk.NewSet()
	err = jwks.AddKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to add public key to JWKS: %w", err)
	}
	s.jwksEncoded, err = json.Marshal(jwks)
	if err != nil {
		return fmt.Errorf("failed to encode JWKS to JSON: %w", err)
	}

	return nil
}

func (s *JwtService) SaveKey(privateKey jwk.Key) error {
	keyProvider, err := jwkutils.GetKeyProvider(s.db, s.envConfig, s.appConfigService.GetDbConfig().InstanceID.Value)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	return keyProvider.SaveKey(privateKey)
}

func (s *JwtService) GenerateAccessToken(user model.User) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(user.ID).
		Expiration(now.Add(s.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes())).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, s.envConfig.AppURL)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	err = SetTokenType(token, AccessTokenJWTType)
	if err != nil {
		return "", fmt.Errorf("failed to set 'type' claim in token: %w", err)
	}

	err = SetIsAdmin(token, user.IsAdmin)
	if err != nil {
		return "", fmt.Errorf("failed to set 'isAdmin' claim in token: %w", err)
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *JwtService) VerifyAccessToken(tokenString string) (jwt.Token, error) {
	alg, _ := s.privateKey.Algorithm()
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithAudience(s.envConfig.AppURL),
		jwt.WithIssuer(s.envConfig.AppURL),
		jwt.WithValidator(TokenTypeValidator(AccessTokenJWTType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

// BuildIDToken creates an ID token with all claims
func (s *JwtService) BuildIDToken(userClaims map[string]any, clientID string, nonce string) (jwt.Token, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Expiration(now.Add(1 * time.Hour)).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	err = SetTokenType(token, IDTokenJWTType)
	if err != nil {
		return nil, fmt.Errorf("failed to set 'type' claim in token: %w", err)
	}

	for k, v := range userClaims {
		err = token.Set(k, v)
		if err != nil {
			return nil, fmt.Errorf("failed to set claim '%s': %w", k, err)
		}
	}

	if nonce != "" {
		err = token.Set("nonce", nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to set claim 'nonce': %w", err)
		}
	}

	return token, nil
}

// GenerateIDToken creates and signs an ID token
func (s *JwtService) GenerateIDToken(userClaims map[string]any, clientID string, nonce string) (string, error) {
	token, err := s.BuildIDToken(userClaims, clientID, nonce)
	if err != nil {
		return "", err
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *JwtService) VerifyIdToken(tokenString string, acceptExpiredTokens bool) (jwt.Token, error) {
	alg, _ := s.privateKey.Algorithm()

	opts := make([]jwt.ParseOption, 0)

	// These options are always present
	opts = append(opts,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithIssuer(s.envConfig.AppURL),
		jwt.WithValidator(TokenTypeValidator(IDTokenJWTType)),
	)

	// By default, jwt.Parse includes 3 default validators for "nbf", "iat", and "exp"
	// In case we want to accept expired tokens (during logout), we need to set the validators explicitly without validating "exp"
	if acceptExpiredTokens {
		// This is equivalent to the default validators except it doesn't validate "exp"
		opts = append(opts,
			jwt.WithResetValidators(true),
			jwt.WithValidator(jwt.IsIssuedAtValid()),
			jwt.WithValidator(jwt.IsNbfValid()),
		)
	}

	token, err := jwt.ParseString(tokenString, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

// BuildOAuthAccessToken creates an OAuth access token with all claims
func (s *JwtService) BuildOAuthAccessToken(user model.User, clientID string) (jwt.Token, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(user.ID).
		Expiration(now.Add(1 * time.Hour)).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	err = SetTokenType(token, OAuthAccessTokenJWTType)
	if err != nil {
		return nil, fmt.Errorf("failed to set 'type' claim in token: %w", err)
	}

	return token, nil
}

// GenerateOAuthAccessToken creates and signs an OAuth access token
func (s *JwtService) GenerateOAuthAccessToken(user model.User, clientID string) (string, error) {
	token, err := s.BuildOAuthAccessToken(user, clientID)
	if err != nil {
		return "", err
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *JwtService) VerifyOAuthAccessToken(tokenString string) (jwt.Token, error) {
	alg, _ := s.privateKey.Algorithm()
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithIssuer(s.envConfig.AppURL),
		jwt.WithValidator(TokenTypeValidator(OAuthAccessTokenJWTType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

func (s *JwtService) GenerateOAuthRefreshToken(userID string, clientID string, refreshToken string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(userID).
		Expiration(now.Add(RefreshTokenDuration)).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = token.Set(RefreshTokenClaim, refreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to set 'rt' claim in token: %w", err)
	}

	err = SetAudienceString(token, clientID)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	err = SetTokenType(token, OAuthRefreshTokenJWTType)
	if err != nil {
		return "", fmt.Errorf("failed to set 'type' claim in token: %w", err)
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *JwtService) VerifyOAuthRefreshToken(tokenString string) (userID, clientID, rt string, err error) {
	alg, _ := s.privateKey.Algorithm()
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithIssuer(s.envConfig.AppURL),
		jwt.WithValidator(TokenTypeValidator(OAuthRefreshTokenJWTType)),
	)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse token: %w", err)
	}

	err = token.Get(RefreshTokenClaim, &rt)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get '%s' claim from token: %w", RefreshTokenClaim, err)
	}

	audiences, ok := token.Audience()
	if !ok || len(audiences) != 1 || audiences[0] == "" {
		return "", "", "", errors.New("failed to get 'aud' claim from token")
	}
	clientID = audiences[0]

	userID, ok = token.Subject()
	if !ok {
		return "", "", "", errors.New("failed to get 'sub' claim from token")
	}

	return userID, clientID, rt, nil
}

// GetTokenType returns the type of the JWT token issued by Pocket ID, but **does not validate it**.
func (s *JwtService) GetTokenType(tokenString string) (string, jwt.Token, error) {
	// Disable validation and verification to parse the token without checking it
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(false),
		jwt.WithVerify(false),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse token: %w", err)
	}

	var tokenType string
	err = token.Get(TokenTypeClaim, &tokenType)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get token type claim: %w", err)
	}

	return tokenType, token, nil
}

// GetPublicJWK returns the JSON Web Key (JWK) for the public key.
func (s *JwtService) GetPublicJWK() (jwk.Key, error) {
	if s.privateKey == nil {
		return nil, errors.New("key is not initialized")
	}

	pubKey, err := s.privateKey.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	jwkutils.EnsureAlgInKey(pubKey, "", "")

	return pubKey, nil
}

// GetPublicJWKSAsJSON returns the JSON Web Key Set (JWKS) for the public key, encoded as JSON.
// The value is cached since the key is static.
func (s *JwtService) GetPublicJWKSAsJSON() ([]byte, error) {
	if len(s.jwksEncoded) == 0 {
		return nil, errors.New("key is not initialized")
	}

	return s.jwksEncoded, nil
}

// GetKeyAlg returns the algorithm of the key
func (s *JwtService) GetKeyAlg() (jwa.KeyAlgorithm, error) {
	if len(s.jwksEncoded) == 0 {
		return nil, errors.New("key is not initialized")
	}

	alg, ok := s.privateKey.Algorithm()
	if !ok || alg == nil {
		return nil, errors.New("failed to retrieve algorithm for key")
	}

	return alg, nil
}

// GetIsAdmin returns the value of the "isAdmin" claim in the token
func GetIsAdmin(token jwt.Token) (bool, error) {
	if !token.Has(IsAdminClaim) {
		return false, nil
	}
	var isAdmin bool
	err := token.Get(IsAdminClaim, &isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to get 'isAdmin' claim from token: %w", err)
	}
	return isAdmin, nil
}

// SetTokenType sets the "type" claim in the token
func SetTokenType(token jwt.Token, tokenType string) error {
	if tokenType == "" {
		return nil
	}
	return token.Set(TokenTypeClaim, tokenType)
}

// SetIsAdmin sets the "isAdmin" claim in the token
func SetIsAdmin(token jwt.Token, isAdmin bool) error {
	// Only set if true
	if !isAdmin {
		return nil
	}
	return token.Set(IsAdminClaim, isAdmin)
}

// SetAudienceString sets the "aud" claim with a value that is a string, and not an array
// This is permitted by RFC 7519, and it's done here for backwards-compatibility
func SetAudienceString(token jwt.Token, audience string) error {
	return token.Set(jwt.AudienceKey, audience)
}

// TokenTypeValidator is a validator function that checks the "type" claim in the token
func TokenTypeValidator(expectedTokenType string) jwt.ValidatorFunc {
	return func(_ context.Context, t jwt.Token) error {
		var tokenType string
		err := t.Get(TokenTypeClaim, &tokenType)
		if err != nil {
			return fmt.Errorf("failed to get token type claim: %w", err)
		}
		if tokenType != expectedTokenType {
			return fmt.Errorf("invalid token type: expected %s, got %s", expectedTokenType, tokenType)
		}
		return nil
	}
}
