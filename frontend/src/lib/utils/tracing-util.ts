// Frontend tracing built on the OpenTelemetry Web SDK.
//
// A page-level span is started for each page view and a CLIENT span is created for every API request as its child, with the W3C `traceparent` header injected into the outgoing request.
// The backend server spans therefore join the same trace, so a page view and everything it triggers — frontend, backend and database — show up as one trace rooted at the page view.
//
// Spans are exported over OTLP/HTTP to a same-origin backend endpoint (`/internal/telemetry/traces`), which forwards them to the collector the backend already uses.
// Same-origin means no CORS and no browser-facing collector to configure.

import { browser, version } from '$app/environment';
import {
	ROOT_CONTEXT,
	SpanKind,
	SpanStatusCode,
	propagation,
	trace,
	type Context,
	type Span
} from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { BatchSpanProcessor, WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';

const TRACER_NAME = 'pocket-id-frontend';

// Same-origin endpoint that forwards the browser's OTLP payloads to the backend's collector.
const EXPORTER_URL = '/internal/telemetry/traces';

/** Minimal shape of a header carrier we can inject into (satisfied by axios' AxiosHeaders). */
export type HeaderCarrier = { set(name: string, value: string): void };

let initialized = false;

// Whether the backend is configured to receive traces (OTEL_TRACES_EXPORTER=otlp with a resolvable collector endpoint).
// It stays false until the app configuration is loaded, so nothing is exported to `/internal/telemetry/traces` when tracing is disabled (that route isn't even registered then).
let tracingEnabled = false;

// The current page-level span and a Context carrying it.
// API request spans are created as its children so a page view groups everything it triggers into one trace.
let pageSpan: Span | undefined;
let pageContext: Context = ROOT_CONTEXT;

/**
 * Enables or disables frontend tracing based on the backend configuration.
 * Called from the root layout once the application configuration is loaded.
 * When disabled, no provider is created and no spans are started, so nothing is sent to the backend telemetry endpoint.
 */
export function setTracingEnabled(enabled: boolean): void {
	tracingEnabled = enabled;
}

function ensureProvider(): void {
	if (initialized || !browser || !tracingEnabled) {
		return;
	}
	initialized = true;

	const provider = new WebTracerProvider({
		resource: resourceFromAttributes({
			[ATTR_SERVICE_NAME]: TRACER_NAME,
			[ATTR_SERVICE_VERSION]: version
		}),
		spanProcessors: [new BatchSpanProcessor(new OTLPTraceExporter({ url: EXPORTER_URL }))]
	});

	// register() installs the provider globally together with the default W3C trace context (`traceparent`) propagator.
	provider.register();

	// End (and flush) the current page span when the tab/page goes away, so its trace completes.
	addEventListener('pagehide', () => {
		endPageSpan();
		void provider.forceFlush();
	});
}

function endPageSpan(): void {
	pageSpan?.end();
	pageSpan = undefined;
	pageContext = ROOT_CONTEXT;
}

/**
 * Starts a new page-level trace, ending the previous one, so that a page view and the API calls it triggers form a single trace rooted at the page view.
 * Called on each navigation from the root +layout.
 */
export function startPageTrace(path?: string): void {
	if (!browser || !tracingEnabled) {
		return;
	}
	ensureProvider();
	endPageSpan();

	pageSpan = trace
		.getTracer(TRACER_NAME)
		.startSpan(path ? `pageview ${path}` : 'pageview', { root: true, kind: SpanKind.INTERNAL });
	pageContext = trace.setSpan(ROOT_CONTEXT, pageSpan);
}

/**
 * Creates a CLIENT span for an outgoing API request (as a child of the current page span) and injects the W3C trace headers into `carrier`.
 * The returned span must be ended with  {@link endRequestSpan} once the response (or error) is available.
 */
export function startRequestSpan(
	method: string,
	url: string,
	carrier: HeaderCarrier
): Span | undefined {
	if (!tracingEnabled) {
		return undefined;
	}
	ensureProvider();

	// If no navigation has happened yet (e.g. the initial page load), start the page trace lazily so the initial requests are grouped under it too.
	if (pageContext === ROOT_CONTEXT) {
		startPageTrace(typeof location === 'undefined' ? undefined : location.pathname);
	}

	const span = trace.getTracer(TRACER_NAME).startSpan(
		`${method} ${url}`,
		{
			kind: SpanKind.CLIENT,
			attributes: {
				'http.request.method': method,
				'url.full': url
			}
		},
		pageContext
	);

	propagation.inject(trace.setSpan(pageContext, span), carrier, {
		set: (c, key, value) => c.set(key, value)
	});

	return span;
}

/** Ends a request span, recording the response status (and error, if any). */
export function endRequestSpan(span: Span | undefined, statusCode?: number, error?: unknown): void {
	if (!span) {
		return;
	}
	if (statusCode !== undefined) {
		span.setAttribute('http.response.status_code', statusCode);
	}
	if (error) {
		span.recordException(error instanceof Error ? error : String(error));
		span.setStatus({ code: SpanStatusCode.ERROR });
	} else if (statusCode !== undefined && statusCode >= 400) {
		span.setStatus({ code: SpanStatusCode.ERROR });
	}
	span.end();
}
