package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/job"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

type services struct {
	appConfigService     *service.AppConfigService
	appImagesService     *service.AppImagesService
	emailService         *service.EmailService
	geoLiteService       *service.GeoLiteService
	auditLogService      *service.AuditLogService
	jwtService           *service.JwtService
	webauthnService      *service.WebAuthnService
	scimService          *service.ScimService
	userService          *service.UserService
	customClaimService   *service.CustomClaimService
	oidcService          *service.OidcService
	userGroupService     *service.UserGroupService
	ldapService          *service.LdapService
	apiKeyService        *service.ApiKeyService
	versionService       *service.VersionService
	fileStorage          storage.FileStorage
	appLockService       *service.AppLockService
	userSignUpService    *service.UserSignUpService
	oneTimeAccessService *service.OneTimeAccessService
}

// Initializes all services
func initServices(ctx context.Context, db *gorm.DB, httpClient *http.Client, imageExtensions map[string]string, fileStorage storage.FileStorage, scheduler *job.Scheduler) (svc *services, err error) {
	svc = &services{}

	svc.appConfigService, err = service.NewAppConfigService(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create app config service: %w", err)
	}

	svc.fileStorage = fileStorage
	svc.appImagesService = service.NewAppImagesService(imageExtensions, fileStorage)
	svc.appLockService = service.NewAppLockService(db)

	svc.emailService, err = service.NewEmailService(db, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create email service: %w", err)
	}

	svc.geoLiteService = service.NewGeoLiteService(httpClient)
	svc.auditLogService = service.NewAuditLogService(db, svc.appConfigService, svc.emailService, svc.geoLiteService)
	svc.jwtService, err = service.NewJwtService(ctx, db, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}

	svc.customClaimService = service.NewCustomClaimService(db)
	svc.webauthnService, err = service.NewWebAuthnService(db, svc.jwtService, svc.auditLogService, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn service: %w", err)
	}

	svc.scimService = service.NewScimService(db, scheduler, httpClient)

	svc.oidcService, err = service.NewOidcService(ctx, db, svc.jwtService, svc.appConfigService, svc.auditLogService, svc.customClaimService, svc.webauthnService, svc.scimService, httpClient, fileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC service: %w", err)
	}

	svc.userGroupService = service.NewUserGroupService(db, svc.appConfigService, svc.scimService)
	svc.userService = service.NewUserService(db, svc.jwtService, svc.auditLogService, svc.emailService, svc.appConfigService, svc.customClaimService, svc.appImagesService, svc.scimService, fileStorage)
	svc.ldapService = service.NewLdapService(db, httpClient, svc.appConfigService, svc.userService, svc.userGroupService, fileStorage)
	svc.apiKeyService = service.NewApiKeyService(db, svc.emailService)
	svc.userSignUpService = service.NewUserSignupService(db, svc.jwtService, svc.auditLogService, svc.appConfigService, svc.userService)
	svc.oneTimeAccessService = service.NewOneTimeAccessService(db, svc.userService, svc.jwtService, svc.auditLogService, svc.emailService, svc.appConfigService)

	svc.versionService = service.NewVersionService(httpClient)

	return svc, nil
}
