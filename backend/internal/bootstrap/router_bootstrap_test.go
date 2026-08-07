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
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/stretchr/testify/require"
)

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
	require.Contains(t, logLine, `msg="HTTP request completed"`)
	require.Contains(t, logLine, "request_id="+body.RequestID)
	require.Contains(t, logLine, "error_code=not_signed_in")
	require.Contains(t, logLine, "status=401")
	require.NotContains(t, logLine, "Request with errors")
	require.NotContains(t, logLine, "Error #01")
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
