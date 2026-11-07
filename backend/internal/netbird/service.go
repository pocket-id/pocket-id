package netbird

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// UserIDHeader carries the NetBird user ID of the peer's owner
// The NetBird proxy sets it only after validating the tunnel peer and strips any client-sent value
const UserIDHeader = "X-NetBird-User-Id"

// ProxySecretHeader carries NETBIRD_PROXY_SECRET, which the NetBird proxy target is configured to add to every request
// #nosec G101 -- this is the name of the header that carries the secret, not a credential
const ProxySecretHeader = "X-NetBird-Proxy-Secret"

// AuthenticationMethod marks sessions started via NetBird, which may only sign in to apps
const AuthenticationMethod = "netbird"

// Service signs users in by the NetBird identity the NetBird reverse proxy attaches to requests from the overlay network
type Service struct {
	db       *gorm.DB
	sessions SessionCreator
	auditLog AuditLogger
}

// Identify returns the enabled Pocket ID user that owns the NetBird peer behind the request
func (s *Service) Identify(ctx context.Context, proxySecret, netbirdUserID string) (model.User, error) {
	// Only the NetBird proxy knows the secret, so requests reaching Pocket ID any other way are anonymous
	secret := common.EnvConfig.NetBirdProxySecret
	if secret == "" || netbirdUserID == "" || subtle.ConstantTimeCompare([]byte(proxySecret), []byte(secret)) != 1 {
		return model.User{}, errNoIdentity()
	}

	var user model.User
	err := s.db.WithContext(ctx).Where("id = ?", pocketIDUserID(netbirdUserID)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, errNoIdentity()
	} else if err != nil {
		return model.User{}, err
	}

	if user.Disabled {
		return model.User{}, errNoIdentity()
	}

	return user, nil
}

// Login starts a session for the user behind the request, provided it is still the user the browser confirmed
func (s *Service) Login(ctx context.Context, dbConfig *appconfig.AppConfigModel, proxySecret, netbirdUserID, confirmedUserID, ipAddress, userAgent, browserToken string) (model.User, model.LoginTokens, error) {
	user, err := s.Identify(ctx, proxySecret, netbirdUserID)
	if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	// A peer whose owner changed after the button was shown must not sign in as someone else
	if user.ID != confirmedUserID {
		return model.User{}, model.LoginTokens{}, errNoIdentity()
	}

	sessionToken, err := s.sessions.Create(ctx, user.ID, AuthenticationMethod, ipAddress, userAgent, dbConfig.SessionDuration.AsDurationMinutes())
	if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	signIn := s.auditLog.CreateSignIn(ctx, auditlogs.EventNetBirdSignIn, ipAddress, userAgent, user.ID, browserToken, s.db, dbConfig.EmailLoginNotificationMode)
	s.auditLog.SendSignInNotification(ctx, signIn)

	return user, model.LoginTokens{SessionToken: sessionToken, KnownBrowserToken: signIn.KnownBrowserToken}, nil
}

func errNoIdentity() *apperror.Error {
	return apperror.NotFound("NetBird identity")
}

// pocketIDUserID maps a NetBird user ID to the Pocket ID user ID it was created from
// With NetBird's embedded Dex the ID is base64 of a protobuf message holding the upstream subject (field 1) and the connector ID (field 2)
// Otherwise NetBird uses Pocket ID directly and both IDs are the same, and Pocket ID's UUIDs are never valid base64 because of their hyphens
func pocketIDUserID(netbirdUserID string) string {
	buf, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(netbirdUserID, "="))
	if err != nil {
		return netbirdUserID
	}

	var subject, connector string
	for len(buf) > 0 {
		field, wireType, n := protowire.ConsumeTag(buf)
		if n < 0 || wireType != protowire.BytesType {
			return netbirdUserID
		}
		buf = buf[n:]

		value, n := protowire.ConsumeBytes(buf)
		if n < 0 {
			return netbirdUserID
		}
		buf = buf[n:]

		switch field { //nolint:exhaustive // Other fields carry nothing Pocket ID needs
		case 1:
			subject = string(value)
		case 2:
			connector = string(value)
		}
	}

	if subject == "" || connector == "" {
		return netbirdUserID
	}
	return subject
}
