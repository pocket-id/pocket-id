package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

type services struct {
	appConfigService     *service.AppConfigService
	appImagesService     *service.AppImagesService
	emailService         *service.EmailService
	geoLiteService       *service.GeoLiteService
	auditLogService      *service.AuditLogService
	jwtService           *service.JwtService
	scimService          *service.ScimService
	userService          *service.UserService
	customClaimService   *service.CustomClaimService
	oidcService          *service.OidcService
	userGroupService     *service.UserGroupService
	ldapService          *service.LdapService
	versionService       *service.VersionService
	fileStorage          storage.FileStorage
	appLockService       *service.AppLockService
	oneTimeAccessService *service.OneTimeAccessService

	apiKeyModule     *apikey.Module
	oidcModule       *oidc.Module
	webauthnModule   *webauthn.Module
	userSignUpModule *usersignup.Module
	apiModule        *api.Module
}

// Initializes all services
func initServices(ctx context.Context, db *gorm.DB, instanceID string, httpClient *http.Client, imageExtensions map[string]string, fileStorage storage.FileStorage, scheduler *job.Scheduler) (svc *services, err error) {
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
	svc.jwtService, err = service.NewJwtService(ctx, db, instanceID, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}

	svc.customClaimService = service.NewCustomClaimService(db)
	svc.webauthnModule, err = webauthn.New(webauthn.Dependencies{
		DB:        db,
		AppURL:    common.EnvConfig.AppURL,
		Signer:    svc.jwtService,
		AuditLog:  svc.auditLogService,
		AppConfig: svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn module: %w", err)
	}

	svc.scimService = service.NewScimService(db, scheduler, httpClient)

	svc.apiModule = api.New(api.Dependencies{DB: db, Issuer: common.EnvConfig.AppURL})

	svc.oidcModule, err = oidc.New(ctx, oidc.Dependencies{
		DB:         db,
		HTTPClient: httpClient,
		Config: oidc.Config{
			BaseURL:      common.EnvConfig.AppURL,
			TokenBaseURL: common.EnvConfig.AppURL,
			Secret:       common.EnvConfig.EncryptionKey,
		},
		Signer:       svc.jwtService,
		CustomClaims: svc.customClaimService,
		Reauth:       svc.webauthnModule,
		AuditLog:     svc.auditLogService,
		APIAccess:    svc.apiModule,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC module: %w", err)
	}

	svc.oidcService, err = service.NewOidcService(db, svc.jwtService, svc.appConfigService, svc.oidcModule.Preview, svc.scimService, httpClient, fileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC service: %w", err)
	}

	svc.userGroupService = service.NewUserGroupService(db, svc.appConfigService, svc.scimService)
	svc.userService = service.NewUserService(db, svc.jwtService, svc.auditLogService, svc.emailService, svc.appConfigService, svc.customClaimService, svc.appImagesService, svc.scimService, fileStorage)
	svc.ldapService = service.NewLdapService(db, httpClient, svc.appConfigService, svc.userService, svc.userGroupService, fileStorage)

	svc.apiKeyModule, err = apikey.New(ctx, apikey.Dependencies{
		DB:           db,
		StaticApiKey: common.EnvConfig.StaticApiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API key module: %w", err)
	}

	svc.userSignUpModule = usersignup.New(usersignup.Dependencies{
		DB:          db,
		Signer:      svc.jwtService,
		AuditLog:    svc.auditLogService,
		AppConfig:   svc.appConfigService,
		UserCreator: svc.userService,
	})
	svc.oneTimeAccessService = service.NewOneTimeAccessService(db, svc.userService, svc.jwtService, svc.auditLogService, svc.emailService, svc.appConfigService)

	svc.versionService = service.NewVersionService(httpClient)

	return svc, nil
}
