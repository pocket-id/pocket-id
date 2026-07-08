package oidc

import (
	"context"
	"fmt"
	"net/url"
	"time"

	jwxjwt "github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/ory/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type ClientPreview struct {
	IDToken     map[string]any
	AccessToken map[string]any
	UserInfo    map[string]any
}

type ClientPreviewBuilder struct {
	claimsService *ClaimsService
	strategies    tokenStrategies
}

func newClientPreviewBuilder(claimsService *ClaimsService, strategies tokenStrategies) *ClientPreviewBuilder {
	return &ClientPreviewBuilder{
		claimsService: claimsService,
		strategies:    strategies,
	}
}

func (b *ClientPreviewBuilder) BuildClientPreview(ctx context.Context, client model.OidcClient, userID string, scopes []string, authenticationMethod string) (*ClientPreview, error) {
	scopeArgs, err := b.validatedScopes(ctx, client, scopes)
	if err != nil {
		return nil, err
	}

	userInfo, err := b.claimsService.GetUserClaims(ctx, userID, scopeArgs)
	if err != nil {
		return nil, err
	}

	request := b.newPreviewRequest(ctx, client, userID, scopeArgs, authenticationMethod)
	session := request.GetSession().(*Session)
	applyUserClaimsToIDToken(session, userID, userInfo)

	idToken, err := b.strategies.idToken.GenerateIDToken(ctx, b.strategies.config.GetIDTokenLifespan(ctx), request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview ID token: %w", err)
	}

	accessToken, _, err := b.strategies.accessToken.GenerateAccessToken(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview access token: %w", err)
	}

	idTokenPayload, err := claimsFromJWTString(idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode preview ID token: %w", err)
	}

	accessTokenPayload, err := claimsFromJWTString(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode preview access token: %w", err)
	}

	return &ClientPreview{
		IDToken:     idTokenPayload,
		AccessToken: accessTokenPayload,
		UserInfo:    userInfo,
	}, nil
}

func (b *ClientPreviewBuilder) validatedScopes(ctx context.Context, client model.OidcClient, scopes []string) (fosite.Arguments, error) {
	// Mirror the authorize endpoint's scope handling so the preview matches the tokens a real
	// authorization would produce, including dropping unknown scopes when configured.
	requested := fosite.Arguments(fosite.RemoveEmpty(scopes))
	validated, err := fosite.FilterRequestedScopes(b.strategies.config.GetScopeStrategy(ctx), Client{OidcClient: client}, requested, b.strategies.config.GetIgnoreUnknownScopes(ctx))
	if err != nil {
		return nil, err
	}

	scopeArgs := make(fosite.Arguments, 0, len(validated))
	for _, scope := range validated {
		if !scopeArgs.Has(scope) {
			scopeArgs = append(scopeArgs, scope)
		}
	}

	return scopeArgs, nil
}

func (b *ClientPreviewBuilder) newPreviewRequest(ctx context.Context, client model.OidcClient, userID string, scopes fosite.Arguments, authenticationMethod string) *fosite.Request {
	now := time.Now().UTC()
	session := NewAuthenticatedSession(userID, authenticationMethod, now, now)
	session.SetExpiresAt(fosite.AccessToken, now.Add(b.strategies.config.GetAccessTokenLifespan(ctx)))

	request := fosite.NewRequest()
	request.RequestedAt = now
	request.Client = Client{OidcClient: client}
	request.RequestedScope = scopes
	request.GrantedScope = scopes
	request.RequestedAudience = fosite.Arguments{client.ID}
	request.GrantedAudience = fosite.Arguments{client.ID}
	request.Form = url.Values{}
	request.Session = session

	return request
}

func claimsFromJWTString(tokenString string) (map[string]any, error) {
	token, err := jwxjwt.ParseString(
		tokenString,
		jwxjwt.WithValidate(false),
		jwxjwt.WithVerify(false),
	)
	if err != nil {
		return nil, err
	}

	return utils.GetClaimsFromToken(token)
}
