// Frontend tracing built on the OpenTelemetry Web SDK.
//
// A page-level span is started for each page view and a CLIENT span is created for every API request as its child, with the W3C `traceparent` header injected into the outgoing request.
// The backend server spans therefore join the same trace, so a page view and everything it triggers — frontend, backend and database — show up as one trace rooted at the page view.
//
// Spans are exported over OTLP/HTTP to a same-origin backend endpoint (`/internal/telemetry/traces`), which forwards them to the collector the backend already uses.
// Same-origin means no CORS and no browser-facing collector to configure.
//
// The OpenTelemetry libraries are heavy, so they are only pulled in via dynamic `import()` once tracing is
// enabled (see `setTracingEnabled`). When tracing is disabled they are never loaded and stay out of the main app bundle.

import { browser, version } from '$app/environment';
// Type-only imports are erased at build time, so they don't pull the OpenTelemetry libraries into the bundle.
import type { Context, Span } from '@opentelemetry/api';

const TRACER_NAME = 'pocket-id-frontend';

// Same-origin endpoint that forwards the browser's OTLP payloads to the backend's collector.
const EXPORTER_URL = '/internal/telemetry/traces';

/** Minimal shape of a header carrier we can inject into (satisfied by axios' AxiosHeaders). */
export type HeaderCarrier = { set(name: string, value: string): void };

// Runtime bindings from `@opentelemetry/api`, loaded lazily via dynamic import once tracing is enabled.
// Undefined until the OpenTelemetry libraries have been imported and the tracer provider registered.
let otel: typeof import('@opentelemetry/api') | undefined;

// Memoizes the dynamic import + provider setup so the libraries are loaded and the provider registered only once.
let providerPromise: Promise<void> | undefined;

// Whether the backend is configured to receive traces (OTEL_TRACES_EXPORTER=otlp with a resolvable collector endpoint).
// It stays false until the app configuration is loaded, so nothing is exported to `/internal/telemetry/traces` when tracing is disabled (that route isn't even registered then).
let tracingEnabled = false;

// The current page-level span and a Context carrying it.
// API request spans are created as its children so a page view groups everything it triggers into one trace.
// `pageContext` is undefined while no page trace is active.
let pageSpan: Span | undefined;
let pageContext: Context | undefined;

/**
 * Enables or disables frontend tracing based on the backend configuration.
 * Called from the root layout once the application configuration is loaded.
 * When enabled, the OpenTelemetry libraries are dynamically imported and the tracer provider is set up;
 * when disabled, they are never loaded, so nothing is sent to the backend telemetry endpoint and the libraries stay out of the main bundle.
 */
export async function setTracingEnabled(enabled: boolean): Promise<void> {
	tracingEnabled = enabled;
	if (enabled) {
		await ensureProvider();
	}
}

// Dynamically imports the OpenTelemetry libraries (so they're only fetched when tracing is enabled) and installs the tracer provider.
// Safe to call multiple times: the work is memoized and only runs once.
function ensureProvider(): Promise<void> {
	if (!browser || !tracingEnabled) {
		return Promise.resolve();
	}
	providerPromise ??= loadAndRegister();
	return providerPromise;
}

async function loadAndRegister(): Promise<void> {
	try {
		const [
			api,
			{ OTLPTraceExporter },
			{ resourceFromAttributes },
			{ BatchSpanProcessor, WebTracerProvider },
			{ ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION }
		] = await Promise.all([
			import('@opentelemetry/api'),
			import('@opentelemetry/exporter-trace-otlp-http'),
			import('@opentelemetry/resources'),
			import('@opentelemetry/sdk-trace-web'),
			import('@opentelemetry/semantic-conventions')
		]);

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

		otel = api;
	} catch (e) {
		// Tracing is best-effort: never let a failure to load the OpenTelemetry libraries break the app.
		console.error('Failed to initialize frontend tracing', e);
	}
}

function endPageSpan(): void {
	pageSpan?.end();
	pageSpan = undefined;
	pageContext = undefined;
}

/**
 * Starts a new page-level trace, ending the previous one, so that a page view and the API calls it triggers form a single trace rooted at the page view.
 * Called on each navigation from the root +layout.
 */
export function startPageTrace(path?: string): void {
	if (!browser || !tracingEnabled || !otel) {
		return;
	}
	endPageSpan();

	pageSpan = otel.trace
		.getTracer(TRACER_NAME)
		.startSpan(path ? `pageview ${path}` : 'pageview', {
			root: true,
			kind: otel.SpanKind.INTERNAL
		});
	pageContext = otel.trace.setSpan(otel.ROOT_CONTEXT, pageSpan);
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
	if (!tracingEnabled || !otel) {
		return undefined;
	}

	// If no navigation has happened yet (e.g. the initial page load), start the page trace lazily so the initial requests are grouped under it too.
	if (!pageContext) {
		startPageTrace(typeof location === 'undefined' ? undefined : location.pathname);
	}

	const parent = pageContext ?? otel.ROOT_CONTEXT;
	const span = otel.trace.getTracer(TRACER_NAME).startSpan(
		`${method} ${url}`,
		{
			kind: otel.SpanKind.CLIENT,
			attributes: {
				'http.request.method': method,
				'url.full': url
			}
		},
		parent
	);

	otel.propagation.inject(otel.trace.setSpan(parent, span), carrier, {
		set: (c, key, value) => c.set(key, value)
	});

	return span;
}

/** Ends a request span, recording the response status (and error, if any). */
export function endRequestSpan(span: Span | undefined, statusCode?: number, error?: unknown): void {
	// A span can only exist when the OpenTelemetry libraries were loaded, so `otel` is defined here.
	if (!span || !otel) {
		return;
	}
	if (statusCode !== undefined) {
		span.setAttribute('http.response.status_code', statusCode);
	}
	if (error) {
		span.recordException(error instanceof Error ? error : String(error));
		span.setStatus({ code: otel.SpanStatusCode.ERROR });
	} else if (statusCode !== undefined && statusCode >= 400) {
		span.setStatus({ code: otel.SpanStatusCode.ERROR });
	}
	span.end();
}
