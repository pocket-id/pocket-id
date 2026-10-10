package oidc

import (
	fositeoauth2 "github.com/pocket-id/fosite/handler/oauth2"
	"github.com/pocket-id/fosite/handler/openid"
	"github.com/pocket-id/fosite/handler/rfc8628"
	"github.com/pocket-id/fosite/token/jwt"
)

// strategyProvider hands every Fosite handler the token strategy it needs
type strategyProvider struct {
	opaque      fositeoauth2.CoreStrategy
	accessToken fositeoauth2.AccessTokenStrategy
	device      *deviceStrategy
	idToken     openid.OpenIDConnectTokenStrategy

	jwt.Signer
}

var (
	_ fositeoauth2.AuthorizeCodeStrategyProvider = (*strategyProvider)(nil)
	_ fositeoauth2.AccessTokenStrategyProvider   = (*strategyProvider)(nil)
	_ fositeoauth2.RefreshTokenStrategyProvider  = (*strategyProvider)(nil)
	_ openid.OpenIDConnectTokenStrategyProvider  = (*strategyProvider)(nil)
	_ rfc8628.DeviceRateLimitStrategyProvider    = (*strategyProvider)(nil)
	_ rfc8628.DeviceCodeStrategyProvider         = (*strategyProvider)(nil)
	_ rfc8628.UserCodeStrategyProvider           = (*strategyProvider)(nil)
	_ jwt.Signer                                 = (*strategyProvider)(nil)
)

func (s *strategyProvider) AuthorizeCodeStrategy() fositeoauth2.AuthorizeCodeStrategy {
	return s.opaque
}

func (s *strategyProvider) AccessTokenStrategy() fositeoauth2.AccessTokenStrategy {
	return s.accessToken
}

func (s *strategyProvider) RefreshTokenStrategy() fositeoauth2.RefreshTokenStrategy {
	return s.opaque
}

func (s *strategyProvider) OpenIDConnectTokenStrategy() openid.OpenIDConnectTokenStrategy {
	return s.idToken
}

func (s *strategyProvider) DeviceRateLimitStrategy() rfc8628.DeviceRateLimitStrategy {
	return s.device
}

func (s *strategyProvider) DeviceCodeStrategy() rfc8628.DeviceCodeStrategy {
	return s.device
}

func (s *strategyProvider) UserCodeStrategy() rfc8628.UserCodeStrategy {
	return s.device
}
