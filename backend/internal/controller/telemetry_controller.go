package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// maxTelemetryBodySize caps the size of a browser telemetry payload we're willing to forward.
const maxTelemetryBodySize = 1 << 20 // 1 MiB

type telemetryController struct {
	targetURL string
	client    *http.Client
}

// NewTelemetryController registers an endpoint that receives OTLP/HTTP JSON trace payloads from the browser SPA and forwards them to the OpenTelemetry collector the backend already exports to.
// Keeping this same-origin means the SPA needs no CORS-enabled, browser-facing collector.
//
// The endpoint lives under /internal rather than /api: it is intentionally unauthenticated (traces are produced before/without a session) and not traced itself, but it is rate-limited.
//
// The route is only registered when the backend is configured to export traces via OTLP to a resolved collector endpoint; otherwise it's left unregistered and requests return 404.
func NewTelemetryController(router gin.IRouter, rateLimit gin.HandlerFunc) {
	targetURL := otlpTracesEndpoint()
	if os.Getenv("OTEL_TRACES_EXPORTER") != "otlp" {
		return
	}

	tc := &telemetryController{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
	router.POST("/internal/telemetry/traces", rateLimit, tc.forwardTraces)
}

func (tc *telemetryController) forwardTraces(c *gin.Context) {
	body := http.MaxBytesReader(c.Writer, c.Request.Body, maxTelemetryBodySize)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, tc.targetURL, body)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to build telemetry forward request: %w", err))
		c.Status(http.StatusBadGateway)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to forward telemetry to collector: %w", err))
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	c.Status(resp.StatusCode)
}

// otlpTracesEndpoint derives the collector's OTLP/HTTP traces URL from the standard OpenTelemetry environment variables.
// It returns an empty string when no OTLP endpoint is configured, in which case the telemetry route is not registered.
// The endpoint must speak OTLP/HTTP (typically port 4318).
func otlpTracesEndpoint() string {
	v := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if v != "" {
		return v
	}

	base := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if base == "" {
		return ""
	}

	return strings.TrimRight(base, "/") + "/v1/traces"
}
