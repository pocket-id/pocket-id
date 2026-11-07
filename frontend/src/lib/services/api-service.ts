import { signInAgain } from '#lib/stores/sign-in-again-store.svelte.ts';
import userStore from '#lib/stores/user-store.ts';
import { getAxiosErrorCode } from '#lib/utils/error-util.ts';
import { endRequestSpan, startRequestSpan } from '#lib/utils/tracing-util.ts';
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { Span } from '@opentelemetry/api';
import { get } from 'svelte/store';

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
			async (error: unknown) => {
				if (!(error instanceof AxiosError && error.config)) throw error;
				endRequestSpan(requestSpans.get(error.config), error.response?.status, error);
				requestSpans.delete(error.config);

				// A signed-in person whose session ended signs in again in a dialog, and the request then runs once more
				// Signing out with an ended session has nothing left to end, so it never asks
				if (
					getAxiosErrorCode(error) === 'not_signed_in' &&
					get(userStore) &&
					error.config.url !== '/webauthn/logout' &&
					(await signInAgain.wait())
				) {
					return this.api.request(error.config);
				}
				throw error;
			}
		);
	}
}

export default APIService;
