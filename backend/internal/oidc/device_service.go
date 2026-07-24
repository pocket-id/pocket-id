package oidc

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/rfc8628"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type deviceService struct {
	provider             fosite.OAuth2Provider
	store                *Store
	userCodeStrategy     rfc8628.UserCodeStrategy
	authorizationService *authorizationService
	claimsService        *ClaimsService
	auditLog             AuditLogger
	db                   *gorm.DB
}

func newDeviceService(
	provider fosite.OAuth2Provider,
	store *Store,
	userCodeStrategy rfc8628.UserCodeStrategy,
	authorizationService *authorizationService,
	claimsService *ClaimsService,
	auditLog AuditLogger,
	db *gorm.DB,
) *deviceService {
	return &deviceService{
		provider:             provider,
		store:                store,
		userCodeStrategy:     userCodeStrategy,
		authorizationService: authorizationService,
		claimsService:        claimsService,
		auditLog:             auditLog,
		db:                   db,
	}
}

func (s *deviceService) createDeviceAuthorization(ctx context.Context, req *http.Request) (*dto.OidcDeviceAuthorizationResponseDto, fosite.Requester, error) {
	request, err := s.provider.NewDeviceRequest(ctx, req)
	if err != nil {
		return nil, request, err
	}

	// Validate the requested scopes and resolve the resource indicator to an audience and the subset of requested scopes that may be granted
	client := request.GetClient().(Client)
	resource, err := request.GetResource()
	if err != nil {
		return nil, request, err
	}
	audience, grantedScopes, _, err := s.authorizationService.resolveGrant(ctx, client.GetID(), resource, request.GetRequestedScopes())
	if err != nil {
		if resource != "" && errors.Is(err, fosite.ErrAccessDenied) {
			return nil, request, fosite.ErrInvalidTarget.WithHintf("The requested resource '%s' is invalid, missing, unknown, or malformed.", resource)
		}
		return nil, request, err
	}
	grantResourceIndicator(request, audience, grantedScopes)

	session := NewEmptySession()
	response, err := s.provider.NewDeviceResponse(ctx, request, session)
	if err != nil {
		return nil, request, err
	}

	return &dto.OidcDeviceAuthorizationResponseDto{
		DeviceCode:              response.GetDeviceCode(),
		UserCode:                response.GetUserCode(),
		VerificationURI:         response.GetVerificationURI(),
		VerificationURIComplete: response.GetVerificationURIComplete(),
		ExpiresIn:               int(response.GetExpiresIn()),
		Interval:                response.GetInterval(),
	}, request, nil
}

func (s *deviceService) acceptDeviceCode(ctx context.Context, userCode, userID, authenticationMethod string, authenticationTime time.Time, reauthenticationToken string, meta requestMeta) error {
	request, userCodeSignature, err := s.deviceRequestFromUserCode(ctx, userCode)
	if err != nil {
		return err
	}

	// A user code may be approved only once. Rejecting an already-approved code prevents a second
	// logged-in user from rebinding a pending device authorization to themselves before the device
	// polls for its token.
	if request.GetUserCodeState() != fosite.UserCodeUnused {
		return &common.OidcInvalidDeviceCodeError{}
	}

	client := request.GetClient().(Client)
	var user model.User
	if err = s.db.WithContext(ctx).Preload("UserGroups").First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if !IsUserGroupAllowedToAuthorize(user, client.OidcClient) {
		return fosite.ErrAccessDenied.WithHint("You are not allowed to access this service.")
	}

	resource, err := request.GetResource()
	if err != nil {
		return err
	}
	audience, grantedScopes, consentKeys, err := s.authorizationService.resolveGrant(ctx, client.GetID(), resource, request.GetRequestedScopes())
	if err != nil {
		return err
	}
	grantResourceIndicator(request, audience, grantedScopes)

	return withTx(ctx, s.db, func(ctx context.Context) error {
		if client.RequiresReauthentication {
			if reauthenticationToken == "" || s.authorizationService == nil || s.authorizationService.reauth == nil {
				return &common.ReauthenticationRequiredError{}
			}

			reauthenticatedAt, err := s.authorizationService.reauth.ConsumeReauthenticationToken(ctx, dbFromContext(ctx, s.db), reauthenticationToken, userID)
			if err != nil {
				return err
			}
			authenticationTime = reauthenticatedAt
		}
		if authenticationTime.IsZero() {
			authenticationTime = time.Now().UTC()
		}

		session := NewAuthenticatedSession(userID, authenticationMethod, authenticationTime, request.GetRequestedAt())

		if err = s.claimsService.applyIDTokenClaims(ctx, session, request.GetGrantedScopes()); err != nil {
			return err
		}
		request.SetSession(session)

		hasAlreadyAuthorizedClient, err := s.authorizationService.consent(ctx, userID, client.GetID(), consentKeys)
		if err != nil {
			return err
		}

		event := model.AuditLogEventDeviceCodeAuthorization
		if !hasAlreadyAuthorizedClient {
			event = model.AuditLogEventNewDeviceCodeAuthorization
		}
		s.auditLog.Create(ctx, event, meta.IPAddress, meta.UserAgent, userID, model.AuditLogData{"clientName": client.Name}, dbFromContext(ctx, s.db))

		deviceCodeSignature, err := s.store.AcceptDeviceCodeSessionByUserCodeSignature(ctx, userCodeSignature, request)
		if err != nil {
			return err
		}

		if request.GetGrantedScopes().Has("openid") {
			if err := s.store.CreateOpenIDConnectSession(ctx, deviceCodeSignature, request); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *deviceService) getDeviceCodeInfo(ctx context.Context, userCode, userID string) (*dto.DeviceCodeInfoDto, error) {
	request, _, err := s.deviceRequestFromUserCode(ctx, userCode)
	if err != nil {
		return nil, err
	}

	client := request.GetClient().(Client)
	resource, err := request.GetResource()
	if err != nil {
		return nil, err
	}
	authorizationRequired := true
	if userID != "" {
		_, _, consentKeys, err := s.authorizationService.resolveGrant(ctx, client.GetID(), resource, request.GetRequestedScopes())
		if err != nil {
			return nil, err
		}
		hasAuthorizedClient, err := s.authorizationService.hasAuthorizedClient(ctx, client.GetID(), userID, consentKeys)
		if err != nil {
			return nil, err
		}
		// The device flow has no per-request prompt parameter, so consent depends only on prior authorization and the client's skip-consent setting
		authorizationRequired = consentRequired(hasAuthorizedClient, client.SkipConsent, nil)
	}

	scope := request.GetRequestedScopes()
	if scope == nil {
		scope = []string{}
	}

	// Resolve friendly names for the requested custom-API permissions so the device consent screen matches the browser flow
	scopeInfo, err := s.authorizationService.resolveScopeInfoForRequest(ctx, resource, scope)
	if err != nil {
		return nil, err
	}
	// Always serialize a possibly empty array rather than null
	if scopeInfo == nil {
		scopeInfo = []dto.ScopeInfoDto{}
	}

	return &dto.DeviceCodeInfoDto{
		Client: dto.OidcClientMetaDataDto{
			ID:                       client.ID,
			Name:                     client.Name,
			HasLogo:                  client.HasLogo(),
			HasDarkLogo:              client.HasDarkLogo(),
			LaunchURL:                client.LaunchURL,
			RequiresReauthentication: client.RequiresReauthentication,
			ClientType:               string(client.ClientType),
			ClientIdHost:             client.ClientIDHost(),
		},
		Scope:                    scope,
		ScopeInfo:                scopeInfo,
		AuthorizationRequired:    authorizationRequired,
		ReauthenticationRequired: client.RequiresReauthentication,
	}, nil
}

func (s *deviceService) deviceRequestFromUserCode(ctx context.Context, userCode string) (fosite.DeviceRequester, string, error) {
	userCodeSignature, err := s.userCodeStrategy.UserCodeSignature(ctx, userCode)
	if err != nil {
		return nil, "", err
	}

	request, err := s.store.GetDeviceCodeSessionByUserCodeSignature(ctx, userCodeSignature)
	if errors.Is(err, fosite.ErrNotFound) {
		return nil, "", &common.OidcInvalidDeviceCodeError{}
	}
	if err != nil {
		return nil, "", err
	}

	if err = s.userCodeStrategy.ValidateUserCode(ctx, request, userCode); err != nil {
		if errors.Is(err, fosite.ErrDeviceExpiredToken) {
			return nil, "", &common.OidcDeviceCodeExpiredError{}
		}
		return nil, "", err
	}

	return request, userCodeSignature, nil
}
