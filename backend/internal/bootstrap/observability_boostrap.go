package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	sloggin "github.com/gin-contrib/slog"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	globallog "go.opentelemetry.io/otel/log/global"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func defaultResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(common.Name),
			semconv.ServiceVersion(common.Version),
		),
	)
}

func initObservability(ctx context.Context, metrics, traces bool) (shutdownFns []utils.Service, httpClient *http.Client, err error) {
	resource, err := defaultResource()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	shutdownFns = make([]utils.Service, 0, 2)

	httpClient = &http.Client{}
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// Indicates a development-time error
		panic("Default transport is not of type *http.Transport")
	}
	httpClient.Transport = defaultTransport.Clone()

	// Logging
	err = initOtelLogging(ctx, resource)
	if err != nil {
		return nil, nil, err
	}

	// Tracing
	tracingShutdownFn, err := initOtelTracing(ctx, traces, resource, httpClient)
	if err != nil {
		return nil, nil, err
	} else if tracingShutdownFn != nil {
		shutdownFns = append(shutdownFns, tracingShutdownFn)
	}

	// Metrics
	metricsShutdownFn, err := initOtelMetrics(ctx, metrics, resource)
	if err != nil {
		return nil, nil, err
	} else if metricsShutdownFn != nil {
		shutdownFns = append(shutdownFns, metricsShutdownFn)
	}

	return shutdownFns, httpClient, nil
}

func initOtelLogging(ctx context.Context, resource *resource.Resource) error {
	// If the env var OTEL_LOGS_EXPORTER is empty, we set it to "none", for autoexport to work
	if os.Getenv("OTEL_LOGS_EXPORTER") == "" {
		os.Setenv("OTEL_LOGS_EXPORTER", "none")
	}
	exp, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry log exporter: %w", err)
	}

	level, _ := sloggin.ParseLevel(common.EnvConfig.LogLevel)

	// Create the handler
	var handler slog.Handler
	if common.EnvConfig.LogJSON {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			TimeFormat: time.Stamp,
			Level:      level,
			NoColor:    !isatty.IsTerminal(os.Stdout.Fd()),
		})
	}

	// Create the logger provider
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(exp),
		),
		sdklog.WithResource(resource),
	)

	// Set the logger provider globally
	globallog.SetLoggerProvider(provider)

	// Wrap the handler in a "fanout" one
	handler = utils.LogFanoutHandler{
		handler,
		otelslog.NewHandler(common.Name, otelslog.WithLoggerProvider(provider)),
	}

	// Set the default slog to send logs to OTel and add the app name
	log := slog.New(handler).
		With(slog.String("app", common.Name)).
		With(slog.String("version", common.Version))
	slog.SetDefault(log)

	return nil
}

func initOtelTracing(ctx context.Context, traces bool, resource *resource.Resource, httpClient *http.Client) (shutdownFn utils.Service, err error) {
	if !traces {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		return nil, nil
	}

	tr, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry span exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithBatcher(tr),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	shutdownFn = func(shutdownCtx context.Context) error { //nolint:contextcheck
		tpCtx, tpCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer tpCancel()
		shutdownErr := tp.Shutdown(tpCtx)
		if shutdownErr != nil {
			return fmt.Errorf("failed to gracefully shut down traces exporter: %w", shutdownErr)
		}
		return nil
	}

	// Add tracing to the HTTP client
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)

	return shutdownFn, nil
}

func initOtelMetrics(ctx context.Context, metrics bool, resource *resource.Resource) (shutdownFn utils.Service, err error) {
	if !metrics {
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		return nil, nil
	}

	mr, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry metric reader: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(resource),
		metric.WithReader(mr),
	)
	otel.SetMeterProvider(mp)

	shutdownFn = func(shutdownCtx context.Context) error { //nolint:contextcheck
		mpCtx, mpCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer mpCancel()
		shutdownErr := mp.Shutdown(mpCtx)
		if shutdownErr != nil {
			return fmt.Errorf("failed to gracefully shut down metrics exporter: %w", shutdownErr)
		}
		return nil
	}

	return shutdownFn, nil
}
