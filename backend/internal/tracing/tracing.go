// Package tracing wraps the global OpenTelemetry tracer for Pocket ID
// It centralizes the instrumentation name, the span helpers, and the attribute keys so spans are named and tagged consistently across packages
// The tracer is resolved from the global provider, so every span is a no-op until the embedding app or the runtime binary configures OpenTelemetry
package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// tracer delegates to the global provider, picking up the real provider once one is installed
var tracer = otel.Tracer(common.TracerName)

// Start begins a span as a child of the span in ctx, returning the derived context and the span
// The caller must end the span, normally with End so the status is set from the operation error
//
//nolint:spancheck // the span is intentionally returned for the caller to end
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// End finishes the span, recording err and marking the span failed when err is non-nil
func End(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

// EndExpected finishes the span like End, but treats any of the benign errors as a normal outcome rather than a failure
// It is used, for example, for operations whose "not found" result is expected
func EndExpected(span trace.Span, err error, benign ...error) {
	for _, b := range benign {
		if errors.Is(err, b) {
			span.End()
			return
		}
	}

	End(span, err)
}

// Fail records err on the span in ctx and marks it failed, without ending it
// It is used where the operation reports failure through a non-error value, such as a structured protocol error
func Fail(ctx context.Context, msg string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Error, msg)
}

// JobID returns the attribute for a job ID
func JobID(v string) attribute.KeyValue {
	return attribute.String("pocketid.job.id", v)
}
