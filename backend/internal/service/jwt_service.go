package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

	// AuthenticationMethodPhishingResistant identifies phishing-resistant authentication, such as passkeys
	AuthenticationMethodPhishingResistant = "phr"

	// AuthenticationMethodOneTimePassword identifies one-time password/code authentication
	AuthenticationMethodOneTimePassword = "otp"

	// AccessTokenJWTType identifies a JWT as an access token used by Pocket ID
	AccessTokenJWTType = "access-token"

	// AccessTokenJWTTypeIsolated identifies a JWT as an isolated access token used by Pocket ID
	AccessTokenJWTTypeIsolated = "isolated-token"

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

func NewJwtService(ctx context.Context, db *gorm.DB, appConfigService *AppConfigService) (*JwtService, error) {
	service := &JwtService{}

	err := service.init(ctx, db, appConfigService, &common.EnvConfig)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *JwtService) init(ctx context.Context, db *gorm.DB, appConfigService *AppConfigService, envConfig *common.EnvConfigSchema) (err error) {
	s.appConfigService = appConfigService
	s.envConfig = envConfig
	s.db = db

	// Ensure keys are generated or loaded
	return s.LoadOrGenerateKey(ctx)
}

func (s *JwtService) LoadOrGenerateKey(ctx context.Context) error {
	// Get the key provider
	keyProvider, err := jwkutils.GetKeyProvider(s.db, s.envConfig, s.appConfigService.GetDbConfig().InstanceID.Value)
	if err != nil {
		return fmt.Errorf("failed to get key provider: %w", err)
	}

	// Try loading a key
	key, err := keyProvider.LoadKey(ctx)
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
	err = keyProvider.SaveKey(ctx, s.privateKey)
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

func (s *JwtService) GenerateAccessToken(user model.User, authenticationMethod string) (string, error) {
	return s.GenerateAccessTokenForClient(user, authenticationMethod, "")
}

func (s *JwtService) GenerateAccessTokenForClient(user model.User, authenticationMethod, incognitoClientID string) (string, error) {
	tokenType := AccessTokenJWTType

	if incognitoClientID != "" {
		tokenType = AccessTokenJWTTypeIsolated
	}

	now := time.Now()
	builder := jwt.NewBuilder().
		Subject(user.ID).
		Expiration(now.Add(s.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes())).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		JwtID(uuid.New().String())

	if incognitoClientID != "" {
		builder.Claim("permitted_clients", incognitoClientID)
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, s.envConfig.AppURL)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	err = SetTokenType(token, tokenType)
	if err != nil {
		return "", fmt.Errorf("failed to set 'type' claim in token: %w", err)
	}

	err = SetIsAdmin(token, user.IsAdmin)
	if err != nil {
		return "", fmt.Errorf("failed to set 'isAdmin' claim in token: %w", err)
	}

	err = SetAuthenticationMethods(token, authenticationMethod)
	if err != nil {
		return "", fmt.Errorf("failed to set '%s' claim in token: %w", common.AuthenticationMethodsClaim, err)
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}
func (s *JwtService) VerifyAccessToken(tokenString string) (jwt.Token, error) {
	return s.VerifyAccessTokenWithIsolated(tokenString, false)
}
func (s *JwtService) VerifyAccessTokenWithIsolated(tokenString string, includeIsolated bool) (jwt.Token, error) {
	alg, _ := s.privateKey.Algorithm()
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithAudience(s.envConfig.AppURL),
		jwt.WithIssuer(s.envConfig.AppURL),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	var tokenType string
	_ = token.Get(TokenTypeClaim, &tokenType)

	if tokenType == AccessTokenJWTType || (includeIsolated && tokenType == AccessTokenJWTTypeIsolated) {
		return token, nil
	}

	return nil, fmt.Errorf("invalid token type: %s", tokenType)
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

// GetKeyID returns the key ID (kid) of the signing key, if one is set.
func (s *JwtService) GetKeyID() (string, bool) {
	if s.privateKey == nil {
		return "", false
	}
	return s.privateKey.KeyID()
}

// GetAuthenticationMethod returns the first authentication method in the "amr" claim in the token
func (s *JwtService) GetAuthenticationMethod(token jwt.Token) (string, error) {
	if !token.Has(common.AuthenticationMethodsClaim) {
		return "", nil
	}
	var rawAuthenticationMethods []any
	err := token.Get(common.AuthenticationMethodsClaim, &rawAuthenticationMethods)
	if err != nil {
		return "", fmt.Errorf("failed to get '%s' claim from token: %w", common.AuthenticationMethodsClaim, err)
	}

	if len(rawAuthenticationMethods) == 0 {
		return "", nil
	}
	authenticationMethod, ok := rawAuthenticationMethods[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid '%s' claim in token: expected array of strings", common.AuthenticationMethodsClaim)
	}
	return authenticationMethod, nil
}

// GetPermittedClients returns the value in the "permitted_clients" claim in the token
func (s *JwtService) GetPermittedClients(token jwt.Token) (string, error) {
	const permittedClientsClaim = "permitted_clients"

	if !token.Has(permittedClientsClaim) {
		return "", nil
	}

	var permittedClients string
	err := token.Get(permittedClientsClaim, &permittedClients)
	if err != nil {
		return "", fmt.Errorf("failed to get '%s' claim from token: %w", permittedClientsClaim, err)
	}
	return permittedClients, nil
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

// SetAuthenticationMethods sets the authentication method references claim in the token
func SetAuthenticationMethods(token jwt.Token, authenticationMethod string) error {
	if authenticationMethod == "" {
		return nil
	}
	return token.Set(common.AuthenticationMethodsClaim, []string{authenticationMethod})
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

func (s *JwtService) GetPrivateKey() any {
	var privateKey any
	_ = jwk.Export(s.privateKey, &privateKey)
	return privateKey
}
