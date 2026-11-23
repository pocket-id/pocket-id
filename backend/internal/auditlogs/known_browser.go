package auditlogs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

const (
	KnownBrowserLifetime = 180 * 24 * time.Hour
	knownBrowserJWTType  = "known-browser"
)

type browserTokenService interface {
	generate(userID string, lifetime time.Duration) (string, error)
	verify(token, userID string) error
}

type browserTokens struct {
	db        *gorm.DB
	envConfig *common.EnvConfigSchema
	appURL    string
	// key is the symmetric key that signs known-browser tokens, which is never published since only Pocket ID verifies them
	key jwk.Key
}

// loadKey loads the stored key for the instance, generating and storing one when there is none yet
func (s *browserTokens) loadKey(ctx context.Context, instanceID string) error {
	provider, err := jwkutils.GetSessionKeyProvider(s.db, s.envConfig, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get known-browser key provider: %w", err)
	}

	key, err := jwkutils.LoadOrGenerateKey(ctx, provider, jwkutils.GenerateSessionKey)
	if err != nil {
		return fmt.Errorf("error loading known-browser key: %w", err)
	}
	err = validateBrowserKey(key)
	if err != nil {
		return fmt.Errorf("known-browser key is not valid: %w", err)
	}

	s.key = key
	return nil
}

func validateBrowserKey(key jwk.Key) error {
	err := key.Validate()
	if err != nil {
		return fmt.Errorf("key object is invalid: %w", err)
	}
	if key.KeyType() != jwa.OctetSeq() {
		return errors.New("key object is not a symmetric key")
	}
	keyID, ok := key.KeyID()
	if !ok || keyID == "" {
		return errors.New("key object does not contain a key ID")
	}
	usage, ok := key.KeyUsage()
	if !ok || usage != jwkutils.KeyUsageSigning {
		return errors.New("key object is not valid for signing")
	}

	// Known-browser tokens are always signed with the same algorithm, so a key for anything else can't be used
	alg, ok := key.Algorithm()
	if !ok || alg == nil || alg.String() != jwkutils.SessionKeyAlg().String() {
		return fmt.Errorf("key object is not valid for the %s algorithm", jwkutils.SessionKeyAlg().String())
	}
	return nil
}

// generate creates a notification-only browser marker
func (s *browserTokens) generate(userID string, lifetime time.Duration) (string, error) {
	if userID == "" || lifetime <= 0 {
		return "", errors.New("user ID and positive lifetime are required for a known-browser token")
	}

	// Bind recognition to one user and instance, with a purpose that access-token validation rejects
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(userID).
		Issuer(s.appURL).
		Audience([]string{s.appURL}).
		IssuedAt(now).
		Expiration(now.Add(lifetime)).
		Claim("type", knownBrowserJWTType).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build known-browser token: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwkutils.SessionKeyAlg(), s.key))
	if err != nil {
		return "", fmt.Errorf("failed to sign known-browser token: %w", err)
	}
	return string(signed), nil
}

// verify validates recognition without granting authentication or reading browser state
func (s *browserTokens) verify(token, userID string) error {
	if userID == "" {
		return errors.New("user ID is required for a known-browser token")
	}

	_, err := jwt.ParseString(token,
		jwt.WithKey(jwkutils.SessionKeyAlg(), s.key),
		jwt.WithValidate(true),
		jwt.WithIssuer(s.appURL),
		jwt.WithAudience(s.appURL),
		jwt.WithSubject(userID),
		jwt.WithRequiredClaim(jwt.IssuedAtKey),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
		jwt.WithClaimValue("type", knownBrowserJWTType),
	)
	if err != nil {
		return fmt.Errorf("failed to verify known-browser token: %w", err)
	}
	return nil
}

func (s *service) rememberBrowser(userID, token string) (bool, string, error) {
	// Only a valid token for the authenticated user can suppress the new-browser notification
	known := token != "" && s.browserTokens.verify(token, userID) == nil

	// Renew recognition after every successful sign-in without persisting browser state
	renewed, err := s.browserTokens.generate(userID, KnownBrowserLifetime)
	if err != nil {
		return known, token, err
	}
	return known, renewed, nil
}
