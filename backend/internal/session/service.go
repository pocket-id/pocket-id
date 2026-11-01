package session

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// touchInterval throttles how often a session's last activity is written, so most requests only read the session
const touchInterval = 5 * time.Minute

type Service struct {
	actorService *actor.Service
	ipLocator    iplocation.Resolver
}

func newService(actorService *actor.Service, ipLocator iplocation.Resolver) *Service {
	return &Service{actorService: actorService, ipLocator: ipLocator}
}

// Create starts a session for the user and returns its token
// It invokes an actor, so it must not be called while a database transaction is open, which would deadlock on SQLite
func (s *Service) Create(ctx context.Context, userID, authenticationMethod, ipAddress, userAgent string, duration time.Duration) (string, error) {
	sessionID, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return "", err
	}
	secret, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", err
	}

	// The location is resolved once at sign-in so listing sessions never waits on a lookup
	country, city, err := s.ipLocator.GetLocationByIP(ctx, ipAddress)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get IP location", slog.String("ip", ipAddress), slog.Any("error", err))
	}

	now := time.Now()
	_, err = s.actorService.Invoke(ctx, ActorType, userID, MethodCreate, Session{
		ID:                   sessionID,
		TokenHash:            utils.CreateSha256Hash(secret),
		AuthenticationMethod: authenticationMethod,
		IPAddress:            ipAddress,
		UserAgent:            userAgent,
		Country:              country,
		City:                 city,
		CreatedAt:            now,
		LastActiveAt:         now,
		ExpiresAt:            now.Add(duration),
	})
	if err != nil {
		return "", fmt.Errorf("error creating session: %w", err)
	}

	// The token carries the user ID so it can be routed to the actor that holds the session
	return userID + "." + sessionID + "." + secret, nil
}

// Validate returns the active session the token belongs to, and the ID of its user
// It returns a not_signed_in error when the token is malformed, revoked or expired
func (s *Service) Validate(ctx context.Context, token string) (string, Session, error) {
	userID, sessionID, secret, ok := parseToken(token)
	if !ok {
		return "", Session{}, apperror.NotSignedIn()
	}

	envelope, err := s.actorService.Peek(ctx, ActorType, userID, methodValidate, validateRequest{
		SessionID: sessionID,
		TokenHash: utils.CreateSha256Hash(secret),
	})
	if err != nil {
		return "", Session{}, fmt.Errorf("error validating session: %w", err)
	}
	var res validateResponse
	err = envelope.Decode(&res)
	if err != nil {
		return "", Session{}, fmt.Errorf("error decoding session validation response: %w", err)
	}
	if !res.Valid {
		return "", Session{}, apperror.NotSignedIn()
	}

	// A failed write of the last activity must not fail the request it was recorded for
	if time.Since(res.Session.LastActiveAt) > touchInterval {
		_, err = s.actorService.Invoke(ctx, ActorType, userID, methodTouch, sessionRequest{SessionID: sessionID})
		if err != nil {
			slog.WarnContext(ctx, "Failed to record session activity", slog.Any("error", err))
		}
	}

	return userID, res.Session, nil
}

// List returns the active sessions of the user, most recently active first
func (s *Service) List(ctx context.Context, userID string) ([]Session, error) {
	envelope, err := s.actorService.Peek(ctx, ActorType, userID, methodList, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing sessions: %w", err)
	}
	var sessions []Session
	err = envelope.Decode(&sessions)
	if err != nil {
		return nil, fmt.Errorf("error decoding sessions: %w", err)
	}

	slices.SortFunc(sessions, func(a, b Session) int {
		return cmp.Compare(b.LastActiveAt.UnixNano(), a.LastActiveAt.UnixNano())
	})
	return sessions, nil
}

// Revoke signs the user out of one session
func (s *Service) Revoke(ctx context.Context, userID, sessionID string) error {
	envelope, err := s.actorService.Invoke(ctx, ActorType, userID, methodRevoke, sessionRequest{SessionID: sessionID})
	if err != nil {
		return fmt.Errorf("error revoking session: %w", err)
	}
	var res revokeResponse
	err = envelope.Decode(&res)
	if err != nil {
		return fmt.Errorf("error decoding session revocation response: %w", err)
	}
	if !res.Found {
		return apperror.NotFound("session")
	}
	return nil
}

// RevokeAll signs the user out of every session except the one with exceptSessionID, which may be empty
func (s *Service) RevokeAll(ctx context.Context, userID, exceptSessionID string) error {
	_, err := s.actorService.Invoke(ctx, ActorType, userID, methodRevokeAll, revokeAllRequest{ExceptSessionID: exceptSessionID})
	if err != nil {
		return fmt.Errorf("error revoking sessions: %w", err)
	}
	return nil
}

func parseToken(token string) (userID, sessionID, secret string, ok bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
