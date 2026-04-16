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
	// Set the appropriate Gin mode based on the environment
	switch common.EnvConfig.AppEnv {
	case common.AppEnvProduction:
		gin.SetMode(gin.ReleaseMode)
	case common.AppEnvDevelopment:
		gin.SetMode(gin.DebugMode)
	case common.AppEnvTest:
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	initLogger(r)

	if !common.EnvConfig.TrustProxy {
		_ = r.SetTrustedProxies(nil)
	}

	if common.EnvConfig.TrustedPlatform != "" {
		r.TrustedPlatform = common.EnvConfig.TrustedPlatform
	}

	if common.EnvConfig.TracingEnabled {
		r.Use(otelgin.Middleware(common.Name))
	}

	// Setup global middleware
	r.Use(middleware.HeadMiddleware())
	r.Use(middleware.NewCacheControlMiddleware().Add())
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewCspMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())

	frontendRateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(100*time.Millisecond), 300)
	err := frontend.RegisterFrontend(r, frontendRateLimitMiddleware)
	if errors.Is(err, frontend.ErrFrontendNotIncluded) {
		slog.Warn("Frontend is not included in the build. Skipping frontend registration.")
	} else if err != nil {
		return nil, fmt.Errorf("failed to register frontend: %w", err)
	}

	// Initialize middleware for specific routes
	authMiddleware := middleware.NewAuthMiddleware(svc.apiKeyService, svc.userService, svc.jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	apiRateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 100)

	// Set up API routes
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

	// Add test controller in non-production environments
	if !common.EnvConfig.AppEnv.IsProduction() {
		for _, f := range registerTestControllers {
			f(apiGroup, db, svc)
		}
	}

	// Set up base routes
	baseGroup := r.Group("/", apiRateLimitMiddleware)
	controller.NewWellKnownController(baseGroup, svc.jwtService)

	// Set up healthcheck routes
	// These are not rate-limited
	controller.NewHealthzController(r)

	var protocols http.Protocols
	protocols.SetHTTP1(true)

	var tlsConfig *tls.Config
	var certProvider *tlsCertProvider
	var certWatcher *fsnotify.Watcher

	if common.EnvConfig.TLSCertFile != "" && common.EnvConfig.TLSKeyFile != "" {
		protocols.SetHTTP2(true)

		certProvider, err = newCertProvider(common.EnvConfig.TLSCertFile, common.EnvConfig.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			GetCertificate: certProvider.GetCertificate,
			MinVersion:     tls.VersionTLS13,
			NextProtos:     []string{"h2"},
		}

		slog.Info("TLS enabled")
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	// Set up the server
	srv := &http.Server{
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 10 * time.Second,
		Protocols:         &protocols,
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

	// Set up the listener
	network := "tcp"
	addr := net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port)
	if common.EnvConfig.UnixSocket != "" {
		network = "unix"
		addr = common.EnvConfig.UnixSocket
		os.Remove(addr) // remove dangling the socket file to avoid file-exist error
	}

	listener, err := net.Listen(network, addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to create %s listener: %w", network, err)
	}

	// Set the socket mode if using a Unix socket
	if network == "unix" && common.EnvConfig.UnixSocketMode != "" {
		mode, err := strconv.ParseUint(common.EnvConfig.UnixSocketMode, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}

		if err := os.Chmod(addr, os.FileMode(mode)); err != nil {
			return nil, fmt.Errorf("failed to set UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}
	}

	// Service runner function
	runFn := func(ctx context.Context) error {
		slog.Info("Server listening", slog.String("addr", addr), slog.Bool("tls", tlsConfig != nil))

		// Set up certificate hot reloading if TLS is enabled
		if certProvider != nil {
			certWatcher, err = fsnotify.NewWatcher()
			if err != nil {
				return fmt.Errorf("failed to create certificate watcher: %w", err)
			}

			// Watch both certificate and key files
			if err := certWatcher.Add(common.EnvConfig.TLSCertFile); err != nil {
				return fmt.Errorf("failed to watch TLS certificate: %w", err)
			}
			if err := certWatcher.Add(common.EnvConfig.TLSKeyFile); err != nil {
				return fmt.Errorf("failed to watch TLS key: %w", err)
			}

			// Start certificate watcher goroutine
			go certProvider.StartWatching(ctx, certWatcher)
		}

		// Start the server in a background goroutine
		go func() {
			defer listener.Close()

			// Next call blocks until the server is shut down
			var srvErr error
			if tlsConfig != nil {
				srvErr = srv.Serve(tls.NewListener(listener, tlsConfig))
			} else {
				srvErr = srv.Serve(listener)
			}

			if srvErr != http.ErrServerClosed {
				slog.Error("Error starting app server", "error", srvErr)
				os.Exit(1)
			}
		}()

		// Notify systemd that we are ready
		err = systemd.SdNotifyReady()
		if err != nil {
			// Log the error only
			slog.Warn("Unable to notify systemd that the service is ready", "error", err)
		}

		// Block until the context is canceled
		<-ctx.Done()

		// Handle graceful shutdown
		// Note we use the background context here as ctx has been canceled already
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := srv.Shutdown(shutdownCtx) //nolint:contextcheck
		shutdownCancel()
		if shutdownErr != nil {
			// Log the error only (could be context canceled)
			slog.Warn("App server shutdown error", "error", shutdownErr)
		}

		// Close certificate watcher
		if certWatcher != nil {
			certWatcher.Close()
		}

		return nil
	}

	return runFn, nil
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
