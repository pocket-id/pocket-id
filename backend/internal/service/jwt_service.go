package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

const (
	// KeyUsageSigning is the usage for the private keys, for the "use" property
	KeyUsageSigning = "sig"

	// AuthenticationMethodPhishingResistant identifies phishing-resistant authentication, such as passkeys
	AuthenticationMethodPhishingResistant = "phr"

	// LogoutTokenJWTTyp is the JOSE "typ" header for OIDC back-channel logout tokens
	LogoutTokenJWTTyp = "logout+jwt" //nolint:gosec // this is a JOSE typ header, not a credential

	// BackchannelLogoutEvent is the member of the "events" claim that marks a JWT as a back-channel logout token
	BackchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"

	// logoutTokenDuration is the lifetime of a logout token, which only needs to survive the delivery to the client
	logoutTokenDuration = 2 * time.Minute
)

type JwtService struct {
	db        *gorm.DB
	envConfig *common.EnvConfigSchema
	// privateKey signs tokens that are consumed externally, such as ID tokens and access tokens for apps
	privateKey jwk.Key
	// sessionKey is the symmetric key that signs Pocket ID's own session tokens
	sessionKey  jwk.Key
	keyId       string
	instanceID  string
	jwksEncoded []byte
}

func NewJwtService(ctx context.Context, db *gorm.DB, instanceID string) (*JwtService, error) {
	service := &JwtService{}

	err := service.init(ctx, db, instanceID, &common.EnvConfig)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *JwtService) init(ctx context.Context, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema) (err error) {
	s.envConfig = envConfig
	s.db = db
	s.instanceID = instanceID

	// Ensure keys are generated or loaded
	return s.LoadOrGenerateKey(ctx)
}

// LoadOrGenerateKey loads the signing keys from the database, generating and persisting them if they don't exist yet
func (s *JwtService) LoadOrGenerateKey(ctx context.Context) error {
	// Load the key used for tokens that are consumed externally, such as ID tokens and access tokens for apps
	err := retryKeyStorage(ctx, s.loadOrGenerateSigningKey)
	if err != nil {
		return fmt.Errorf("error loading signing key: %w", err)
	}

	// Load the key used for Pocket ID's own sessions, which is symmetric
	err = retryKeyStorage(ctx, s.loadOrGenerateSessionKey)
	if err != nil {
		return fmt.Errorf("error loading session key: %w", err)
	}

	return nil
}

func retryKeyStorage(ctx context.Context, loadOrGenerate func(context.Context) error) error {
	for retries := 0; ; retries++ {
		// Run the full load and store sequence so a key written by another replica takes precedence
		err := loadOrGenerate(ctx)
		if !errors.Is(err, jwkutils.ErrRetryKeyStorage) {
			return err
		}

		// Return the last conflict after the configured number of retries has been exhausted
		if retries == 3 {
			return err
		}

		// Wait briefly before reloading so the competing database transaction has time to finish
		slog.WarnContext(ctx, "Failed to store key, retrying", slog.Int("retry", retries+1), slog.Any("error", err))
		err = utils.SleepWithContext(ctx, 200*time.Millisecond)
		if err != nil {
			return err
		}
	}
}

func (s *JwtService) loadOrGenerateSigningKey(ctx context.Context) error {
	// Get the key provider
	keyProvider, err := jwkutils.GetKeyProvider(s.db, s.envConfig, s.instanceID)
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

func (s *JwtService) loadOrGenerateSessionKey(ctx context.Context) error {
	// Get the key provider for the session key
	keyProvider, err := jwkutils.GetSessionKeyProvider(s.db, s.envConfig, s.instanceID)
	if err != nil {
		return fmt.Errorf("failed to get session key provider: %w", err)
	}

	// Try loading a key
	key, err := keyProvider.LoadKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to load session key: %w", err)
	}

	// If we have a key, store it in the object and we're done
	if key != nil {
		err = s.SetSessionKey(key)
		if err != nil {
			return fmt.Errorf("failed to set session key: %w", err)
		}
		return nil
	}

	// If we are here, we need to generate a new key
	key, err = jwkutils.GenerateSessionKey()
	if err != nil {
		return fmt.Errorf("failed to generate session key: %w", err)
	}

	// Set the key in the object, which also validates it
	err = s.SetSessionKey(key)
	if err != nil {
		return fmt.Errorf("failed to set session key: %w", err)
	}

	// Save the newly-generated key
	err = keyProvider.SaveKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to save session key: %w", err)
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

// ValidateSessionKey validates the symmetric key used to sign session tokens
func ValidateSessionKey(sessionKey jwk.Key) error {
	// Validate the loaded key
	err := sessionKey.Validate()
	if err != nil {
		return fmt.Errorf("key object is invalid: %w", err)
	}
	if sessionKey.KeyType() != jwa.OctetSeq() {
		return errors.New("key object is not a symmetric key")
	}
	keyID, ok := sessionKey.KeyID()
	if !ok || keyID == "" {
		return errors.New("key object does not contain a key ID")
	}
	usage, ok := sessionKey.KeyUsage()
	if !ok || usage != KeyUsageSigning {
		return errors.New("key object is not valid for signing")
	}

	// Session tokens are always signed with the same algorithm, so a key for anything else can't be used
	alg, ok := sessionKey.Algorithm()
	if !ok || alg == nil || alg.String() != jwkutils.SessionKeyAlg().String() {
		return fmt.Errorf("key object is not valid for the %s algorithm", jwkutils.SessionKeyAlg().String())
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

// SetSessionKey sets the symmetric key used to sign session tokens
// This key is never published in the JWKS, since it's a shared secret that only Pocket ID needs
func (s *JwtService) SetSessionKey(sessionKey jwk.Key) error {
	err := ValidateSessionKey(sessionKey)
	if err != nil {
		return fmt.Errorf("session key is not valid: %w", err)
	}

	s.sessionKey = sessionKey

	return nil
}

// GenerateLogoutToken creates a logout token for OIDC Back-Channel Logout 1.0
// The token identifies the user with a "sub" claim only, as Pocket ID does not track per-session "sid" values
func (s *JwtService) GenerateLogoutToken(userID string, clientID string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(userID).
		Audience([]string{clientID}).
		Expiration(now.Add(logoutTokenDuration)).
		IssuedAt(now).
		Issuer(s.envConfig.AppURL).
		JwtID(uuid.NewV4().String()).
		Claim("events", map[string]any{BackchannelLogoutEvent: struct{}{}}).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	// The spec requires the "typ" header so clients can tell logout tokens apart from ID tokens
	headers := jws.NewHeaders()
	err = headers.Set(jws.TypeKey, LogoutTokenJWTTyp)
	if err != nil {
		return "", fmt.Errorf("failed to set 'typ' header: %w", err)
	}

	alg, err := s.GetKeyAlg()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(alg, s.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

// SignSessionToken signs feature-owned claims with the private symmetric session key
func (s *JwtService) SignSessionToken(token jwt.Token) (string, error) {
	if s.sessionKey == nil {
		return "", errors.New("session key is not initialized")
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwkutils.SessionKeyAlg(), s.sessionKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return string(signed), nil
}

// VerifySessionToken pins signature verification while the caller supplies its feature's validation rules
func (s *JwtService) VerifySessionToken(tokenString string, options ...jwt.ValidateOption) (jwt.Token, error) {
	if s.sessionKey == nil {
		return nil, errors.New("session key is not initialized")
	}

	parseOptions := []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.WithKey(jwkutils.SessionKeyAlg(), s.sessionKey),
	}
	for _, option := range options {
		parseOptions = append(parseOptions, option)
	}
	token, err := jwt.ParseString(tokenString, parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
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

func (s *JwtService) GetPrivateKey() any {
	privateKey, _ := jwk.Export[any](s.privateKey)
	return privateKey
}
