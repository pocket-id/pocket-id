package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/italypaleale/francis/host/local"
	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/devicelogin"
	"github.com/pocket-id/pocket-id/backend/internal/email"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/environment"
	"github.com/pocket-id/pocket-id/backend/internal/geolite"
	"github.com/pocket-id/pocket-id/backend/internal/ldapsync"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
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
	geoLiteModule      *geolite.Module
	auditLogService    *service.AuditLogService
	jwtService         *service.JwtService
	userService        *service.UserService
	customClaimService *service.CustomClaimService
	oidcService        *service.OidcService
	userGroupService   *service.UserGroupService
	fileStorage        storage.FileStorage

	apiKeyModule            *apikey.Module
	auditLogsModule         *auditlogs.Module
	deviceLoginModule       *devicelogin.Module
	ldapSyncModule          *ldapsync.Module
	scimSyncModule          *scimsync.Module
	oidcModule              *oidc.Module
	webauthnModule          *webauthn.Module
	userSignUpModule        *usersignup.Module
	oneTimeAccessModule     *onetimeaccess.Module
	emailVerificationModule *emailverification.Module
	apiModule               *api.Module
	environmentModule       *environment.Module
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

	svc.geoLiteModule, err = geolite.New(ctx, geolite.Dependencies{
		HTTPClient:  httpClient,
		DBPath:      common.EnvConfig.GeoLiteDBPath,
		DownloadURL: common.EnvConfig.GeoLiteDBUrl,
		LicenseKey:  common.EnvConfig.MaxMindLicenseKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GeoLite module: %w", err)
	}

	svc.auditLogService = service.NewAuditLogService(db, svc.emailModule, svc.geoLiteModule, svc.appConfigService)
	svc.auditLogsModule, err = auditlogs.New(auditlogs.Dependencies{
		DB:            db,
		Actors:        actors,
		RetentionDays: common.EnvConfig.AuditLogRetentionDays,
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logs module: %w", err)
	}

	svc.jwtService, err = service.NewJwtService(ctx, db, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}

	svc.customClaimService = service.NewCustomClaimService(db)
	svc.webauthnModule, err = webauthn.New(webauthn.Dependencies{
		DB:        db,
		Actors:    actors,
		AppURL:    common.EnvConfig.AppURL,
		Signer:    svc.jwtService,
		AuditLog:  svc.auditLogService,
		AppConfig: svc.appConfigService,
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
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
		IPLocator: svc.geoLiteModule,
		AppConfig: svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create device login module: %w", err)
	}

	svc.scimSyncModule, err = scimsync.New(scimsync.Dependencies{
		DB:         db,
		Actors:     actors,
		HTTPClient: httpClient,
		// Disable in test environment
		ScheduleDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SCIM sync module: %w", err)
	}

	svc.apiModule = api.New(api.Dependencies{DB: db, Issuer: common.EnvConfig.AppURL})

	svc.oidcModule, err = oidc.New(ctx, oidc.Dependencies{
		DB:                  db,
		Actors:              actors,
		HTTPClient:          httpClient,
		GetCIMDURLAllowlist: svc.appConfigService.GetCIMDURLAllowlist,
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
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC module: %w", err)
	}

	backchannelLogoutService, err := backchannellogout.NewService(db, svc.jwtService, httpClient, actors)
	if err != nil {
		return nil, fmt.Errorf("failed to create back-channel logout service: %w", err)
	}

	svc.oidcService, err = service.NewOidcService(db, svc.jwtService, svc.oidcModule.Preview, svc.oidcModule, svc.scimSyncModule, backchannelLogoutService, httpClient, fileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC service: %w", err)
	}

	svc.userGroupService = service.NewUserGroupService(db, svc.scimSyncModule, backchannelLogoutService)
	svc.userService = service.NewUserService(db, svc.jwtService, svc.auditLogService, svc.customClaimService, svc.appImagesService, svc.scimSyncModule, backchannelLogoutService, fileStorage)

	svc.ldapSyncModule, err = ldapsync.New(ldapsync.Dependencies{
		DB:                db,
		Actors:            actors,
		HTTPClient:        httpClient,
		FileStorage:       fileStorage,
		Users:             svc.userService,
		Groups:            svc.userGroupService,
		AppConfig:         svc.appConfigService,
		ScimSync:          svc.scimSyncModule,
		BackchannelLogout: backchannelLogoutService,
		// Disable in test environment
		ScheduleDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LDAP sync module: %w", err)
	}

	svc.apiKeyModule, err = apikey.New(ctx, apikey.Dependencies{
		DB:              db,
		Actors:          actors,
		StaticApiKey:    common.EnvConfig.StaticApiKey,
		AppConfig:       svc.appConfigService,
		EmailSender:     svc.emailModule,
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
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
		ScimSync:    svc.scimSyncModule,
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

	svc.environmentModule = environment.New(environment.Dependencies{
		HTTPClient:                  httpClient,
		SQLiteOnNetworkedFilesystem: sqliteOnNetworkedFilesystem,
	})

	return svc, nil
}
