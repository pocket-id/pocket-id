package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	privateKeyFilePem = "jwt_private_key.pem"
	privateKeyFileJwk = "jwt_private_key.json"

	// Size, in bits, of the RSA key to generate if none is found
	rsaKeySize = 2048

	keyUsageSigning = "sig"
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

type AccessTokenJWTClaims struct {
	jwt.RegisteredClaims
	IsAdmin bool `json:"isAdmin,omitempty"`
}

// loadOrGenerateKey loads the private key from the given path or generates it if not existing.
func (s *JwtService) loadOrGenerateKey(keysPath string) error {
	var key jwk.Key

	// First, check if we have a JWK file
	// If we do, then we just load that
	jwkPath := filepath.Join(keysPath, privateKeyFileJwk)
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

	// If we are here, we have not loaded a JWK yet
	// First, check if we have a PEM-encoded key, which would be "legacy".
	// If so, we load that and convert it to JWK too.
	pemPath := filepath.Join(keysPath, privateKeyFilePem)
	ok, err = utils.FileExists(pemPath)
	if err != nil {
		return fmt.Errorf("failed to check if private key file (PEM) exists at path '%s': %w", pemPath, err)
	}
	if ok {
		// PEM file exists, load that
		key, err = s.loadKeyPEM(pemPath)
		if err != nil {
			return fmt.Errorf("failed to load private key file (PEM) at path '%s': %w", pemPath, err)
		}
	} else {
		// If we are here, we need to generate a new key
		key, err = s.generateNewRSAKey()
		if err != nil {
			return fmt.Errorf("failed to generate new private key: %w", err)
		}
	}

	// Set the key in the object, which also validates it
	err = s.SetKey(key)
	if err != nil {
		return fmt.Errorf("failed to set private key: %w", err)
	}

	// Save the key as JWK
	err = saveKeyJWK(s.privateKey, jwkPath)
	if err != nil {
		return fmt.Errorf("failed to save private key file at path '%s': %w", jwkPath, err)
	}

	return nil
}

func (s *JwtService) SetKey(privateKey jwk.Key) error {
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
	if !ok || usage != keyUsageSigning {
		return errors.New("key object is not valid for signing")
	}
	ok, err = jwk.IsPrivateKey(privateKey)
	if err != nil || !ok {
		return errors.New("key object is not a private key")
	}

	// Set the private key in the object
	s.privateKey = privateKey

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
	sessionDurationInMinutes, _ := strconv.Atoi(s.appConfigService.DbConfig.SessionDuration.Value)
	claim := AccessTokenJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(sessionDurationInMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Audience:  jwt.ClaimStrings{common.EnvConfig.AppURL},
		},
		IsAdmin: user.IsAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claim)
	token.Header["kid"] = s.keyId

	return token.SignedString(s.privateKey)
}

func (s *JwtService) VerifyAccessToken(tokenString string) (*AccessTokenJWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenJWTClaims{}, func(token *jwt.Token) (any, error) {
		return s.getPublicKeyRaw()
	})
	if err != nil || !token.Valid {
		return nil, errors.New("couldn't handle this token")
	}

	claims, isValid := token.Claims.(*AccessTokenJWTClaims)
	if !isValid {
		return nil, errors.New("can't parse claims")
	}

	if !slices.Contains(claims.Audience, common.EnvConfig.AppURL) {
		return nil, errors.New("audience doesn't match")
	}
	return claims, nil
}

func (s *JwtService) GenerateIDToken(userClaims map[string]interface{}, clientID string, nonce string) (string, error) {
	claims := jwt.MapClaims{
		"aud": clientID,
		"exp": jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
		"iss": common.EnvConfig.AppURL,
	}

	for k, v := range userClaims {
		claims[k] = v
	}

	if nonce != "" {
		claims["nonce"] = nonce
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyId

	return token.SignedString(s.privateKey)
}

func (s *JwtService) GenerateOauthAccessToken(user model.User, clientID string) (string, error) {
	claim := jwt.RegisteredClaims{
		Subject:   user.ID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Audience:  jwt.ClaimStrings{clientID},
		Issuer:    common.EnvConfig.AppURL,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claim)
	token.Header["kid"] = s.keyId

	return token.SignedString(s.privateKey)
}

func (s *JwtService) VerifyOauthAccessToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return s.getPublicKeyRaw()
	})
	if err != nil || !token.Valid {
		return nil, errors.New("couldn't handle this token")
	}

	claims, isValid := token.Claims.(*jwt.RegisteredClaims)
	if !isValid {
		return nil, errors.New("can't parse claims")
	}

	return claims, nil
}

func (s *JwtService) VerifyIdToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return s.getPublicKeyRaw()
	}, jwt.WithIssuer(common.EnvConfig.AppURL))

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return nil, errors.New("couldn't handle this token")
	}

	claims, isValid := token.Claims.(*jwt.RegisteredClaims)
	if !isValid {
		return nil, errors.New("can't parse claims")
	}

	return claims, nil
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

	ensureAlgInKey(pubKey)

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

func (s *JwtService) getPublicKeyRaw() (any, error) {
	pubKey, err := s.privateKey.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	var pubKeyRaw any
	err = jwk.Export(pubKey, &pubKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to export raw public key: %w", err)
	}
	return pubKeyRaw, nil
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

func (s *JwtService) loadKeyPEM(path string) (jwk.Key, error) {
	// Load the key from disk and parse it
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key data: %w", err)
	}

	key, err := jwk.ParseKey(data, jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	// Populate the key ID using the "legacy" algorithm
	keyId, err := generatePemKeyID(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	key.Set(jwk.KeyIDKey, keyId)

	// Populate other required fields
	_ = key.Set(jwk.KeyUsageKey, keyUsageSigning)
	ensureAlgInKey(key)

	return key, nil
}

// ensureAlgInKey ensures that the key contains an "alg" parameter, set depending on the key type
func ensureAlgInKey(key jwk.Key) {
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
	rawKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
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
	_ = key.Set(jwk.KeyUsageKey, keyUsageSigning)
	ensureAlgInKey(key)

	return key, err
}

// saveKeyJWK saves a JWK to a file
func saveKeyJWK(key jwk.Key, path string) error {
	keyFile, err := os.Create(path)
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
// It is used for newly-generated keys
func generateRandomKeyID() (string, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// generatePemKeyID generates a Key ID for the public key using the first 8 bytes of the SHA-256 hash of the public key's PKIX-serialized structure.
// This is used for legacy keys, imported from PEM.
func generatePemKeyID(key jwk.Key) (string, error) {
	// Export the public key and serialize it to PKIX (not in a PEM block)
	// This is for backwards-compatibility with the algorithm used before the switch to JWK
	pubKey, err := key.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}
	var pubKeyRaw any
	err = jwk.Export(pubKey, &pubKeyRaw)
	if err != nil {
		return "", fmt.Errorf("failed to export public key: %w", err)
	}
	pubASN1, err := x509.MarshalPKIXPublicKey(pubKeyRaw)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Compute SHA-256 hash of the public key
	hash := sha256.New()
	hash.Write(pubASN1)
	hashed := hash.Sum(nil)

	// Truncate the hash to the first 8 bytes for a shorter Key ID
	shortHash := hashed[:8]

	// Return Base64 encoded truncated hash as Key ID
	return base64.RawURLEncoding.EncodeToString(shortHash), nil
}
