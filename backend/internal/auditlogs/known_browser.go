package auditlogs

import (
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
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
	signer SessionTokenService
	appURL string
}

// generate creates a notification-only browser marker using the shared session signer
func (s *browserTokens) generate(userID string, lifetime time.Duration) (string, error) {
	if s.signer == nil {
		return "", errors.New("session token signer is not initialized")
	}
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

	// The shared signer owns the private key and pinned signing algorithm
	return s.signer.SignSessionToken(token)
}

// verify validates recognition without granting authentication or reading browser state
func (s *browserTokens) verify(token, userID string) error {
	if s.signer == nil {
		return errors.New("session token signer is not initialized")
	}
	if userID == "" {
		return errors.New("user ID is required for a known-browser token")
	}

	_, err := s.signer.VerifySessionToken(token,
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
