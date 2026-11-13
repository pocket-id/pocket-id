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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	sloggin "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/italypaleale/francis/builtin/ratelimit"
	"github.com/italypaleale/go-kit/servicerunner"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/frontend"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/tracing"
	"github.com/pocket-id/pocket-id/backend/internal/utils/systemd"
)

// This is used to register additional modules for tests
var registerTestModules []func(apiRouter *authz.Router, db *gorm.DB, svc *services)

func initRouter(db *gorm.DB, svc *services, rateLimitServices map[string]*ratelimit.RateLimitService) (servicerunner.Service, error) {
	r, err := initEngine()
	if err != nil {
		return nil, err
	}
	auth := middleware.NewAuthorization(svc.apiKeyModule, svc.oidcModule, svc.usersModule, svc.sessionModule, svc.appConfigService)
	err = registerRoutes(r, db, svc, auth, rateLimitServices)
	if err != nil {
		return nil, err
	}

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
	err := configureEngine(r)
	if err != nil {
		return nil, err
	}
	registerGlobalMiddleware(r)

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

func configureEngine(r *gin.Engine) error {
	err := r.SetTrustedProxies(common.EnvConfig.TrustProxy)
	if err != nil {
		return fmt.Errorf("failed to configure trusted proxies: %w", err)
	}

	if common.EnvConfig.TrustedPlatform != "" {
		r.TrustedPlatform = common.EnvConfig.TrustedPlatform
	}

	r.Use(otelgin.Middleware(
		common.Name,
		otelgin.WithFilter(shouldTraceRequest)),
	)

	return nil
}

// shouldTraceRequest reports whether an incoming request should be traced.
// It traces only requests handled by real backend routes (the API, the OIDC/OAuth endpoints, the MCP endpoint, and the well-known documents).
// Everything else falls through to the frontend NoRoute handler, which serves the SPA shell and static assets; tracing those would produce noisy, unparented spans named just "GET" with an empty http.route.
func shouldTraceRequest(r *http.Request) bool {
	p := r.URL.Path
	switch {
	case strings.HasPrefix(p, "/api/"),
		strings.HasPrefix(p, "/.well-known/"),
		p == "/authorize",
		p == authz.MCPPath:
		return true
	default:
		return false
	}
}

func registerGlobalMiddleware(r *gin.Engine) {
	if common.EnvConfig.CloudflareLocationHeaders {
		r.Use(middleware.CloudflareLocationMiddleware())
	}
	r.Use(middleware.HeadMiddleware())
	r.Use(middleware.NewBodySizeLimitMiddleware().Add(1 << 20))
	r.Use(middleware.NewCacheControlMiddleware().Add())
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewCspMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())
	r.Use(middleware.NewCrossOriginProtectionMiddleware(common.EnvConfig.AppURL).Add())
}

// registerTools collects the MCP tools of every package, the way registerRoutes collects their routes
func registerTools(t *mcp.Tools, svc *services) {
	svc.usersModule.RegisterTools(t)
	svc.userGroupsModule.RegisterTools(t)
	svc.oidcModule.RegisterTools(t)
	svc.oneTimeAccessModule.RegisterTools(t)
	svc.userSignUpModule.RegisterTools(t)
	svc.auditLogsModule.RegisterTools(t)
}

func registerRoutes(r *gin.Engine, db *gorm.DB, svc *services, auth *authz.Middleware, rateLimitServices map[string]*ratelimit.RateLimitService) error {

	err := frontend.RegisterFrontend(r)
	if errors.Is(err, frontend.ErrFrontendNotIncluded) {
		slog.Warn("Frontend is not included in the build. Skipping frontend registration.")
	} else if err != nil {
		return fmt.Errorf("failed to register frontend: %w", err)
	}

	// Initialize middleware for specific routes
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitServices)
	apiRateLimitMiddleware := rateLimitMiddleware.Add(middleware.RateLimitAPI)

	apiGroup := r.Group("/api", apiRateLimitMiddleware)
	// Decode "~<base64url>" client ID path params (used for CIMD URL client IDs).
	apiGroup.Use(middleware.NewClientIDParamMiddleware().Add())
	baseGroup := r.Group("/", apiRateLimitMiddleware)

	// Every route below declares the scope it requires, or that it is public, through these routers
	apiRouter := auth.Router(apiGroup)
	baseRouter := auth.Router(baseGroup)

	svc.apiKeyModule.RegisterRoutes(apiRouter)
	svc.sessionModule.RegisterRoutes(apiRouter)
	svc.webauthnModule.RegisterRoutes(apiRouter,
		rateLimitMiddleware.Add(middleware.RateLimitWebauthnLogin),
		rateLimitMiddleware.Add(middleware.RateLimitWebauthnReauthenticate),
	)
	svc.deviceLoginModule.RegisterRoutes(apiRouter,
		rateLimitMiddleware.Add(middleware.RateLimitDeviceLoginCreate),
		rateLimitMiddleware.Add(middleware.RateLimitDeviceLoginExchange),
		rateLimitMiddleware.Add(middleware.RateLimitDeviceLoginVerification),
	)
	svc.usersModule.RegisterRoutes(apiRouter, fileSizeLimitMiddleware.Add(10<<20))
	svc.claimMappingPolicyModule.RegisterRoutes(apiRouter)
	svc.appConfigService.RegisterRoutes(apiRouter, baseGroup, svc.emailModule)
	svc.ldapSyncModule.RegisterRoutes(apiRouter)
	svc.appAssetsModule.RegisterRoutes(apiRouter, fileSizeLimitMiddleware.Add(10<<20))
	svc.auditLogsModule.RegisterRoutes(apiRouter)
	svc.userGroupsModule.RegisterRoutes(apiRouter)
	svc.apiModule.RegisterRoutes(apiRouter)
	svc.customClaimsModule.RegisterRoutes(apiRouter)
	svc.environmentModule.RegisterRoutes(apiRouter)
	svc.logoPresetModule.RegisterRoutes(apiRouter)
	svc.featureModule.RegisterRoutes(apiRouter)
	svc.scimSyncModule.RegisterRoutes(apiRouter)
	svc.userSignUpModule.RegisterRoutes(apiRouter, rateLimitMiddleware.Add(middleware.RateLimitSignup))
	svc.oneTimeAccessModule.RegisterRoutes(apiRouter,
		rateLimitMiddleware.Add(middleware.RateLimitOneTimeAccessToken),
		rateLimitMiddleware.Add(middleware.RateLimitOneTimeAccessEmail),
	)
	svc.emailVerificationModule.RegisterRoutes(apiRouter, rateLimitMiddleware.Add(middleware.RateLimitVerifyEmail))
	svc.oidcModule.RegisterRoutes(baseRouter, apiRouter, fileSizeLimitMiddleware.Add(2<<20))
	svc.mcpModule.RegisterRoutes(baseRouter)

	registerTestRoutes(apiRouter, db, svc)

	// These are not rate-limited.
	auth.Router(r.Group("/")).Public().GET("/healthz", httpserver.NoContent(func(*gin.Context) error { return nil }),
		option.Summary("Responds to healthchecks"),
		option.Description("Responds with a successful status code to healthcheck requests"),
		option.Tags("Health"),
	)

	// Receives OTLP trace payloads from the browser SPA (POST /internal/telemetry/traces) and forwards them to the collector, when trace export is enabled.
	// Outside /api, so it's unauthenticated and not traced, but it is rate-limited.
	tracing.NewTelemetryController(r, rateLimitMiddleware.Add(middleware.RateLimitInternal))

	return nil
}

func registerTestRoutes(apiRouter *authz.Router, db *gorm.DB, svc *services) {
	if common.EnvConfig.AppEnv.IsProduction() {
		return
	}

	for _, f := range registerTestModules {
		f(apiRouter, db, svc)
	}
}

func initServer(r *gin.Engine) (*serverConfig, error) {
	protocols, tlsConfig, certProvider, err := initServerProtocols()
	if err != nil {
		return nil, err
	}

	var socketFn func() (*socket, error)
	switch {
	case common.EnvConfig.SystemdSocket:
		socketFn = systemdSocket
	case common.EnvConfig.UnixSocket != "":
		socketFn = unixSocket
	default:
		socketFn = tcpSocket
	}

	socket, err := socketFn()
	if err != nil {
		return nil, err
	}

	addr := socket.addr

	listener := socket.listener

	// Wrap the listener with a proxy protocol listener if configured and not using a Unix socket
	if len(common.EnvConfig.ProxyProtocol) > 0 && common.EnvConfig.UnixSocket == "" {
		listener, err = newProxyProtocolListener(socket.listener, common.EnvConfig.ProxyProtocol)
		if err != nil {
			_ = socket.listener.Close()
			return nil, err
		}
	}

	server := newHTTPServer(r, protocols)

	return &serverConfig{addr, certProvider, listener, server, tlsConfig}, nil
}

func initServerProtocols() (*http.Protocols, *tls.Config, *tlsCertProvider, error) {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)

	tlsConfigured := common.EnvConfig.TLSCert != "" || common.EnvConfig.TLSKey != "" ||
		common.EnvConfig.TLSCertFile != "" || common.EnvConfig.TLSKeyFile != ""
	if !tlsConfigured {
		protocols.SetUnencryptedHTTP2(true)
		return protocols, nil, nil, nil
	}

	protocols.SetHTTP2(true)
	certProvider, err := newCertProvider(
		common.EnvConfig.TLSCert,
		common.EnvConfig.TLSKey,
		common.EnvConfig.TLSCertFile,
		common.EnvConfig.TLSKeyFile,
	)
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
		// Some request headers, such as Content-Type, cost a multiple of their size to parse
		MaxHeaderBytes:    64 << 10,
		ReadHeaderTimeout: 10 * time.Second,
		Protocols:         protocols,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// HEAD requests don't get matched by Gin routes, so we convert them to GET
			// middleware.HeadMiddleware will convert them back to HEAD later
			if req.Method == http.MethodHead {
				req.Method = http.MethodGet
				ctx := context.WithValue(req.Context(), middleware.IsHeadRequestCtxKey{}, true)
				req = req.WithContext(ctx)
			}

			r.ServeHTTP(w, req)
		}),
	}
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

	// We do not pass the context because it's already been canceled
	//nolint:contextcheck
	return shutdownServer(config.server)
}

func startCertWatcher(ctx context.Context, certProvider *tlsCertProvider) (*fsnotify.Watcher, error) {
	if certProvider == nil || certProvider.certFile == "" || certProvider.keyFile == "" {
		return nil, nil
	}

	certWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate watcher: %w", err)
	}

	watchedDirectories := make(map[string]struct{}, 2)
	for _, file := range []string{certProvider.certFile, certProvider.keyFile} {
		directory := filepath.Dir(file)
		if _, ok := watchedDirectories[directory]; ok {
			continue
		}

		if err := certWatcher.Add(directory); err != nil {
			_ = certWatcher.Close()
			return nil, fmt.Errorf("failed to watch TLS directory %q: %w", directory, err)
		}
		watchedDirectories[directory] = struct{}{}
	}

	go certProvider.StartWatching(ctx, certWatcher)
	return certWatcher, nil
}

func closeCertWatcher(certWatcher *fsnotify.Watcher) {
	if certWatcher != nil {
		_ = certWatcher.Close()
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

		if !errors.Is(srvErr, http.ErrServerClosed) {
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

func shutdownServer(srv *http.Server) error {
	// Note we use the background context here as ctx has been canceled already
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := srv.Shutdown(shutdownCtx) //nolint:contextcheck
	shutdownCancel()
	if shutdownErr != nil {
		// Log the error only (could be context canceled)
		slog.Warn("App server shutdown error", "error", shutdownErr)
	}

	return nil
}

func initLogger(r *gin.Engine) {
	loggerSkipPathsPrefix := []string{
		"GET /api/application-assets/logo",
		"GET /api/application-assets/background",
		"GET /api/application-assets/favicon",
		"GET /api/application-assets/email",
		"GET /api/application-assets/font",
		"GET /_app",
		"GET /fonts",
		"GET /healthz",
		"HEAD /healthz",
	}

	r.Use(sloggin.SetLogger(
		sloggin.WithLogger(func(_ *gin.Context, _ *slog.Logger) *slog.Logger {
			// gin-contrib/slog calls Handler.Handle directly instead of Logger.LogAttrs
			// Wrapping the default handler in MultiHandler restores the Enabled check that enforces LOG_LEVEL
			return slog.New(slog.NewMultiHandler(slog.Default().Handler()))
		}),
		sloggin.WithClientErrorLevel(slog.LevelInfo),
		sloggin.WithSpecificLogLevelByStatusCode(map[int]slog.Level{
			http.StatusTooManyRequests: slog.LevelWarn,
		}),
		sloggin.WithContext(enrichRequestLog),
		// Skip logging for certain paths to reduce noise in the logs
		// Server errors are always logged because the request log is their only record
		sloggin.WithSkipper(func(c *gin.Context) bool {
			if c.Writer.Status() >= http.StatusInternalServerError {
				return false
			}
			for _, prefix := range loggerSkipPathsPrefix {
				if strings.HasPrefix(c.Request.Method+" "+c.Request.URL.String(), prefix) {
					return true
				}
			}
			return false
		}),
	))
}

// enrichRequestLog rebuilds the request log record with the request ID, the handled error and the trace, leaving out secrets
// Text logs lead with an access log summary and, unless the log level is debug, keep only the client IP of the request fields
func enrichRequestLog(c *gin.Context, record *slog.Record) *slog.Record {
	compact := !common.EnvConfig.LogJSON && common.EnvConfig.LogLevel != "debug"

	// The signup and login code pages are the only paths that carry a token
	path := c.Request.URL.Path
	for _, prefix := range []string{"/st/", "/lc/"} {
		if strings.HasPrefix(path, prefix) {
			path = prefix + ":token"
		}
	}

	// Collect the request fields, dropping the query string and the referer because they carry secrets such as ID token hints and email verification tokens
	var latency time.Duration
	requestAttrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		switch attr.Key {
		case "query", "referer":
			return true
		case "path":
			attr = slog.String("path", path)
		case "latency":
			latency = attr.Value.Duration()
		}
		if !compact || attr.Key == "ip" {
			requestAttrs = append(requestAttrs, attr)
		}
		return true
	})

	// Log processors filter on a fixed message, while people scan text logs by method, path and status
	message := "HTTP request completed"
	if !common.EnvConfig.LogJSON {
		message = fmt.Sprintf("%s %s %d %s", c.Request.Method, path, c.Writer.Status(), latency.Round(time.Microsecond))
	}
	enriched := slog.NewRecord(record.Time, record.Level, message, record.PC)

	// This line is the only log of a failed request, so it carries the error itself
	if err := middleware.RequestError(c); err != nil {
		enriched.AddAttrs(slog.Any("error", err))
	}
	if errorCode := middleware.RequestErrorCode(c); errorCode != "" {
		enriched.AddAttrs(slog.String("error_code", string(errorCode)))
	}
	enriched.AddAttrs(requestAttrs...)
	if requestID := middleware.RequestID(c); requestID != "" {
		enriched.AddAttrs(slog.String("request_id", requestID))
	}

	// The span context is only valid when traces are exported, so the trace ID always leads somewhere
	if spanContext := trace.SpanFromContext(c.Request.Context()).SpanContext(); spanContext.IsValid() {
		enriched.AddAttrs(slog.String("trace_id", spanContext.TraceID().String()))
		if !compact {
			enriched.AddAttrs(slog.String("span_id", spanContext.SpanID().String()))
		}
	}

	return &enriched
}

// tlsCertProvider holds certificates that can be dynamically reloaded
type tlsCertProvider struct {
	certMutex sync.RWMutex
	cert      *tls.Certificate
	certFile  string
	keyFile   string
}

// GetCertificate implements tls.GetCertificate interface for dynamic certificate loading
func (p *tlsCertProvider) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	p.certMutex.RLock()
	defer p.certMutex.RUnlock()
	return p.cert, nil
}

// newCertProvider creates a certificate provider from either inline data or reloadable files
func newCertProvider(certPEM, keyPEM, certFile, keyFile string) (*tlsCertProvider, error) {
	inlineConfigured := certPEM != "" || keyPEM != ""
	fileConfigured := certFile != "" || keyFile != ""

	switch {
	case inlineConfigured && fileConfigured:
		return nil, errors.New("inline and file-based TLS configuration cannot be combined")
	case certPEM != "" && keyPEM == "", certPEM == "" && keyPEM != "":
		return nil, errors.New("inline TLS certificate and key must both be configured")
	case certFile != "" && keyFile == "", certFile == "" && keyFile != "":
		return nil, errors.New("TLS certificate and key files must both be configured")
	case !inlineConfigured && !fileConfigured:
		return nil, errors.New("TLS certificate and key must both be configured")
	}

	var cert tls.Certificate
	var err error
	if inlineConfigured {
		cert, err = tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	} else {
		certFile, err = filepath.Abs(certFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve TLS certificate path: %w", err)
		}
		keyFile, err = filepath.Abs(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve TLS key path: %w", err)
		}
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	}
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
	const debounceDuration = time.Second

	reloadTimer := time.NewTimer(debounceDuration)
	reloadTimer.Stop()
	defer reloadTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Ignore events that are not related to the certificate or key files
			if !p.isCertificateEvent(event) {
				continue
			}

			// Reset the debounce timer so both files can settle before the pair is reloaded
			reloadTimer.Reset(debounceDuration)
			slog.Debug("TLS file change detected, debouncing", slog.String("path", event.Name))

		case <-reloadTimer.C:
			// Reload the pair atomically after the certificate directories have settled
			slog.Info("Reloading TLS certificate")

			if err := p.reloadCertificate(); err != nil {
				slog.Error("Failed to reload TLS certificate", "error", err)
			} else {
				slog.Info("TLS certificate reloaded successfully")
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			slog.Error("Certificate watcher error", "error", err)
		}
	}
}

func (p *tlsCertProvider) isCertificateEvent(event fsnotify.Event) bool {
	if !event.Has(fsnotify.Write | fsnotify.Create | fsnotify.Rename | fsnotify.Remove) {
		return false
	}

	eventPath := filepath.Clean(event.Name)
	return eventPath == p.certFile || eventPath == p.keyFile
}
