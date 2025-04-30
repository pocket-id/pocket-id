package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func defaultResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName("pocket-id-backend"),
			semconv.ServiceVersion(common.Version),
		),
	)
}

func initOtel(ctx context.Context, metrics, traces bool) (shutdownFns []utils.Service, httpClient *http.Client, err error) {
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

	if traces {
		tr, err := autoexport.NewSpanExporter(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry span exporter: %w", err)
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

		shutdownFns = append(shutdownFns, func(shutdownCtx context.Context) error { //nolint:contextcheck
			tpCtx, tpCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
			defer tpCancel()
			shutdownErr := tp.Shutdown(tpCtx)
			if shutdownErr != nil {
				return fmt.Errorf("failed to gracefully shut down traces exporter: %w", shutdownErr)
			}
			return nil
		})

		httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)
	} else {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
	}

	if metrics {
		mr, err := autoexport.NewMetricReader(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry metric reader: %w", err)
		}
		mp := metric.NewMeterProvider(
			metric.WithResource(resource),
			metric.WithReader(mr),
		)

		otel.SetMeterProvider(mp)
		shutdownFns = append(shutdownFns, func(shutdownCtx context.Context) error { //nolint:contextcheck
			mpCtx, mpCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
			defer mpCancel()
			shutdownErr := mp.Shutdown(mpCtx)
			if shutdownErr != nil {
				return fmt.Errorf("failed to gracefully shut down metrics exporter: %w", shutdownErr)
			}
			return nil
		})
	} else {
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
	}

	return shutdownFns, httpClient, nil
}
