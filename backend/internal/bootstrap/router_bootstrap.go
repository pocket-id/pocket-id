package bootstrap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	sloggin "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/frontend"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/systemd"
)

// This is used to register additional controllers for tests
var registerTestControllers []func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services)

func initRouter(db *gorm.DB, svc *services) (utils.Service, error) {
	r, err := initEngine()
	if err != nil {
		return nil, err
	}
	registerRoutes(r, db, svc)

	serverConfig, err := initServer(r)
	if err != nil {
		return nil, err
	}

	runFn := func(ctx context.Context) error {
		return runServer(ctx, serverConfig)
	}

	return runFn, nil
}

type serverConfig struct {
	addr         string
	certProvider *tlsCertProvider
	listener     net.Listener
	server       *http.Server
	tlsConfig    *tls.Config
}

func initEngine() (*gin.Engine, error) {
	setGinMode()

	r := gin.New()
	initLogger(r)
	configureEngine(r)
	registerGlobalMiddleware(r)

	frontendRateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(100*time.Millisecond), 300)
	if err := frontend.RegisterFrontend(r, frontendRateLimitMiddleware); err != nil {
		if errors.Is(err, frontend.ErrFrontendNotIncluded) {
			slog.Warn("Frontend is not included in the build. Skipping frontend registration.")
			return r, nil
		}
		return nil, fmt.Errorf("failed to register frontend: %w", err)
	}

	return r, nil
}

func setGinMode() {
	// Set the appropriate Gin mode based on the environment
	switch common.EnvConfig.AppEnv {
	case common.AppEnvProduction:
		gin.SetMode(gin.ReleaseMode)
	case common.AppEnvDevelopment:
		gin.SetMode(gin.DebugMode)
	case common.AppEnvTest:
		gin.SetMode(gin.TestMode)
	}
}

func configureEngine(r *gin.Engine) {
	if !common.EnvConfig.TrustProxy {
		_ = r.SetTrustedProxies(nil)
	}

	if common.EnvConfig.TrustedPlatform != "" {
		r.TrustedPlatform = common.EnvConfig.TrustedPlatform
	}

	if common.EnvConfig.TracingEnabled {
		r.Use(otelgin.Middleware(common.Name))
	}
}

func registerGlobalMiddleware(r *gin.Engine) {
	r.Use(middleware.HeadMiddleware())
	r.Use(middleware.NewCacheControlMiddleware().Add())
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewCspMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())
}

func registerRoutes(r *gin.Engine, db *gorm.DB, svc *services) {
	authMiddleware := middleware.NewAuthMiddleware(svc.apiKeyService, svc.userService, svc.jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()
	apiRateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 100)

	apiGroup := r.Group("/api", apiRateLimitMiddleware)
	controller.NewApiKeyController(apiGroup, authMiddleware, svc.apiKeyService)
	controller.NewWebauthnController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.webauthnService, svc.appConfigService)
	controller.NewOidcController(apiGroup, authMiddleware, fileSizeLimitMiddleware, svc.oidcService, svc.jwtService)
	controller.NewUserController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.userService, svc.oneTimeAccessService, svc.webauthnService, svc.appConfigService)
	controller.NewAppConfigController(apiGroup, authMiddleware, svc.appConfigService, svc.emailService, svc.ldapService)
	controller.NewAppImagesController(apiGroup, authMiddleware, svc.appImagesService)
	controller.NewAuditLogController(apiGroup, svc.auditLogService, authMiddleware)
	controller.NewUserGroupController(apiGroup, authMiddleware, svc.userGroupService)
	controller.NewCustomClaimController(apiGroup, authMiddleware, svc.customClaimService)
	controller.NewVersionController(apiGroup, authMiddleware, svc.versionService)
	controller.NewScimController(apiGroup, authMiddleware, svc.scimService)
	controller.NewUserSignupController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.userSignUpService, svc.appConfigService)

	registerTestRoutes(apiGroup, db, svc)

	baseGroup := r.Group("/", apiRateLimitMiddleware)
	controller.NewWellKnownController(baseGroup, svc.jwtService)

	// These are not rate-limited.
	controller.NewHealthzController(r)
}

func registerTestRoutes(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services) {
	if common.EnvConfig.AppEnv.IsProduction() {
		return
	}

	for _, f := range registerTestControllers {
		f(apiGroup, db, svc)
	}
}

func initServer(r *gin.Engine) (*serverConfig, error) {
	protocols, tlsConfig, certProvider, err := initServerProtocols()
	if err != nil {
		return nil, err
	}

	network, addr := listenerNetworkAndAddr()
	listener, err := net.Listen(network, addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to create %s listener: %w", network, err)
	}

	if err := setUnixSocketMode(network, addr); err != nil {
		listener.Close()
		return nil, err
	}

	return &serverConfig{
		addr:         addr,
		certProvider: certProvider,
		listener:     listener,
		server:       newHTTPServer(r, protocols),
		tlsConfig:    tlsConfig,
	}, nil
}

func initServerProtocols() (*http.Protocols, *tls.Config, *tlsCertProvider, error) {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)

	if common.EnvConfig.TLSCertFile == "" || common.EnvConfig.TLSKeyFile == "" {
		protocols.SetUnencryptedHTTP2(true)
		return protocols, nil, nil, nil
	}

	protocols.SetHTTP2(true)
	certProvider, err := newCertProvider(common.EnvConfig.TLSCertFile, common.EnvConfig.TLSKeyFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: certProvider.GetCertificate,
		MinVersion:     tls.VersionTLS13,
		NextProtos:     []string{"h2"},
	}

	slog.Info("TLS enabled")
	return protocols, tlsConfig, certProvider, nil
}

func newHTTPServer(r *gin.Engine, protocols *http.Protocols) *http.Server {
	return &http.Server{
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 10 * time.Second,
		Protocols:         protocols,
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// HEAD requests don't get matched by Gin routes, so we convert them to GET
			// middleware.HeadMiddleware will convert them back to HEAD later
			if req.Method == http.MethodHead {
				req.Method = http.MethodGet
				ctx := context.WithValue(req.Context(), middleware.IsHeadRequestCtxKey{}, true)
				req = req.WithContext(ctx)
			}

			r.ServeHTTP(w, req)
		}), &http2.Server{}),
	}
}

func listenerNetworkAndAddr() (string, string) {
	if common.EnvConfig.UnixSocket == "" {
		return "tcp", net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port)
	}

	addr := common.EnvConfig.UnixSocket
	os.Remove(addr) // remove dangling the socket file to avoid file-exist error
	return "unix", addr
}

func setUnixSocketMode(network, addr string) error {
	if network != "unix" || common.EnvConfig.UnixSocketMode == "" {
		return nil
	}

	mode, err := strconv.ParseUint(common.EnvConfig.UnixSocketMode, 8, 32)
	if err != nil {
		return fmt.Errorf("failed to parse UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
	}

	if err := os.Chmod(addr, os.FileMode(mode)); err != nil {
		return fmt.Errorf("failed to set UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
	}

	return nil
}

func runServer(ctx context.Context, config *serverConfig) error {
	slog.Info("Server listening", slog.String("addr", config.addr), slog.Bool("tls", config.tlsConfig != nil))

	certWatcher, err := startCertWatcher(ctx, config.certProvider)
	if err != nil {
		return err
	}
	defer closeCertWatcher(certWatcher)

	startHTTPServer(config)
	notifySystemdReady()

	<-ctx.Done()
	return shutdownServer(ctx, config.server)
}

func startCertWatcher(ctx context.Context, certProvider *tlsCertProvider) (*fsnotify.Watcher, error) {
	if certProvider == nil {
		return nil, nil
	}

	certWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate watcher: %w", err)
	}

	if err := certWatcher.Add(common.EnvConfig.TLSCertFile); err != nil {
		certWatcher.Close()
		return nil, fmt.Errorf("failed to watch TLS certificate: %w", err)
	}
	if err := certWatcher.Add(common.EnvConfig.TLSKeyFile); err != nil {
		certWatcher.Close()
		return nil, fmt.Errorf("failed to watch TLS key: %w", err)
	}

	go certProvider.StartWatching(ctx, certWatcher)
	return certWatcher, nil
}

func closeCertWatcher(certWatcher *fsnotify.Watcher) {
	if certWatcher != nil {
		certWatcher.Close()
	}
}

func startHTTPServer(config *serverConfig) {
	go func() {
		defer config.listener.Close()

		listener := config.listener
		if config.tlsConfig != nil {
			listener = tls.NewListener(config.listener, config.tlsConfig)
		}
		srvErr := config.server.Serve(listener)

		if srvErr != http.ErrServerClosed {
			slog.Error("Error starting app server", "error", srvErr)
			os.Exit(1)
		}
	}()
}

func notifySystemdReady() {
	err := systemd.SdNotifyReady()
	if err != nil {
		// Log the error only
		slog.Warn("Unable to notify systemd that the service is ready", "error", err)
	}
}

func shutdownServer(ctx context.Context, srv *http.Server) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	shutdownErr := srv.Shutdown(shutdownCtx)
	shutdownCancel()
	if shutdownErr != nil {
		// Log the error only (could be context canceled)
		slog.Warn("App server shutdown error", "error", shutdownErr)
	}

	return nil
}

func initLogger(r *gin.Engine) {
	loggerSkipPathsPrefix := []string{
		"GET /api/application-images/logo",
		"GET /api/application-images/background",
		"GET /api/application-images/favicon",
		"GET /api/application-images/email",
		"GET /_app",
		"GET /fonts",
		"GET /healthz",
		"HEAD /healthz",
	}

	r.Use(sloggin.SetLogger(
		sloggin.WithLogger(func(_ *gin.Context, _ *slog.Logger) *slog.Logger {
			return slog.Default()
		}),
		sloggin.WithSkipper(func(c *gin.Context) bool {
			for _, prefix := range loggerSkipPathsPrefix {
				if strings.HasPrefix(c.Request.Method+" "+c.Request.URL.String(), prefix) {
					return true
				}
			}
			return false
		}),
	))
}

// tlsCertProvider holds certificates that can be dynamically reloaded
type tlsCertProvider struct {
	certMutex   sync.RWMutex
	cert        *tls.Certificate
	certFile    string
	keyFile     string
	forceReload atomic.Bool
}

// GetCertificate implements tls.GetCertificate interface for dynamic certificate loading
func (p *tlsCertProvider) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if p.forceReload.Load() {
		p.certMutex.Lock()
		p.forceReload.Store(false)
		p.certMutex.Unlock()
	}

	p.certMutex.RLock()
	defer p.certMutex.RUnlock()
	return p.cert, nil
}

// newCertProvider creates a new certificate provider with initial certificates loaded
func newCertProvider(certFile, keyFile string) (*tlsCertProvider, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tlsCertProvider{
		cert:     &cert,
		certFile: certFile,
		keyFile:  keyFile,
	}, nil
}

// reloadCertificate reloads the certificate from disk
func (p *tlsCertProvider) reloadCertificate() error {
	cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		return fmt.Errorf("failed to reload TLS certificate: %w", err)
	}

	p.certMutex.Lock()
	p.cert = &cert
	p.certMutex.Unlock()

	return nil
}

// StartWatching begins monitoring the certificate files for changes with debouncing
func (p *tlsCertProvider) StartWatching(ctx context.Context, watcher *fsnotify.Watcher) {
	debounceDuration := 1 * time.Second
	reloadTimer := time.NewTimer(debounceDuration)
	reloadTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only process write/rename events for certificate/key files
			if event.Has(fsnotify.Write | fsnotify.Rename) {
				// Reset the debounce timer whenever we get a relevant event
				reloadTimer.Stop()
				// Drain the channel if there's a pending value
				select {
				case <-reloadTimer.C:
				default:
				}
				reloadTimer.Reset(debounceDuration)
				slog.Debug("TLS file change detected, debouncing", slog.String("path", event.Name))
			}
		case <-reloadTimer.C:
			// Timer fired - no more events in 500ms, so reload
			slog.Info("Reloading TLS certificate")

			if err := p.reloadCertificate(); err != nil {
				slog.Error("Failed to reload TLS certificate", "error", err)
				continue
			}

			p.forceReload.Store(true)
			slog.Info("TLS certificate reloaded successfully")
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Certificate watcher error", "error", err)
		}
	}
}
