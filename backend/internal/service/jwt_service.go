package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

const (
	privateKeyFile = "jwt_private_key.pem"
)

type JwtService struct {
	privateKey       *rsa.PrivateKey
	keyId            string
	appConfigService *AppConfigService
}

func NewJwtService(appConfigService *AppConfigService) *JwtService {
	service := &JwtService{
		appConfigService: appConfigService,
	}

	// Ensure keys are generated or loaded
	if err := service.loadOrGenerateKey(common.EnvConfig.KeysPath); err != nil {
		log.Fatalf("Failed to initialize jwt service: %v", err)
	}

	return service
}

type AccessTokenJWTClaims struct {
	jwt.RegisteredClaims
	IsAdmin bool `json:"isAdmin,omitempty"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// loadOrGenerateKey loads RSA keys from the given paths or generates them if they do not exist.
func (s *JwtService) loadOrGenerateKey(keysPath string) error {
	privateKeyPath := filepath.Join(keysPath, privateKeyFile)

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		if err := s.generateKey(keysPath); err != nil {
			return fmt.Errorf("can't generate key: %w", err)
		}
	}

	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("can't read jwt private key: %w", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("can't parse jwt private key: %w", err)
	}

	err = s.SetKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to set private key: %w", err)
	}

	return nil
}

func (s *JwtService) SetKey(privateKey *rsa.PrivateKey) (err error) {
	s.privateKey = privateKey

	s.keyId, err = s.generateKeyID()
	if err != nil {
		return fmt.Errorf("can't generate key ID: %w", err)
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
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return &s.privateKey.PublicKey, nil
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
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return &s.privateKey.PublicKey, nil
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
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return &s.privateKey.PublicKey, nil
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

// GetJWK returns the JSON Web Key (JWK) for the public key.
func (s *JwtService) GetJWK() (JWK, error) {
	if s.privateKey == nil {
		return JWK{}, errors.New("public key is not initialized")
	}

	jwk := JWK{
		Kid: s.keyId,
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(s.privateKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.privateKey.E)).Bytes()),
	}

	return jwk, nil
}

// GenerateKeyID generates a Key ID for the public key using the first 8 bytes of the SHA-256 hash of the public key.
func (s *JwtService) generateKeyID() (string, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(&s.privateKey.PublicKey)
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

// generateKey generates a new RSA key and saves it to the specified path.
func (s *JwtService) generateKey(keysPath string) error {
	if err := os.MkdirAll(keysPath, 0700); err != nil {
		return fmt.Errorf("failed to create directories for keys: %w", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	privateKeyPath := filepath.Join(keysPath, privateKeyFile)
	if err := s.savePEMKey(privateKeyPath, x509.MarshalPKCS1PrivateKey(privateKey), "RSA PRIVATE KEY"); err != nil {
		return err
	}

	return nil
}

// savePEMKey saves a PEM encoded key to a file.
func (s *JwtService) savePEMKey(path string, keyBytes []byte, keyType string) error {
	keyFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	})

	if _, err := keyFile.Write(keyPEM); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}
