package oidc

import (
	"encoding/json"
	"time"

	"github.com/ory/fosite"
	fositeoauth2 "github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	fositejwt "github.com/ory/fosite/token/jwt"
)

var _ openid.Session = (*Session)(nil)
var _ fositeoauth2.JWTSessionContainer = (*Session)(nil)

type Session struct {
	Claims               *fositejwt.IDTokenClaims       `json:"id_token_claims"`
	Headers              *fositejwt.Headers             `json:"headers"`
	JWTClaims            *fositejwt.JWTClaims           `json:"jwt_claims,omitempty"`
	JWTHeader            *fositejwt.Headers             `json:"jwt_header,omitempty"`
	ExpiresAt            map[fosite.TokenType]time.Time `json:"expires_at,omitempty"`
	Subject              string                         `json:"subject"`
	AuthenticationMethod string                         `json:"authentication_method,omitempty"`
}

func NewEmptySession() *Session {
	return &Session{
		Claims: &fositejwt.IDTokenClaims{
			RequestedAt: time.Now().UTC(),
			Extra:       map[string]interface{}{},
		},
	}
}

func NewAuthenticatedSession(subject, authenticationMethod string, authenticationTime, requestedAt time.Time) *Session {
	now := time.Now().UTC()
	if authenticationTime.IsZero() {
		authenticationTime = now
	}
	if requestedAt.IsZero() {
		requestedAt = now
	}

	session := NewEmptySession()
	session.Subject = subject
	session.AuthenticationMethod = authenticationMethod
	session.Claims.Subject = subject
	session.Claims.AuthTime = authenticationTime.UTC()
	session.Claims.RequestedAt = requestedAt.UTC()

	return session
}

func (s *Session) SetExpiresAt(key fosite.TokenType, exp time.Time) {
	if s.ExpiresAt == nil {
		s.ExpiresAt = make(map[fosite.TokenType]time.Time)
	}
	s.ExpiresAt[key] = exp
}

func (s *Session) GetExpiresAt(key fosite.TokenType) time.Time {
	if s.ExpiresAt == nil {
		s.ExpiresAt = make(map[fosite.TokenType]time.Time)
	}
	return s.ExpiresAt[key]
}

func (s *Session) GetUsername() string {
	return s.GetSubject()
}

func (s *Session) GetExtraClaims() map[string]interface{} {
	if s == nil || s.Claims.Issuer == "" {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"iss": s.Claims.Issuer,
	}
}

func (s *Session) GetSubject() string {
	if s == nil {
		return ""
	}
	return s.Subject
}

func (s *Session) Clone() fosite.Session {
	if s == nil {
		return nil
	}

	var clone Session
	data, err := json.Marshal(s)
	if err != nil {
		return NewEmptySession()
	}
	if err = json.Unmarshal(data, &clone); err != nil {
		return NewEmptySession()
	}
	return &clone
}

func (s *Session) IDTokenClaims() *fositejwt.IDTokenClaims {
	if s.Claims == nil {
		s.Claims = &fositejwt.IDTokenClaims{}
	}
	if s.Claims.Extra == nil {
		s.Claims.Extra = map[string]interface{}{}
	}
	return s.Claims
}

func (s *Session) IDTokenHeaders() *fositejwt.Headers {
	if s.Headers == nil {
		s.Headers = &fositejwt.Headers{}
	}
	if s.Headers.Extra == nil {
		s.Headers.Extra = map[string]interface{}{}
	}
	return s.Headers
}

func (s *Session) GetJWTClaims() fositejwt.JWTClaimsContainer {
	if s.JWTClaims == nil {
		s.JWTClaims = &fositejwt.JWTClaims{}
	}
	if s.JWTClaims.Subject == "" {
		s.JWTClaims.Subject = s.GetSubject()
	}
	if s.JWTClaims.Extra == nil {
		s.JWTClaims.Extra = map[string]interface{}{}
	}
	return s.JWTClaims
}

func (s *Session) GetJWTHeader() *fositejwt.Headers {
	if s.JWTHeader == nil {
		s.JWTHeader = &fositejwt.Headers{}
	}
	if s.JWTHeader.Extra == nil {
		s.JWTHeader.Extra = map[string]interface{}{}
	}
	return s.JWTHeader
}
