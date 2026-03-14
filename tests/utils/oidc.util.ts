import type { Page } from '@playwright/test';

export async function interceptCallbackRedirect(
	page: Page,
	callbackPath: string,
	action: () => Promise<void>,
	timeoutMs: number = 10000
): Promise<URL> {
	const routeMatcher = (url: URL) => url.pathname === callbackPath;

	const callbackPromise = new Promise<URL>((resolve, reject) => {
		const timer = setTimeout(
			() => reject(new Error(`Timed out waiting for redirect to ${callbackPath}`)),
			timeoutMs
		);

		page.route(routeMatcher, async (route) => {
			clearTimeout(timer);
			resolve(new URL(route.request().url()));
			await route.abort();
		});
	});

	try {
		await action();
		return await callbackPromise;
	} finally {
		await page.unroute(routeMatcher);
	}
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
