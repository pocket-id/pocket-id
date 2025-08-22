package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type services struct {
	appConfigService   *service.AppConfigService
	emailService       *service.EmailService
	geoLiteService     *service.GeoLiteService
	auditLogService    *service.AuditLogService
	jwtService         *service.JwtService
	webauthnService    *service.WebAuthnService
	userService        *service.UserService
	customClaimService *service.CustomClaimService
	oidcService        *service.OidcService
	userGroupService   *service.UserGroupService
	ldapService        *service.LdapService
	apiKeyService      *service.ApiKeyService
}

// Initializes all services
func initServices(ctx context.Context, db *gorm.DB, httpClient *http.Client) (svc *services, err error) {
	svc = &services{}

	svc.appConfigService, err = service.NewAppConfigService(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create app config service: %w", err)
	}

	svc.emailService, err = service.NewEmailService(db, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create email service: %w", err)
	}

	svc.geoLiteService = service.NewGeoLiteService(httpClient)
	svc.auditLogService = service.NewAuditLogService(db, svc.appConfigService, svc.emailService, svc.geoLiteService)
	svc.jwtService, err = service.NewJwtService(db, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}

	svc.customClaimService = service.NewCustomClaimService(db)
	svc.webauthnService, err = service.NewWebAuthnService(db, svc.jwtService, svc.auditLogService, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn service: %w", err)
	}

	svc.oidcService, err = service.NewOidcService(ctx, db, svc.jwtService, svc.appConfigService, svc.auditLogService, svc.customClaimService, svc.webauthnService)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC service: %w", err)
	}

	svc.userGroupService = service.NewUserGroupService(db, svc.appConfigService)
	svc.ldapService = service.NewLdapService(db, httpClient, svc.appConfigService, svc.userService, svc.userGroupService)
	svc.apiKeyService = service.NewApiKeyService(db, svc.emailService)

	return svc, nil
}
