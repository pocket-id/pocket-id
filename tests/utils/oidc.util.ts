import type { Page } from '@playwright/test';

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
