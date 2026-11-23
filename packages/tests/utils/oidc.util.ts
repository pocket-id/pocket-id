import type { Page } from '@playwright/test';

export async function interceptCallbackRedirect(
	page: Page,
	callbackPath: string,
	action: () => Promise<void>,
	timeoutMs: number = 10000
): Promise<URL> {
	const routeMatcher = (url: URL) => url.pathname === callbackPath;
	const callbackPromise = page
		.waitForRequest((request) => routeMatcher(new URL(request.url())), { timeout: timeoutMs })
		.then((request) => new URL(request.url()));

	const actionPromise = action().catch((error) => error);
	const callbackUrl = await callbackPromise;
	await actionPromise;
	return callbackUrl;
}

export async function getUserCode(
	page: Page,
	clientId: string,
	clientSecret: string
): Promise<string> {
	return page.request
		.post('/api/oidc/device/authorize', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				client_id: clientId,
				client_secret: clientSecret,
				scope: 'openid profile email'
			}
		})
		.then((r) => r.json())
		.then((r) => r.user_code);
}

export async function exchangeCode(
	page: Page,
	params: Record<string, string>
): Promise<{ access_token?: string; token_type?: string; expires_in?: number; error?: string }> {
	return page.request
		.post('/api/oidc/token', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: params
		})
		.then((r) => r.json());
}

export async function pushAuthorizationRequest(
	page: Page,
	params: {
		clientId: string;
		clientSecret?: string;
		scope?: string;
		redirectUri?: string;
		responseType?: string;
		codeChallenge?: string;
		codeChallengeMethod?: string;
		nonce?: string;
		state?: string;
		responseMode?: string;
	}
): Promise<{
	request_uri?: string;
	expires_in?: number;
	error?: string;
	error_description?: string;
}> {
	const form: Record<string, string> = {
		client_id: params.clientId,
		response_type: params.responseType ?? 'code',
		scope: params.scope ?? 'openid profile email',
		state: params.state ?? 'nXx-6Qr-owc1SHBa'
	};
	if (params.redirectUri) form.redirect_uri = params.redirectUri;
	if (params.clientSecret) form.client_secret = params.clientSecret;
	if (params.codeChallenge) form.code_challenge = params.codeChallenge;
	if (params.codeChallengeMethod) form.code_challenge_method = params.codeChallengeMethod;
	if (params.nonce) form.nonce = params.nonce;
	if (params.state) form.state = params.state;
	if (params.responseMode) form.response_mode = params.responseMode;

	return page.request
		.post('/api/oidc/par', {
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			form
		})
		.then((r) => r.json());
}

export async function getClientAssertion(
	page: Page,
	data: { issuer: string; audience: string; subject: string }
): Promise<string> {
	return page.request
		.post('/api/externalidp/sign', {
			data: {
				iss: data.issuer,
				aud: data.audience,
				sub: data.subject
			}
		})
		.then((r) => r.text());
}
