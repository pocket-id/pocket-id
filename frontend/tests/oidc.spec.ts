import test, {type APIRequestContext, expect} from '@playwright/test';
import {oidcClients, refreshTokens, users} from './data';
import { cleanupBackend } from './utils/cleanup.util';
import passkeyUtil from './utils/passkey.util';

test.beforeEach(cleanupBackend);

test('Authorize existing client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	await page.goto(`/authorize?${urlParams.toString()}`);

	// Ignore DNS resolution error as the callback URL is not reachable
	await page.waitForURL(oidcClient.callbackUrl).catch((e) => {
		if (!e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
			throw e;
		}
	});
});

test('Authorize existing client while not signed in', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	await page.context().clearCookies();
	await page.goto(`/authorize?${urlParams.toString()}`);

	await (await passkeyUtil.init(page)).addPasskey();
	await page.getByRole('button', { name: 'Sign in' }).click();

	// Ignore DNS resolution error as the callback URL is not reachable
	await page.waitForURL(oidcClient.callbackUrl).catch((e) => {
		if (!e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
			throw e;
		}
	});
});

test('Authorize new client', async ({ page }) => {
	const oidcClient = oidcClients.immich;
	const urlParams = createUrlParams(oidcClient);
	await page.goto(`/authorize?${urlParams.toString()}`);

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Sign in' }).click();

	// Ignore DNS resolution error as the callback URL is not reachable
	await page.waitForURL(oidcClient.callbackUrl).catch((e) => {
		if (!e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
			throw e;
		}
	});
});

test('Authorize new client while not signed in', async ({ page }) => {
	const oidcClient = oidcClients.immich;
	const urlParams = createUrlParams(oidcClient);
	await page.context().clearCookies();
	await page.goto(`/authorize?${urlParams.toString()}`);

	await (await passkeyUtil.init(page)).addPasskey();
	await page.getByRole('button', { name: 'Sign in' }).click();

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Sign in' }).click();

	// Ignore DNS resolution error as the callback URL is not reachable
	await page.waitForURL(oidcClient.callbackUrl).catch((e) => {
		if (!e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
			throw e;
		}
	});
});

test('Authorize new client fails with user group not allowed', async ({ page }) => {
	const oidcClient = oidcClients.immich;
	const urlParams = createUrlParams(oidcClient);
	await page.context().clearCookies();
	await page.goto(`/authorize?${urlParams.toString()}`);

	await (await passkeyUtil.init(page)).addPasskey('craig');
	await page.getByRole('button', { name: 'Sign in' }).click();

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Sign in' }).click();

	await expect(page.getByRole('paragraph').first()).toHaveText(
		"You're not allowed to access this service."
	);
});

function createUrlParams(oidcClient: { id: string; callbackUrl: string }) {
	return new URLSearchParams({
		client_id: oidcClient.id,
		response_type: 'code',
		scope: 'openid profile email',
		redirect_uri: oidcClient.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa',
		nonce: 'P1gN3PtpKHJgKUVcLpLjm'
	});
}

test('End session without id token hint shows confirmation page', async ({ page }) => {
	await page.goto('/api/oidc/end-session');

	await expect(page).toHaveURL('/logout');
	await page.getByRole('button', { name: 'Sign out' }).click();

	await expect(page).toHaveURL('/login');
});

test('End session with id token hint redirects to callback URL', async ({ page }) => {
	const client = oidcClients.nextcloud;
	// Note: this token has expired, but it should be accepted by the logout endpoint anyways, per spec
	const idToken =
		'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiSldUIn0.eyJhdWQiOiIzNjU0YTc0Ni0zNWQ0LTQzMjEtYWM2MS0wYmRjZmYyYjQwNTUiLCJlbWFpbCI6InRpbS5jb29rQHRlc3QuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImV4cCI6MTY5MDAwMDAwMSwiZmFtaWx5X25hbWUiOiJUaW0iLCJnaXZlbl9uYW1lIjoiQ29vayIsImlhdCI6MTY5MDAwMDAwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsIm5hbWUiOiJUaW0gQ29vayIsIm5vbmNlIjoib1cxQTFPNzhHUTE1RDczT3NIRXg3V1FLajdacXZITFp1XzM3bWRYSXFBUSIsInN1YiI6IjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIn0.ruYCyjA2BNjROpmLGPNHrhgUNLnpJMEuncvjDYVuv1dAZwvOPfG-Rn-OseAgJDJbV7wJ0qf6ZmBkGWiifwc_B9h--fgd4Vby9fefj0MiHbSDgQyaU5UmpvJU8OlvM-TueD6ICJL0NeT3DwoW5xpIWaHtt3JqJIdP__Q-lTONL2Zokq50kWm0IO-bIw2QrQviSfHNpv8A5rk1RTzpXCPXYNB-eJbm3oBqYQWzerD9HaNrSvrKA7mKG8Te1mI9aMirPpG9FvcAU-I3lY8ky1hJZDu42jHpVEUdWPAmUZPZafoX8iYtlPfkoklDnHj_cdg4aZBGN5bfjM6xf1Oe_rLDWg';

	let redirectedCorrectly = false;
	await page
		.goto(
			`/api/oidc/end-session?id_token_hint=${idToken}&post_logout_redirect_uri=${client.logoutCallbackUrl}`
		)
		.catch((e) => {
			if (e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
				redirectedCorrectly = true;
			} else {
				throw e;
			}
		});

	expect(redirectedCorrectly).toBeTruthy();
});

test('Successfully refresh tokens with valid refresh token', async ({ request }) => {
	const { token, clientId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: token,
			client_secret: clientSecret
		}
	});

	// Verify we got new tokens
	const tokenData = await refreshResponse.json();
	expect(tokenData.access_token).toBeDefined();
	expect(tokenData.refresh_token).toBeDefined();
	expect(tokenData.token_type).toBe('Bearer');
	expect(tokenData.expires_in).toBe(3600);

	// The new refresh token should be different from the old one
	expect(tokenData.refresh_token).not.toBe(token);
});

test('Using refresh token invalidates it for future use', async ({ request }) => {
	const { token, clientId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: token,
			client_secret: clientSecret
		}
	});

	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: token,
			client_secret: clientSecret
		}
	});
	expect(refreshResponse.status()).toBe(400);
});

test.describe('Introspection endpoint', () => {
	async function getAccessToken(request: APIRequestContext) {
		const { token } = refreshTokens.filter((token) => !token.expired)[0];
		const accessTokenResponse = await request.post('/api/oidc/token', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				refresh_token: token,
				grant_type: 'refresh_token',
				client_id: '3654a746-35d4-4321-ac61-0bdcff2b4055',
				client_secret: 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY'
			}
		});
		const body = await accessTokenResponse.json();
		return body.access_token as string;
	}

	test('without client_id and client_secret fails', async ({ request }) => {
		const access_token = await getAccessToken(request);

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				token: access_token
			}
		});

		expect(introspectionResponse.status()).toBe(400);
	});

	test('with client_id and client_secret succeeds', async ({ request }) => {
		const clientId = '3654a746-35d4-4321-ac61-0bdcff2b4055';
		const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';
		const access_token = await getAccessToken(request);

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				'Authorization': 'Basic ' + Buffer.from(`${clientId}:${clientSecret}`).toString('base64'),
			},
			form: {
				token: access_token
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(true);
		expect(introspectionBody.token_type).toBe("access_token");
		expect(introspectionBody.iss).toBe("http://localhost");
		expect(introspectionBody.sub).toBe(users.tim.id);
		expect(introspectionBody.aud).toStrictEqual([oidcClients.nextcloud.id]);
	});

	test('non-expired refresh_token can be verified', async ({ request }) => {
		const clientId = '3654a746-35d4-4321-ac61-0bdcff2b4055';
		const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';
		const { token } = refreshTokens.filter((token) => !token.expired)[0];

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				'Authorization': 'Basic ' + Buffer.from(`${clientId}:${clientSecret}`).toString('base64'),
			},
			form: {
				token: token,
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(true);
		expect(introspectionBody.token_type).toBe("refresh_token");
	});

	test('expired refresh_token can be verified', async ({ request }) => {
		const clientId = '3654a746-35d4-4321-ac61-0bdcff2b4055';
		const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';
		const { token } = refreshTokens.filter((token) => token.expired)[0];

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				'Authorization': 'Basic ' + Buffer.from(`${clientId}:${clientSecret}`).toString('base64'),
			},
			form: {
				token: token,
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(false);
		expect(introspectionBody.token_type).toBe("refresh_token");
	});
})


