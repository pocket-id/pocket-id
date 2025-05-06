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
// The context should be used by services only for initialization, and not for running
func initServices(initCtx context.Context, db *gorm.DB, httpClient *http.Client) (svc *services, err error) {
	svc = &services{}

	svc.appConfigService = service.NewAppConfigService(initCtx, db)

	svc.emailService, err = service.NewEmailService(db, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("unable to create email service: %w", err)
	}

	svc.geoLiteService = service.NewGeoLiteService(httpClient)
	svc.auditLogService = service.NewAuditLogService(db, svc.appConfigService, svc.emailService, svc.geoLiteService)
	svc.jwtService = service.NewJwtService(svc.appConfigService)
	svc.userService = service.NewUserService(db, svc.jwtService, svc.auditLogService, svc.emailService, svc.appConfigService)
	svc.customClaimService = service.NewCustomClaimService(db)
	svc.oidcService = service.NewOidcService(db, svc.jwtService, svc.appConfigService, svc.auditLogService, svc.customClaimService)
	svc.userGroupService = service.NewUserGroupService(db, svc.appConfigService)
	svc.ldapService = service.NewLdapService(db, httpClient, svc.appConfigService, svc.userService, svc.userGroupService)
	svc.apiKeyService = service.NewApiKeyService(db, svc.emailService)
	svc.webauthnService = service.NewWebAuthnService(db, svc.jwtService, svc.auditLogService, svc.appConfigService)

	return svc, nil
}
