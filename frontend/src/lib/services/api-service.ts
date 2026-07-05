import { endRequestSpan, startRequestSpan } from '$lib/utils/tracing-util';
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { Span } from '@opentelemetry/api';

// Tracks the in-flight span for each request config so the response/error interceptor can end it.
const requestSpans = new WeakMap<InternalAxiosRequestConfig, Span>();

abstract class APIService {
	protected api = axios.create({ baseURL: '/api' });

	constructor() {
		if (typeof process !== 'undefined' && process?.env?.DEVELOPMENT_BACKEND_URL) {
			this.api.defaults.baseURL = process.env.DEVELOPMENT_BACKEND_URL;
		}

		// Wrap each API request in a CLIENT span and inject its W3C trace headers, so backend spans correlate with the SPA
		this.api.interceptors.request.use((config) => {
			const method = (config.method ?? 'get').toUpperCase();
			const url = `${config.baseURL ?? ''}${config.url ?? ''}`;
			// startRequestSpan returns undefined when tracing is disabled, so only track a span when one was created.
			const span = startRequestSpan(method, url, config.headers);
			if (span) {
				requestSpans.set(config, span);
			}
			return config;
		});
		this.api.interceptors.response.use(
			(response) => {
				endRequestSpan(requestSpans.get(response.config), response.status);
				requestSpans.delete(response.config);
				return response;
			},
			(error: unknown) => {
				if (error instanceof AxiosError && error.config) {
					endRequestSpan(requestSpans.get(error.config), error.response?.status, error);
					requestSpans.delete(error.config);
				}
				return Promise.reject(error);
			}
		);
	}
}

export default APIService;
