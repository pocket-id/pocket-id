package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/italypaleale/francis/host/local"
	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/devicelogin"
	"github.com/pocket-id/pocket-id/backend/internal/email"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
	"gorm.io/gorm"
)

type services struct {
	appConfigService   *appconfig.AppConfigService
	appImagesService   *service.AppImagesService
	emailModule        *email.Module
	geoLiteService     *service.GeoLiteService
	auditLogService    *service.AuditLogService
	jwtService         *service.JwtService
	scimService        *service.ScimService
	userService        *service.UserService
	customClaimService *service.CustomClaimService
	oidcService        *service.OidcService
	userGroupService   *service.UserGroupService
	ldapService        *service.LdapService
	versionService     *service.VersionService
	fileStorage        storage.FileStorage

	apiKeyModule            *apikey.Module
	deviceLoginModule       *devicelogin.Module
	oidcModule              *oidc.Module
	webauthnModule          *webauthn.Module
	userSignUpModule        *usersignup.Module
	oneTimeAccessModule     *onetimeaccess.Module
	emailVerificationModule *emailverification.Module
	apiModule               *api.Module
	actors                  *local.Host
}

// Initializes all services
func initServices(
	ctx context.Context,
	db *gorm.DB,
	instanceID string,
	actors *local.Host,
	httpClient *http.Client,
	imageExtensions map[string]string,
	fileStorage storage.FileStorage,
	scheduler *job.Scheduler,
) (svc *services, err error) {
	svc = &services{
		actors: actors,
	}

	// Init the app config service
	svc.appConfigService, err = appconfig.NewService(ctx, actors, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create app config service: %w", err)
	}

	svc.fileStorage = fileStorage
	svc.appImagesService = service.NewAppImagesService(imageExtensions, fileStorage)

	svc.emailModule, err = email.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create email module: %w", err)
	}

	svc.geoLiteService = service.NewGeoLiteService(httpClient)
	svc.auditLogService = service.NewAuditLogService(db, svc.emailModule, svc.geoLiteService, svc.appConfigService)
	svc.jwtService, err = service.NewJwtService(ctx, db, instanceID)
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
	svc.deviceLoginModule, err = devicelogin.New(devicelogin.Dependencies{
		DB:        db,
		BaseURL:   common.EnvConfig.AppURL,
		Actors:    actors,
		Signer:    svc.jwtService,
		Reauth:    svc.webauthnModule,
		AuditLog:  svc.auditLogService,
		AppConfig: svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create device login module: %w", err)
	}

	svc.scimService = service.NewScimService(db, scheduler, httpClient)

	svc.apiModule = api.New(api.Dependencies{DB: db, Issuer: common.EnvConfig.AppURL})

	svc.oidcModule, err = oidc.New(ctx, oidc.Dependencies{
		DB:         db,
		HTTPClient: httpClient,
		Config: oidc.Config{
			BaseURL:                   common.EnvConfig.AppURL,
			TokenBaseURL:              common.EnvConfig.AppURL,
			Secret:                    common.EnvConfig.EncryptionKey,
			AllowInsecureCallbackURLs: common.EnvConfig.AllowInsecureCallbackURLs,
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

	svc.oidcService, err = service.NewOidcService(db, svc.jwtService, svc.oidcModule.Preview, svc.scimService, httpClient, fileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC service: %w", err)
	}

	svc.userGroupService = service.NewUserGroupService(db, svc.scimService)
	svc.userService = service.NewUserService(db, svc.jwtService, svc.auditLogService, svc.customClaimService, svc.appImagesService, svc.scimService, fileStorage)
	svc.ldapService = service.NewLdapService(db, httpClient, svc.userService, svc.userGroupService, fileStorage)

	svc.apiKeyModule, err = apikey.New(ctx, apikey.Dependencies{
		DB:           db,
		StaticApiKey: common.EnvConfig.StaticApiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API key module: %w", err)
	}

	svc.userSignUpModule, err = usersignup.New(usersignup.Dependencies{
		DB:          db,
		Actors:      actors,
		Signer:      svc.jwtService,
		AuditLog:    svc.auditLogService,
		UserCreator: svc.userService,
		AppConfig:   svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user signup module: %w", err)
	}

	svc.oneTimeAccessModule, err = onetimeaccess.New(onetimeaccess.Dependencies{
		DB:           db,
		Actors:       actors,
		Signer:       svc.jwtService,
		AuditLog:     svc.auditLogService,
		UserProvider: svc.userService,
		EmailSender:  svc.emailModule,
		AppConfig:    svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create one-time access module: %w", err)
	}

	svc.emailVerificationModule, err = emailverification.New(emailverification.Dependencies{
		DB:          db,
		Actors:      actors,
		Users:       svc.userService,
		EmailSender: svc.emailModule,
		AppConfig:   svc.appConfigService,
		AppURL:      common.EnvConfig.AppURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create email verification module: %w", err)
	}

	svc.versionService = service.NewVersionService(httpClient)

	return svc, nil
}
