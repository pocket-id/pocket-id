package service

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jws"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	// PrivateKeyFile is the path in the data/keys folder where the key is stored
	// This is a JSON file containing a key encoded as JWK
	PrivateKeyFile = "jwt_private_key.json"

	// RsaKeySize is the size, in bits, of the RSA key to generate if none is found
	RsaKeySize = 2048

	// KeyUsageSigning is the usage for the private keys, for the "use" property
	KeyUsageSigning = "sig"

	// IsAdminClaim is a boolean claim used in access tokens for admin users
	// This may be omitted on non-admin tokens
	IsAdminClaim = "isAdmin"

	// AccessTokenJWTType is the media type for access tokens
	AccessTokenJWTType = "AT+JWT"

	// IDTokenJWTType is the media type for ID tokens
	IDTokenJWTType = "ID+JWT"

	// Acceptable clock skew for verifying tokens
	clockSkew = time.Minute
)

type JwtService struct {
	privateKey       jwk.Key
	keyId            string
	appConfigService *AppConfigService
	jwksEncoded      []byte
}

func NewJwtService(appConfigService *AppConfigService) *JwtService {
	service := &JwtService{}

	// Ensure keys are generated or loaded
	if err := service.init(appConfigService, common.EnvConfig.KeysPath); err != nil {
		log.Fatalf("Failed to initialize jwt service: %v", err)
	}

	return service
}

func (s *JwtService) init(appConfigService *AppConfigService, keysPath string) error {
	s.appConfigService = appConfigService

	// Ensure keys are generated or loaded
	return s.loadOrGenerateKey(keysPath)
}

// loadOrGenerateKey loads the private key from the given path or generates it if not existing.
func (s *JwtService) loadOrGenerateKey(keysPath string) error {
	var key jwk.Key

	// First, check if we have a JWK file
	// If we do, then we just load that
	jwkPath := filepath.Join(keysPath, PrivateKeyFile)
	ok, err := utils.FileExists(jwkPath)
	if err != nil {
		return fmt.Errorf("failed to check if private key file (JWK) exists at path '%s': %w", jwkPath, err)
	}
	if ok {
		key, err = s.loadKeyJWK(jwkPath)
		if err != nil {
			return fmt.Errorf("failed to load private key file (JWK) at path '%s': %w", jwkPath, err)
		}

		// Set the key, and we are done
		err = s.SetKey(key)
		if err != nil {
			return fmt.Errorf("failed to set private key: %w", err)
		}

		return nil
	}

	// If we are here, we need to generate a new key
	key, err = s.generateNewRSAKey()
	if err != nil {
		return fmt.Errorf("failed to generate new private key: %w", err)
	}

	// Set the key in the object, which also validates it
	err = s.SetKey(key)
	if err != nil {
		return fmt.Errorf("failed to set private key: %w", err)
	}

	// Save the key as JWK
	err = SaveKeyJWK(s.privateKey, jwkPath)
	if err != nil {
		return fmt.Errorf("failed to save private key file at path '%s': %w", jwkPath, err)
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

func (s *JwtService) GenerateAccessToken(user model.User) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(user.ID).
		Expiration(now.Add(s.appConfigService.DbConfig.SessionDuration.AsDurationMinutes())).
		IssuedAt(now).
		Issuer(common.EnvConfig.AppURL).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, common.EnvConfig.AppURL)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
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
		jwt.WithAudience(common.EnvConfig.AppURL),
		jwt.WithIssuer(common.EnvConfig.AppURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

func (s *JwtService) GenerateIDToken(userClaims map[string]any, clientID string, nonce string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Expiration(now.Add(1 * time.Hour)).
		IssuedAt(now).
		Issuer(common.EnvConfig.AppURL).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, clientID)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	for k, v := range userClaims {
		err = token.Set(k, v)
		if err != nil {
			return "", fmt.Errorf("failed to set claim '%s': %w", k, err)
		}
	}

	if nonce != "" {
		err = token.Set("nonce", nonce)
		if err != nil {
			return "", fmt.Errorf("failed to set claim 'nonce': %w", err)
		}
	}

	headers, err := CreateTokenTypeHeader(IDTokenJWTType)
	if err != nil {
		return "", fmt.Errorf("failed to set token type: %w", err)
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey, jws.WithProtectedHeaders(headers)))
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
		jwt.WithIssuer(common.EnvConfig.AppURL),
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

	err = VerifyTokenTypeHeader(tokenString, IDTokenJWTType)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token type: %w", err)
	}

	return token, nil
}

func (s *JwtService) GenerateOauthAccessToken(user model.User, clientID string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(user.ID).
		Expiration(now.Add(1 * time.Hour)).
		IssuedAt(now).
		Issuer(common.EnvConfig.AppURL).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	err = SetAudienceString(token, clientID)
	if err != nil {
		return "", fmt.Errorf("failed to set 'aud' claim in token: %w", err)
	}

	headers, err := CreateTokenTypeHeader(AccessTokenJWTType)
	if err != nil {
		return "", fmt.Errorf("failed to set token type: %w", err)
	}

	alg, _ := s.privateKey.Algorithm()
	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *JwtService) VerifyOauthAccessToken(tokenString string) (jwt.Token, error) {
	alg, _ := s.privateKey.Algorithm()
	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.privateKey),
		jwt.WithAcceptableSkew(clockSkew),
		jwt.WithIssuer(common.EnvConfig.AppURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	err = VerifyTokenTypeHeader(tokenString, AccessTokenJWTType)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token type: %w", err)
	}

	return token, nil
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

	EnsureAlgInKey(pubKey)

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

func (s *JwtService) loadKeyJWK(path string) (jwk.Key, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key data: %w", err)
	}

	key, err := jwk.ParseKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	return key, nil
}

// EnsureAlgInKey ensures that the key contains an "alg" parameter, set depending on the key type
func EnsureAlgInKey(key jwk.Key) {
	_, ok := key.Algorithm()
	if ok {
		// Algorithm is already set
		return
	}

	switch key.KeyType() {
	case jwa.RSA():
		// Default to RS256 for RSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.RS256())
	case jwa.EC():
		// Default to ES256 for ECDSA keys
		_ = key.Set(jwk.AlgorithmKey, jwa.ES256())
	case jwa.OKP():
		// Default to EdDSA for OKP keys
		_ = key.Set(jwk.AlgorithmKey, jwa.EdDSA())
	}
}

func (s *JwtService) generateNewRSAKey() (jwk.Key, error) {
	// We generate RSA keys only
	rawKey, err := rsa.GenerateKey(rand.Reader, RsaKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	// Import the raw key
	return importRawKey(rawKey)
}

func importRawKey(rawKey any) (jwk.Key, error) {
	key, err := jwk.Import(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to import generated private key: %w", err)
	}

	// Generate the key ID
	kid, err := generateRandomKeyID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	_ = key.Set(jwk.KeyIDKey, kid)

	// Set other required fields
	_ = key.Set(jwk.KeyUsageKey, KeyUsageSigning)
	EnsureAlgInKey(key)

	return key, err
}

// SaveKeyJWK saves a JWK to a file
func SaveKeyJWK(key jwk.Key, path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory '%s' for key file: %w", dir, err)
	}

	keyFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	// Write the JSON file to disk
	enc := json.NewEncoder(keyFile)
	enc.SetEscapeHTML(false)
	err = enc.Encode(key)
	if err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// generateRandomKeyID generates a random key ID.
func generateRandomKeyID() (string, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GetIsAdmin returns the value of the "isAdmin" claim in the token
func GetIsAdmin(token jwt.Token) (bool, error) {
	if !token.Has(IsAdminClaim) {
		return false, nil
	}
	var isAdmin bool
	err := token.Get(IsAdminClaim, &isAdmin)
	return isAdmin, err
}

// CreateTokenTypeHeader creates a new JWS header with the given token type
func CreateTokenTypeHeader(tokenType string) (jws.Headers, error) {
	headers := jws.NewHeaders()
	err := headers.Set(jws.TypeKey, tokenType)
	if err != nil {
		return nil, fmt.Errorf("failed to set token type: %w", err)
	}

	return headers, nil
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

// VerifyTokenTypeHeader verifies that the "typ" header in the token matches the expected type
func VerifyTokenTypeHeader(tokenBytes string, expectedTokenType string) error {
	// Parse the raw token string purely as a JWS message structure
	// We don't need to verify the signature at this stage, just inspect headers.
	msg, err := jws.Parse([]byte(tokenBytes))
	if err != nil {
		return fmt.Errorf("failed to parse token as JWS message: %w", err)
	}

	// Get the list of signatures attached to the message. Usually just one for JWT.
	signatures := msg.Signatures()
	if len(signatures) == 0 {
		return errors.New("JWS message contains no signatures")
	}

	protectedHeaders := signatures[0].ProtectedHeaders()
	if protectedHeaders == nil {
		return fmt.Errorf("JWS signature has no protected headers")
	}

	// Retrieve the 'typ' header value from the PROTECTED headers.
	var typHeaderValue string
	err = protectedHeaders.Get(jws.TypeKey, &typHeaderValue)
	if err != nil {
		return fmt.Errorf("token is missing required protected header '%s'", jws.TypeKey)
	}

	if !strings.EqualFold(typHeaderValue, expectedTokenType) {
		return fmt.Errorf("'%s' header mismatch: expected '%s', got '%s'", jws.TypeKey, expectedTokenType, typHeaderValue)
	}

	return nil
}
