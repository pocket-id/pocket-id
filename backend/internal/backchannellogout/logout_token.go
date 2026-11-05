package backchannellogout

import (
	"fmt"
	"time"
	"uuid"

	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"

	"github.com/pocket-id/pocket-id/backend/internal/signingkey"
)

const (
	// logoutTokenType is the JOSE "typ" header the spec requires, so clients can tell logout tokens apart from ID tokens
	logoutTokenType = "logout+jwt" //nolint:gosec // this is a JOSE typ header, not a credential

	// logoutEvent is the member of the "events" claim that marks a JWT as a back-channel logout token
	logoutEvent = "http://schemas.openid.net/event/backchannel-logout"

	// logoutTokenLifetime only needs to cover the delivery to the client
	logoutTokenLifetime = 2 * time.Minute
)

type keySigner struct {
	key    *signingkey.Key
	issuer string
}

// NewTokenSigner signs logout tokens with the instance key, which clients already trust for ID tokens
func NewTokenSigner(key *signingkey.Key, issuer string) TokenSigner {
	return keySigner{key: key, issuer: issuer}
}

// GenerateLogoutToken creates a logout token for OIDC Back-Channel Logout 1.0
// The token identifies the user with a "sub" claim only, as Pocket ID does not track per-session "sid" values
func (s keySigner) GenerateLogoutToken(userID string, clientID string) (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Subject(userID).
		Audience([]string{clientID}).
		Expiration(now.Add(logoutTokenLifetime)).
		IssuedAt(now).
		Issuer(s.issuer).
		JwtID(uuid.NewV4().String()).
		Claim("events", map[string]any{logoutEvent: struct{}{}}).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	headers := jws.NewHeaders()
	err = headers.Set(jws.TypeKey, logoutTokenType)
	if err != nil {
		return "", fmt.Errorf("failed to set 'typ' header: %w", err)
	}

	return s.key.Sign(token, headers)
}
