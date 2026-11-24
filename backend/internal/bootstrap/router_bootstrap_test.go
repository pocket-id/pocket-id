package bootstrap

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/geolite"
	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestCloudflareLocationMiddlewareOnlyWhenEnabled(t *testing.T) {
	previousConfig := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = previousConfig })
	gin.SetMode(gin.TestMode)

	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			common.EnvConfig.CloudflareLocationHeaders = enabled
			locator, geoLiteModule, err := initIPLocationResolver(t.Context(), &common.EnvConfigSchema{
				GeoLiteDBPath:             filepath.Join(t.TempDir(), "missing.mmdb"),
				CloudflareLocationHeaders: enabled,
			})
			require.NoError(t, err)
			if enabled {
				require.IsType(t, iplocation.NewCloudflareResolver(), locator)
				require.Nil(t, geoLiteModule)
			} else {
				require.IsType(t, &geolite.Module{}, locator)
				require.Same(t, geoLiteModule, locator)
			}

			// Exercise the real global middleware with a Cloudflare client IP instead of the proxy's IP
			router := gin.New()
			router.TrustedPlatform = "CF-Connecting-IP"
			registerGlobalMiddleware(router)
			router.GET("/api/test", func(c *gin.Context) {
				country, city, err := locator.GetLocationByIP(c.Request.Context(), c.ClientIP())
				require.NoError(t, err)
				c.JSON(http.StatusOK, gin.H{"ip": c.ClientIP(), "country": country, "city": city})
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
			request.RemoteAddr = "192.168.1.1:1234"
			request.Header.Set("cf-connecting-ip", "81.2.69.142")
			request.Header.Set("cf-ipcountry", "CH")
			request.Header.Set("cf-ipcity", "Zürich")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusOK, recorder.Code)

			var location map[string]string
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &location))
			require.Equal(t, "81.2.69.142", location["ip"])
			if enabled {
				require.Equal(t, "Switzerland", location["country"])
				require.Equal(t, "Zürich", location["city"])
			} else {
				require.Empty(t, location["country"])
				require.Empty(t, location["city"])
			}
		})
	}
}

func TestRequestLoggerUsesStructuredErrorMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	router.GET("/api/users/me", func(c *gin.Context) {
		_ = c.Error(apperror.NotSignedIn())
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/users/me", nil)
	router.ServeHTTP(recorder, request)

	var body struct {
		RequestID string `json:"request_id"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.NotEmpty(t, body.RequestID)

	logLine := output.String()
	require.Contains(t, logLine, "level=INFO")
	require.Contains(t, logLine, `msg="GET /api/users/me 401 `)
	require.Contains(t, logLine, "request_id="+body.RequestID)
	require.Contains(t, logLine, "error_code=not_signed_in")
	require.Contains(t, logLine, `error="You are not signed in"`)
	require.NotContains(t, logLine, "user_agent=")
	require.NotContains(t, logLine, "Request with errors")
	require.NotContains(t, logLine, "Error #01")
}

func TestRequestLoggerKeepsAllFieldsAtDebugLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousConfig := common.EnvConfig
	common.EnvConfig.LogLevel = "debug"
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		common.EnvConfig = previousConfig
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/status", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/status", nil)
	request.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(httptest.NewRecorder(), request)

	logLine := output.String()
	require.Contains(t, logLine, `msg="GET /api/status 204 `)
	require.Contains(t, logLine, "status=204")
	require.Contains(t, logLine, "path=/api/status")
	require.Contains(t, logLine, "user_agent=test-agent")
}

func TestRequestLoggerBoundsClientValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousConfig := common.EnvConfig
	common.EnvConfig.LogLevel = "debug"
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		common.EnvConfig = previousConfig
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x%0Afake"+strings.Repeat("a", 10_000), nil)
	request.Header.Set("User-Agent", strings.Repeat("b", 10_000))
	router.ServeHTTP(httptest.NewRecorder(), request)

	// The path stays escaped in the message so a decoded newline can't forge a line in a log that writes it raw
	logLine := output.String()
	require.Contains(t, logLine, `msg="GET /x%0Afake`)
	require.Less(t, len(logLine), 4096)
}

func TestRequestLoggerKeepsRateLimitsAtWarningLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	router.GET("/api/limited", func(c *gin.Context) {
		_ = c.Error(apperror.TooManyRequests())
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/limited", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, output.String(), "level=WARN")
	require.Contains(t, output.String(), "error_code=rate_limited")
}

func TestRequestLoggerRespectsConfiguredMinimumLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/status", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/status", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, output.String())
}

func TestRequestLoggerLogsAtConfiguredMinimumLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/status", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/status", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Contains(t, output.String(), "level=INFO")
}

func TestRequestLoggerOmitsTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/users/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	requests := map[string]string{ //nolint:gosec // G101 false positive: placeholder tokens, not credentials
		"GET /api/users/user-id?search=secret": "/api/users/user-id",
		"GET /st/secret?redirect=/settings":    "/st/:token",
		"GET /lc/secret":                       "/lc/:token",
	}
	for request, wantPath := range requests {
		t.Run(request, func(t *testing.T) {
			output.Reset()
			method, target, _ := strings.Cut(request, " ")
			req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
			req.Header.Set("Referer", "https://pocket-id.example/lc/secret")
			router.ServeHTTP(httptest.NewRecorder(), req)

			require.Contains(t, output.String(), " "+wantPath+" ")
			require.NotContains(t, output.String(), "secret")
		})
	}
}

// A HEAD response with a body, or one that closes the connection unannounced, makes a reverse proxy fail the next request it sends on that connection
func TestHeadResponsesHaveNoBodyAndKeepConnectionOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.HeadMiddleware())
	router.GET("/api/users", func(c *gin.Context) {
		c.String(http.StatusOK, "users")
	})

	server := httptest.NewServer(newHTTPServer(router, nil).Handler)
	t.Cleanup(server.Close)

	// The path with the trailing slash is answered by Gin's redirect, which doesn't run any middleware
	responses := map[string]string{
		"/api/users":  "HTTP/1.1 200 OK\r\n",
		"/api/users/": "HTTP/1.1 301 Moved Permanently\r\n",
	}
	for path, wantStatusLine := range responses {
		t.Run(path, func(t *testing.T) {
			conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", server.Listener.Addr().String())
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			// Send a second request on the same connection to see what follows the response to the HEAD request
			_, err = fmt.Fprintf(conn, "HEAD %s HTTP/1.1\r\nHost: localhost\r\n\r\nGET /api/users HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n", path)
			require.NoError(t, err)
			stream, err := io.ReadAll(conn)
			require.NoError(t, err)

			headResponse, nextResponse, found := strings.Cut(string(stream), "\r\n\r\n")
			require.True(t, found)
			require.True(t, strings.HasPrefix(headResponse, wantStatusLine), headResponse)
			require.True(t, strings.HasPrefix(nextResponse, "HTTP/1.1 200 OK\r\n"), "bytes after the HEAD response: %q", nextResponse)
			require.True(t, strings.HasSuffix(nextResponse, "\r\n\r\nusers"), nextResponse)
		})
	}
}

func TestNewCertProviderSupportsInlineCertificateData(t *testing.T) {
	certPEM, keyPEM := newTestTLSKeyPair(t, 1)

	provider, err := newCertProvider(certPEM, keyPEM, "", "")
	require.NoError(t, err)
	require.Empty(t, provider.certFile)
	require.Empty(t, provider.keyFile)
	require.True(t, certProviderHasSerial(provider, 1))

	watcher, err := startCertWatcher(t.Context(), provider)
	require.NoError(t, err)
	require.Nil(t, watcher)
}

func TestCertProviderReloadsAfterRepeatedAtomicReplacement(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")
	writeTestTLSKeyPair(t, certFile, keyFile, 1)

	provider, err := newCertProvider("", "", certFile, keyFile)
	require.NoError(t, err)
	require.True(t, certProviderHasSerial(provider, 1))

	ctx, cancel := context.WithCancel(t.Context())
	watcher, err := startCertWatcher(ctx, provider)
	require.NoError(t, err)
	t.Cleanup(func() {
		cancel()
		closeCertWatcher(watcher)
	})

	for serial := int64(2); serial <= 3; serial++ {
		replaceTestTLSKeyPair(t, certFile, keyFile, serial)
		require.Eventually(t, func() bool {
			return certProviderHasSerial(provider, serial)
		}, 5*time.Second, 50*time.Millisecond)
	}
}

func replaceTestTLSKeyPair(t *testing.T, certFile, keyFile string, serial int64) {
	t.Helper()

	replacementCertFile := certFile + ".new"
	replacementKeyFile := keyFile + ".new"
	writeTestTLSKeyPair(t, replacementCertFile, replacementKeyFile, serial)
	require.NoError(t, os.Rename(replacementCertFile, certFile))
	require.NoError(t, os.Rename(replacementKeyFile, keyFile))
}

func writeTestTLSKeyPair(t *testing.T, certFile, keyFile string, serial int64) {
	t.Helper()

	certPEM, keyPEM := newTestTLSKeyPair(t, serial)
	require.NoError(t, os.WriteFile(certFile, []byte(certPEM), 0600))
	require.NoError(t, os.WriteFile(keyFile, []byte(keyPEM), 0600))
}

func newTestTLSKeyPair(t *testing.T, serial int64) (string, string) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM)
}

func certProviderHasSerial(provider *tlsCertProvider, serial int64) bool {
	cert, err := provider.GetCertificate(nil)
	if err != nil || cert == nil || len(cert.Certificate) == 0 {
		return false
	}

	parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
	return err == nil && parsedCert.SerialNumber.Cmp(big.NewInt(serial)) == 0
}
