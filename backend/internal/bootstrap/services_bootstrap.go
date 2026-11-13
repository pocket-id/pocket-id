package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/x/revoke"
	"github.com/italypaleale/francis/builtin/ratelimit"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/customclaims"
	"github.com/pocket-id/pocket-id/backend/internal/devicelogin"
	"github.com/pocket-id/pocket-id/backend/internal/email"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/environment"
	"github.com/pocket-id/pocket-id/backend/internal/feature"
	"github.com/pocket-id/pocket-id/backend/internal/geolite"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/ldapsync"
	"github.com/pocket-id/pocket-id/backend/internal/logopreset"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/outbound"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/signingkey"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/usergroups"
	"github.com/pocket-id/pocket-id/backend/internal/users"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

type services struct {
	appConfigService         *appconfig.AppConfigService
	appAssetsService         *service.AppAssetsService
	emailModule              *email.Module
	geoLiteModule            *geolite.Module
	ipLocator                iplocation.Resolver
	signingKey               *signingkey.Key
	customClaimsModule       *customclaims.Module
	claimMappingPolicyModule *claimmappingpolicy.Module
	fileStorage              storage.FileStorage

	apiKeyModule            *apikey.Module
	auditLogsModule         *auditlogs.Module
	deviceLoginModule       *devicelogin.Module
	ldapSyncModule          *ldapsync.Module
	scimSyncModule          *scimsync.Module
	oidcModule              *oidc.Module
	webauthnModule          *webauthn.Module
	userSignUpModule        *usersignup.Module
	oneTimeAccessModule     *onetimeaccess.Module
	sessionModule           *session.Module
	emailVerificationModule *emailverification.Module
	apiModule               *api.Module
	environmentModule       *environment.Module
	logoPresetModule        *logopreset.Module
	featureModule           *feature.Module
	usersModule             *users.Module
	userGroupsModule        *usergroups.Module
	mcpModule               *mcp.Module
	actors                  francishost.Host
}

// Initializes all services
func initServices(
	ctx context.Context,
	db *gorm.DB,
	instanceID string,
	actors francishost.Host,
	httpClient *http.Client,
	outboundClients *outbound.Clients,
	assetExtensions map[string]string,
	fileStorage storage.FileStorage,
	rateLimitServices map[string]*ratelimit.RateLimitService,
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
	svc.appAssetsService = service.NewAppAssetsService(assetExtensions, fileStorage)

	svc.emailModule, err = email.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create email module: %w", err)
	}

	// Select the location provider once so all consumers use the configured implementation
	svc.ipLocator, svc.geoLiteModule, err = initIPLocationResolver(ctx, &common.EnvConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP location resolver: %w", err)
	}

	svc.signingKey, err = signingkey.Load(ctx, db, &common.EnvConfig, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing key: %w", err)
	}

	svc.auditLogsModule, err = auditlogs.New(ctx, auditlogs.Dependencies{
		EnvConfig:     &common.EnvConfig,
		InstanceID:    instanceID,
		AppURL:        common.EnvConfig.AppURL,
		EmailSender:   svc.emailModule,
		IPLocator:     svc.ipLocator,
		AppConfig:     svc.appConfigService,
		DB:            db,
		Actors:        actors,
		RetentionDays: common.EnvConfig.AuditLogRetentionDays,
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logs module: %w", err)
	}

	backchannelLogoutService, err := backchannellogout.NewService(db, backchannellogout.NewTokenSigner(svc.signingKey, common.EnvConfig.AppURL), outboundClients.Client(outbound.PurposeBackchannelLogout), actors)
	if err != nil {
		return nil, fmt.Errorf("failed to create back-channel logout service: %w", err)
	}

	svc.sessionModule, err = session.New(session.Dependencies{
		Actors:      actors,
		IPLocator:   svc.ipLocator,
		AuditLog:    svc.auditLogsModule,
		EndNotifier: backchannelLogoutService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session module: %w", err)
	}

	svc.customClaimsModule = customclaims.New(db)

	// Registrations wait for the FIDO metadata download and the revocation checks of its signing certificates, so both get a timeout the shared client doesn't have
	// The timeout covers the whole 11 MB blob, and a download that times out isn't retried for an hour
	fidoMetadataClient := &http.Client{Transport: httpClient.Transport, Timeout: 30 * time.Second}
	// go-webauthn makes the revocation checks through a package-level client, and an unreachable revocation server counts as not revoked, so they may give up early
	revoke.HTTPClient = &http.Client{Transport: httpClient.Transport, Timeout: 10 * time.Second}

	// An empty URL keeps the FIDO metadata from ever being downloaded
	var fidoMetadataURL string
	if common.EnvConfig.FidoMetadataURL != common.URLDisabled {
		fidoMetadataURL = common.EnvConfig.FidoMetadataURL
	}

	svc.webauthnModule, err = webauthn.New(webauthn.Dependencies{
		DB:              db,
		Actors:          actors,
		AppURL:          common.EnvConfig.AppURL,
		HTTPClient:      fidoMetadataClient,
		FidoMetadataURL: fidoMetadataURL,
		Sessions:        svc.sessionModule,
		AuditLog:        svc.auditLogsModule,
		AppConfig:       svc.appConfigService,
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
		Sessions:  svc.sessionModule,
		Reauth:    svc.webauthnModule,
		AuditLog:  svc.auditLogsModule,
		IPLocator: svc.ipLocator,
		AppConfig: svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create device login module: %w", err)
	}

	svc.scimSyncModule, err = scimsync.New(scimsync.Dependencies{
		DB:         db,
		Actors:     actors,
		HTTPClient: outboundClients.Client(outbound.PurposeSCIM),
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
		FederatedJWKSClient: outboundClients.Client(outbound.PurposeFederatedJWKS),
		CIMDTransport:       outboundClients.Transport(outbound.PurposeClientMetadata),
		GetCIMDURLAllowlist: svc.appConfigService.GetCIMDURLAllowlist,
		LogoHTTPClient:      outboundClients.Client(outbound.PurposeClientLogo),
		FileStorage:         fileStorage,
		Config: oidc.Config{
			BaseURL:                   common.EnvConfig.AppURL,
			TokenBaseURL:              common.EnvConfig.AppURL,
			Secret:                    common.EnvConfig.EncryptionKey,
			AllowInsecureCallbackURLs: common.EnvConfig.AllowInsecureCallbackURLs,
		},
		Signer:            svc.signingKey,
		CustomClaims:      svc.customClaimsModule,
		Reauth:            svc.webauthnModule,
		AuditLog:          svc.auditLogsModule,
		Sessions:          svc.sessionModule,
		APIAccess:         svc.apiModule,
		AppConfig:         svc.appConfigService,
		ScimSync:          svc.scimSyncModule,
		BackchannelLogout: backchannelLogoutService,
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC module: %w", err)
	}

	svc.claimMappingPolicyModule = claimmappingpolicy.New(claimmappingpolicy.Dependencies{DB: db})
	svc.userGroupsModule = usergroups.New(usergroups.Dependencies{
		DB:                db,
		AppConfig:         svc.appConfigService,
		ScimSync:          svc.scimSyncModule,
		BackchannelLogout: backchannelLogoutService,
		AuditLog:          svc.auditLogsModule,
	})
	// The users module sends verification links through this module, so it's created first and never depends on the users module itself
	var sendLimiter emailverification.SendLimiter
	var ownAccountLimiter users.OwnAccountLimiter
	if middleware.RateLimitingEnabled() {
		sendLimiter = rateLimitServices[middleware.RateLimitSendEmailVerification]
		ownAccountLimiter = rateLimitServices[middleware.RateLimitUpdateOwnAccount]
	}
	svc.emailVerificationModule, err = emailverification.New(emailverification.Dependencies{
		DB:          db,
		Actors:      actors,
		EmailSender: svc.emailModule,
		ScimSync:    svc.scimSyncModule,
		SendLimiter: sendLimiter,
		AppConfig:   svc.appConfigService,
		AuditLog:    svc.auditLogsModule,
		AppURL:      common.EnvConfig.AppURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create email verification module: %w", err)
	}

	svc.usersModule = users.New(users.Dependencies{
		DB:                db,
		AppConfig:         svc.appConfigService,
		FileStorage:       fileStorage,
		CustomClaims:      svc.customClaimsModule,
		AppAssets:         svc.appAssetsService,
		WebAuthn:          svc.webauthnModule,
		ScimSync:          svc.scimSyncModule,
		BackchannelLogout: backchannelLogoutService,
		EmailVerification: svc.emailVerificationModule,
		OwnAccountLimiter: ownAccountLimiter,
		AuditLog:          svc.auditLogsModule,
	})

	svc.ldapSyncModule, err = ldapsync.New(ldapsync.Dependencies{
		DB:                db,
		Actors:            actors,
		HTTPClient:        outboundClients.Client(outbound.PurposeLDAPPicture),
		FileStorage:       fileStorage,
		Users:             svc.usersModule,
		Groups:            svc.userGroupsModule,
		AppConfig:         svc.appConfigService,
		ScimSync:          svc.scimSyncModule,
		BackchannelLogout: backchannelLogoutService,
		// Disable in test environment
		ScheduleDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LDAP sync module: %w", err)
	}

	svc.apiKeyModule, err = apikey.New(apikey.Dependencies{
		DB:              db,
		Actors:          actors,
		StaticApiKey:    common.EnvConfig.StaticApiKey,
		AppConfig:       svc.appConfigService,
		EmailSender:     svc.emailModule,
		AuditLog:        svc.auditLogsModule,
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API key module: %w", err)
	}

	svc.userSignUpModule, err = usersignup.New(usersignup.Dependencies{
		DB:                db,
		Actors:            actors,
		Sessions:          svc.sessionModule,
		AuditLog:          svc.auditLogsModule,
		UserCreator:       svc.usersModule,
		AppConfig:         svc.appConfigService,
		ScimSync:          svc.scimSyncModule,
		UserDeleter:       svc.usersModule,
		EmailVerification: svc.emailVerificationModule,
		// Disable in test environment
		CleanupDisabled: common.EnvConfig.AppEnv.IsTest(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user signup module: %w", err)
	}

	svc.oneTimeAccessModule, err = onetimeaccess.New(onetimeaccess.Dependencies{
		DB:           db,
		Actors:       actors,
		Sessions:     svc.sessionModule,
		AuditLog:     svc.auditLogsModule,
		UserProvider: svc.usersModule,
		EmailSender:  svc.emailModule,
		AppConfig:    svc.appConfigService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create one-time access module: %w", err)
	}

	svc.environmentModule = environment.New(environment.Dependencies{
		HTTPClient:                  httpClient,
		SQLiteOnNetworkedFilesystem: sqliteOnNetworkedFilesystem,
	})

	// An empty base URL keeps the icon library disabled
	var iconLibraryURL string
	if common.EnvConfig.IconLibraryEnabled() {
		iconLibraryURL = common.EnvConfig.IconLibraryURL
	}
	svc.logoPresetModule = logopreset.New(logopreset.Dependencies{
		HTTPClient: httpClient,
		BaseURL:    iconLibraryURL,
	})

	svc.featureModule = feature.New(feature.Dependencies{DB: db, AppConfig: svc.appConfigService})

	tools := &mcp.Tools{}
	registerTools(tools, svc)
	svc.mcpModule = mcp.New(mcp.Dependencies{Tools: tools, HTTPClient: httpClient})

	return svc, nil
}

func initIPLocationResolver(ctx context.Context, config *common.EnvConfigSchema) (iplocation.Resolver, *geolite.Module, error) {
	// Cloudflare locations need no database initialization or background refresh
	if config.CloudflareLocationHeaders {
		return iplocation.NewCloudflareResolver(), nil, nil
	}

	module, err := geolite.New(ctx, geolite.Dependencies{
		// The shared client is instrumented with OpenTelemetry, which would record the license key in the url.full span attribute
		HTTPClient:  &http.Client{},
		DBPath:      config.GeoLiteDBPath,
		DownloadURL: config.GeoLiteDBUrl,
		LicenseKey:  config.MaxMindLicenseKey,
	})
	if err != nil {
		return nil, nil, err
	}
	return module, module, nil
}
