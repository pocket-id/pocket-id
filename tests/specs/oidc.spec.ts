import test, { expect } from '@playwright/test';
import { oidcClients, refreshTokens, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import { generateIdToken, generateOauthAccessToken } from '../utils/jwt.util';
import * as oidcUtil from '../utils/oidc.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => await cleanupBackend());

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

	await expect(page).toHaveURL('/login?redirect=%2F');
});

test('End session with id token hint redirects to callback URL', async ({ page }) => {
	const client = oidcClients.nextcloud;
	const idToken = await generateIdToken(users.tim, client.id);
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
	const { token, clientId, userId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	// Sign the refresh token
	const refreshToken = await request
		.post('/api/test/refreshtoken', {
			data: {
				rt: token,
				client: clientId,
				user: userId
			}
		})
		.then((r) => r.text());

	// Perform the exchange
	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: refreshToken,
			client_secret: clientSecret
		}
	});

	// Verify we got new tokens
	const tokenData = await refreshResponse.json();
	expect(tokenData.access_token).toBeDefined();
	expect(tokenData.refresh_token).toBeDefined();
	expect(tokenData.id_token).toBeDefined();
	expect(tokenData.token_type).toBe('Bearer');
	expect(tokenData.expires_in).toBe(3600);

	// The new refresh token should be different from the old one
	expect(tokenData.refresh_token).not.toBe(token);
});

test('Refresh token fails when used for the wrong client', async ({ request }) => {
	const { token, clientId, userId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	// Sign the refresh token
	const refreshToken = await request
		.post('/api/test/refreshtoken', {
			data: {
				rt: token,
				client: 'bad-client',
				user: userId
			}
		})
		.then((r) => r.text());

	// Perform the exchange
	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: refreshToken,
			client_secret: clientSecret
		}
	});

	expect(refreshResponse.status()).toBe(400);
});

test('Refresh token fails when used for the wrong user', async ({ request }) => {
	const { token, clientId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	// Sign the refresh token
	const refreshToken = await request
		.post('/api/test/refreshtoken', {
			data: {
				rt: token,
				client: clientId,
				user: '44cb5d71-db31-4555-9a1b-5484650f6002'
			}
		})
		.then((r) => r.text());

	// Perform the exchange
	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: refreshToken,
			client_secret: clientSecret
		}
	});

	expect(refreshResponse.status()).toBe(400);
});

test('Using refresh token invalidates it for future use', async ({ request }) => {
	const { token, clientId, userId } = refreshTokens.filter((token) => !token.expired)[0];
	const clientSecret = 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY';

	// Sign the refresh token
	const refreshToken = await request
		.post('/api/test/refreshtoken', {
			data: {
				rt: token,
				client: clientId,
				user: userId
			}
		})
		.then((r) => r.text());

	// Perform the exchange
	await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: refreshToken,
			client_secret: clientSecret
		}
	});

	// Try again
	const refreshResponse = await request.post('/api/oidc/token', {
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		form: {
			grant_type: 'refresh_token',
			client_id: clientId,
			refresh_token: refreshToken,
			client_secret: clientSecret
		}
	});
	expect(refreshResponse.status()).toBe(400);
});

test.describe('Introspection endpoint', () => {
	test('fails without client credentials', async ({ request }) => {
		const validAccessToken = await generateOauthAccessToken(users.tim, oidcClients.nextcloud.id);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				token: validAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(400);
	});

	test('succeeds with client credentials', async ({ request, baseURL }) => {
		const validAccessToken = await generateOauthAccessToken(users.tim, oidcClients.nextcloud.id);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization:
					'Basic ' +
					Buffer.from(`${oidcClients.nextcloud.id}:${oidcClients.nextcloud.secret}`).toString(
						'base64'
					)
			},
			form: {
				token: validAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(true);
		expect(introspectionBody.token_type).toBe('access_token');
		expect(introspectionBody.iss).toBe(baseURL);
		expect(introspectionBody.sub).toBe(users.tim.id);
		expect(introspectionBody.aud).toStrictEqual([oidcClients.nextcloud.id]);
	});

	test('succeeds with federated client credentials', async ({ page, request, baseURL }) => {
		const validAccessToken = await generateOauthAccessToken(users.tim, oidcClients.federated.id);
		const clientAssertion = await oidcUtil.getClientAssertion(
			page,
			oidcClients.federated.federatedJWT
		);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization: 'Bearer ' + clientAssertion
			},
			form: {
				token: validAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(true);
		expect(introspectionBody.token_type).toBe('access_token');
		expect(introspectionBody.iss).toBe(baseURL);
		expect(introspectionBody.sub).toBe(users.tim.id);
		expect(introspectionBody.aud).toStrictEqual([oidcClients.federated.id]);
	});

	test('fails with client credentials for wrong app', async ({ request }) => {
		const validAccessToken = await generateOauthAccessToken(users.tim, oidcClients.nextcloud.id);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization:
					'Basic ' +
					Buffer.from(`${oidcClients.immich.id}:${oidcClients.immich.secret}`).toString('base64')
			},
			form: {
				token: validAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(400);
	});

	test('fails with federated credentials for wrong app', async ({ page, request }) => {
		const validAccessToken = await generateOauthAccessToken(users.tim, oidcClients.nextcloud.id);
		const clientAssertion = await oidcUtil.getClientAssertion(
			page,
			oidcClients.federated.federatedJWT
		);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization: 'Bearer ' + clientAssertion
			},
			form: {
				token: validAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(400);
	});

	test('non-expired refresh_token can be verified', async ({ request }) => {
		const { token, clientId, userId } = refreshTokens.filter((token) => !token.expired)[0];

		// Sign the refresh token
		const refreshToken = await request
			.post('/api/test/refreshtoken', {
				data: {
					rt: token,
					client: clientId,
					user: userId
				}
			})
			.then((r) => r.text());

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization:
					'Basic ' +
					Buffer.from(`${oidcClients.nextcloud.id}:${oidcClients.nextcloud.secret}`).toString(
						'base64'
					)
			},
			form: {
				token: refreshToken
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(true);
		expect(introspectionBody.token_type).toBe('refresh_token');
	});

	test('expired refresh_token can be verified', async ({ request }) => {
		const { token, clientId, userId } = refreshTokens.filter((token) => token.expired)[0];

		// Sign the refresh token
		const refreshToken = await request
			.post('/api/test/refreshtoken', {
				data: {
					rt: token,
					client: clientId,
					user: userId
				}
			})
			.then((r) => r.text());

		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
				Authorization:
					'Basic ' +
					Buffer.from(`${oidcClients.nextcloud.id}:${oidcClients.nextcloud.secret}`).toString(
						'base64'
					)
			},
			form: {
				token: refreshToken
			}
		});

		expect(introspectionResponse.status()).toBe(200);
		const introspectionBody = await introspectionResponse.json();
		expect(introspectionBody.active).toBe(false);
	});

	test("expired access_token can't be verified", async ({ request }) => {
		const expiredAccessToken = await generateOauthAccessToken(
			users.tim,
			oidcClients.nextcloud.id,
			true
		);
		const introspectionResponse = await request.post('/api/oidc/introspect', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				token: expiredAccessToken
			}
		});

		expect(introspectionResponse.status()).toBe(400);
	});
});

test('Authorize new client with device authorization flow', async ({ page }) => {
	const client = oidcClients.immich;
	const userCode = await oidcUtil.getUserCode(page, client.id, client.secret);

	await page.goto(`/device?code=${userCode}`);

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(
		page.getByRole('paragraph').filter({ hasText: 'The device has been authorized.' })
	).toBeVisible();
});

test('Authorize new client with device authorization flow while not signed in', async ({
	page
}) => {
	await page.context().clearCookies();
	const client = oidcClients.immich;
	const userCode = await oidcUtil.getUserCode(page, client.id, client.secret);

	await page.goto(`/device?code=${userCode}`);

	await (await passkeyUtil.init(page)).addPasskey();
	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(
		page.getByRole('paragraph').filter({ hasText: 'The device has been authorized.' })
	).toBeVisible();
});

test('Authorize existing client with device authorization flow', async ({ page }) => {
	const client = oidcClients.nextcloud;
	const userCode = await oidcUtil.getUserCode(page, client.id, client.secret);

	await page.goto(`/device?code=${userCode}`);

	await expect(
		page.getByRole('paragraph').filter({ hasText: 'The device has been authorized.' })
	).toBeVisible();
});

test('Authorize existing client with device authorization flow while not signed in', async ({
	page
}) => {
	await page.context().clearCookies();
	const client = oidcClients.nextcloud;
	const userCode = await oidcUtil.getUserCode(page, client.id, client.secret);

	await page.goto(`/device?code=${userCode}`);

	await (await passkeyUtil.init(page)).addPasskey();
	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(
		page.getByRole('paragraph').filter({ hasText: 'The device has been authorized.' })
	).toBeVisible();
});

test('Authorize client with device authorization flow with invalid code', async ({ page }) => {
	await page.goto('/device?code=invalid-code');

	await expect(
		page.getByRole('paragraph').filter({ hasText: 'Invalid device code.' })
	).toBeVisible();
});

test('Authorize new client with device authorization with user group not allowed', async ({
	page
}) => {
	await page.context().clearCookies();
	const client = oidcClients.immich;
	const userCode = await oidcUtil.getUserCode(page, client.id, client.secret);

	await page.goto(`/device?code=${userCode}`);

	await (await passkeyUtil.init(page)).addPasskey('craig');
	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await page.getByRole('button', { name: 'Authorize' }).click();

	await expect(
		page.getByRole('paragraph').filter({ hasText: "You're not allowed to access this service." })
	).toBeVisible();
});

test('Federated identity fails with invalid client assertion', async ({ page }) => {
	const client = oidcClients.federated;

	const res = await oidcUtil.exchangeCode(page, {
		client_assertion_type: 'urn:ietf:params:oauth:client-assertion-type:jwt-bearer',
		grant_type: 'authorization_code',
		redirect_uri: client.callbackUrl,
		code: client.accessCodes[0],
		client_id: client.id,
		client_assertion: 'not-an-assertion'
	});

	expect(res?.error).toBe('Invalid client assertion');
});

test('Authorize existing client with federated identity', async ({ page }) => {
	const client = oidcClients.federated;
	const clientAssertion = await oidcUtil.getClientAssertion(page, client.federatedJWT);

	const res = await oidcUtil.exchangeCode(page, {
		client_assertion_type: 'urn:ietf:params:oauth:client-assertion-type:jwt-bearer',
		grant_type: 'authorization_code',
		redirect_uri: client.callbackUrl,
		code: client.accessCodes[0],
		client_id: client.id,
		client_assertion: clientAssertion
	});

	expect(res.access_token).not.toBeNull;
	expect(res.expires_in).not.toBeNull;
	expect(res.token_type).toBe('Bearer');
});

test('Forces reauthentication when client requires it', async ({ page, request }) => {
	let webauthnStartCalled = false;
	await page.route('/api/webauthn/login/start', async (route) => {
		webauthnStartCalled = true;
		await route.continue();
	});

	await request.put(`/api/oidc/clients/${oidcClients.nextcloud.id}`, {
		data: { ...oidcClients.nextcloud, requiresReauthentication: true }
	});

	await (await passkeyUtil.init(page)).addPasskey();

	const urlParams = createUrlParams(oidcClients.nextcloud);
	await page.goto(`/authorize?${urlParams.toString()}`);

	await expect(page.getByTestId('scopes')).not.toBeVisible();

	await page.waitForURL(oidcClients.nextcloud.callbackUrl).catch((e) => {
		if (!e.message.includes('net::ERR_NAME_NOT_RESOLVED')) {
			throw e;
		}
	});

	expect(webauthnStartCalled).toBe(true);
});
