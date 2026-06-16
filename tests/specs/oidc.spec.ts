import test, { expect, type Page, type Request } from '@playwright/test';
import { oidcClients, refreshTokens, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import { generateIdToken, generateOauthAccessToken } from '../utils/jwt.util';
import * as oidcUtil from '../utils/oidc.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => await cleanupBackend());

test('Authorize existing client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
		page.goto(`/authorize?${urlParams.toString()}`)
	);
});

test('Authorize existing client while not signed in', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	await page.context().clearCookies();

	await expectCallbackRedirect(page, oidcClient.callbackUrl, async () => {
		await page.goto(`/authorize?${urlParams.toString()}`);

		await (await passkeyUtil.init(page)).addPasskey();
		await page.getByRole('button', { name: 'Sign in' }).click();
	});
});

test('Authorize new client', async ({ page }) => {
	const oidcClient = oidcClients.immich;
	const urlParams = createUrlParams(oidcClient);
	await page.goto(`/authorize?${urlParams.toString()}`);

	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();
	await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })).toBeVisible();

	await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
		page.getByRole('button', { name: 'Sign in' }).click()
	);
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

	await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
		page.getByRole('button', { name: 'Sign in' }).click()
	);
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
	const idToken = await generateIdToken(
		'fe81c12a-7336-4aee-bebc-d901a873bf48',
		users.tim,
		client.id
	);
	await expectCallbackRedirect(page, client.logoutCallbackUrl, () =>
		page.goto(
			`/api/oidc/end-session?id_token_hint=${idToken}&post_logout_redirect_uri=${client.logoutCallbackUrl}`
		)
	);
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
				client_id: oidcClients.federated.id,
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
				client_id: oidcClients.federated.id,
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
	await expectCallbackRedirect(page, oidcClients.nextcloud.callbackUrl, async () => {
		await page.goto(`/authorize?${urlParams.toString()}`);
		await expect(page.getByTestId('scopes')).not.toBeVisible();
	});

	expect(webauthnStartCalled).toBe(true);
});

test('Authorize existing client while not signed in with response_mode=form_post', async ({
	page
}) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	urlParams.set('response_mode', 'form_post');
	await page.context().clearCookies();

	const formPostRequestPromise = waitForFormPostRequest(page, oidcClient.callbackUrl);
	await page.goto(`/authorize?${urlParams.toString()}`);

	await (await passkeyUtil.init(page)).addPasskey();
	await page.getByRole('button', { name: 'Sign in' }).click();

	await expectFormPostRequest(formPostRequestPromise);
});

test('Authorize existing client with response_mode=form_post', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	urlParams.set('response_mode', 'form_post');

	const formPostRequestPromise = waitForFormPostRequest(page, oidcClient.callbackUrl);
	await page.goto(`/authorize?${urlParams.toString()}`);

	await expectFormPostRequest(formPostRequestPromise);
});

test('Authorize existing client with response_mode=fragment', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	const urlParams = createUrlParams(oidcClient);
	urlParams.set('response_mode', 'fragment');

	const redirectUrl = await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
		page.goto(`/authorize?${urlParams.toString()}`)
	);
	expect(redirectUrl.search).toBe('');

	const fragmentParams = new URLSearchParams(redirectUrl.hash.slice(1));
	expect(fragmentParams.get('code')).toBeTruthy();
	expect(fragmentParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	expect(fragmentParams.get('iss')).toBeTruthy();
});

function waitForFormPostRequest(page: Page, callbackUrl: string): Promise<Request> {
	return page.waitForRequest(
		(request) => request.method() === 'POST' && request.url() === callbackUrl
	);
}

async function expectFormPostRequest(formPostRequestPromise: Promise<Request>) {
	const request = await formPostRequestPromise;
	const formData = new URLSearchParams(request.postData() ?? '');

	expect(formData.get('code')).toBeTruthy();
	expect(formData.get('state')).toBe('nXx-6Qr-owc1SHBa');
	expect(formData.get('iss')).toBeTruthy();
}

test.describe('OIDC prompt parameter', () => {
	test('prompt=none redirects with login_required when user not authenticated', async ({
		page
	}) => {
		await page.context().clearCookies();
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');

		// Should redirect to callback URL with error
		const redirectUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', () =>
			page.goto(`/authorize?${urlParams.toString()}`).then(() => {})
		);

		expect(redirectUrl.searchParams.get('error')).toBe('login_required');
		expect(redirectUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});

	test('prompt=none does not redirect login_required to an unregistered redirect_uri', async ({
		page
	}) => {
		await page.context().clearCookies();
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');
		urlParams.set('redirect_uri', 'https://attacker.example/collect');

		let attackerRedirected = false;
		await page.route('https://attacker.example/**', async (route) => {
			attackerRedirected = true;
			await route.fulfill({ status: 200, body: 'attacker' });
		});

		await page.goto(`/authorize?${urlParams.toString()}`);

		await expect(page.getByText(/Invalid callback URL/i)).toBeVisible();
		expect(attackerRedirected).toBe(false);
		expect(page.url()).not.toContain('attacker.example');
	});

	test('prompt=none redirects errors with response_mode=fragment', async ({ page }) => {
		await page.context().clearCookies();
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');
		urlParams.set('response_mode', 'fragment');

		const redirectUrl = await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.goto(`/authorize?${urlParams.toString()}`)
		);
		expect(redirectUrl.search).toBe('');

		const fragmentParams = new URLSearchParams(redirectUrl.hash.slice(1));
		expect(fragmentParams.get('error')).toBe('login_required');
		expect(fragmentParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});

	test('prompt=none redirects with consent_required when authorization needed', async ({
		page
	}) => {
		const oidcClient = oidcClients.immich;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');

		// Should redirect to callback URL with error
		const redirectUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', () =>
			page.goto(`/authorize?${urlParams.toString()}`).then(() => {})
		);

		expect(redirectUrl.searchParams.get('error')).toBe('consent_required');
		expect(redirectUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});

	test('prompt=none does not redirect consent_required to an unregistered redirect_uri', async ({
		page
	}) => {
		const oidcClient = oidcClients.immich;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');
		urlParams.set('redirect_uri', 'https://attacker.example/collect');

		let attackerRedirected = false;
		await page.route('https://attacker.example/**', async (route) => {
			attackerRedirected = true;
			await route.fulfill({ status: 200, body: 'attacker' });
		});

		await page.goto(`/authorize?${urlParams.toString()}`);

		await expect(page.getByText(/Invalid callback URL/i)).toBeVisible();
		expect(attackerRedirected).toBe(false);
	});

	test('prompt=none succeeds when user is authenticated and authorized', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none');

		await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.goto(`/authorize?${urlParams.toString()}`)
		);
	});

	test('prompt=consent forces consent display even for authorized client', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'consent');

		await page.goto(`/authorize?${urlParams.toString()}`);

		// Should show consent UI even though client was already authorized
		await expect(
			page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })
		).toBeVisible();
		await expect(page.getByTestId('scopes').getByRole('heading', { name: 'Email' })).toBeVisible();

		await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.getByRole('button', { name: 'Sign in' }).click()
		);
	});

	test('prompt=login forces reauthentication', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'login');

		let reauthCalled = false;
		await page.route('/api/webauthn/login/start', async (route) => {
			reauthCalled = true;
			await route.continue();
		});

		await (await passkeyUtil.init(page)).addPasskey();
		await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.goto(`/authorize?${urlParams.toString()}`)
		);

		expect(reauthCalled).toBe(true);
	});

	test('prompt=select_account shows current user and continues on confirm', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'select_account');

		await page.goto(`/authorize?${urlParams.toString()}`);

		// Account selection card with the signed-in user should appear
		const selectionCard = page.getByTestId('account-selection');
		await expect(selectionCard).toBeVisible();
		await expect(selectionCard).toContainText('Tim Cook');

		await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.getByRole('button', { name: 'Sign In' }).click()
		);
	});

	test('prompt=select_account account can be changed', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'select_account');

		await page.goto(`/authorize?${urlParams.toString()}`);

		await page.getByRole('button', { name: 'Use a different account' }).click();
		await expect(page.getByText('Do you want to sign in to Nextcloud')).toBeVisible();

		(await passkeyUtil.init(page)).addPasskey('craig');

		await page.getByRole('button', { name: 'Sign In' }).click();

		await expectCallbackRedirect(page, oidcClient.callbackUrl, () =>
			page.getByRole('button', { name: 'Sign In' }).click()
		);
	});

	test('prompt=none with prompt=consent returns interaction_required', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none consent');

		// Should redirect with error since both can't be satisfied
		const redirectUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', () =>
			page.goto(`/authorize?${urlParams.toString()}`).then(() => {})
		);

		expect(redirectUrl.searchParams.get('error')).toBe('interaction_required');
		expect(redirectUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});

	test('prompt=none with prompt=login returns interaction_required', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none login');

		// Should redirect with error since both can't be satisfied
		const redirectUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', () =>
			page.goto(`/authorize?${urlParams.toString()}`).then(() => {})
		);

		expect(redirectUrl.searchParams.get('error')).toBe('interaction_required');
		expect(redirectUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});

	test('prompt=none with prompt=select_account returns interaction_required', async ({ page }) => {
		const oidcClient = oidcClients.nextcloud;
		const urlParams = createUrlParams(oidcClient);
		urlParams.set('prompt', 'none select_account');

		// Should redirect with error since both can't be satisfied
		const redirectUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', () =>
			page.goto(`/authorize?${urlParams.toString()}`).then(() => {})
		);

		expect(redirectUrl.searchParams.get('error')).toBe('interaction_required');
		expect(redirectUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
	});
});

async function waitForCallbackURL(page: Page, callbackUrl: string): Promise<URL> {
	const expectedUrl = new URL(callbackUrl);
	const isCallbackURL = (url: URL) =>
		url.origin === expectedUrl.origin && url.pathname === expectedUrl.pathname;

	const callbackRequest = await page.waitForRequest((request) =>
		isCallbackURL(new URL(request.url()))
	);
	await page.waitForURL(isCallbackURL, { waitUntil: 'commit' }).catch(() => {});

	const currentURL = new URL(page.url());
	if (isCallbackURL(currentURL)) {
		return currentURL;
	}

	return new URL(callbackRequest.url());
}

async function expectCallbackRedirect(
	page: Page,
	callbackUrl: string,
	action: () => Promise<unknown>
): Promise<URL> {
	const callbackRouteMatcher = await routeCallbackPage(page, callbackUrl);

	try {
		const callbackURLPromise = waitForCallbackURL(page, callbackUrl);
		const actionPromise = action().then(
			() => undefined,
			(error) => error
		);
		const callbackURL = await callbackURLPromise;
		await actionPromise;
		return callbackURL;
	} finally {
		await page.unroute(callbackRouteMatcher);
	}
}

async function routeCallbackPage(page: Page, callbackUrl: string): Promise<(url: URL) => boolean> {
	const expectedUrl = new URL(callbackUrl);
	const callbackRouteMatcher = (url: URL) =>
		url.origin === expectedUrl.origin && url.pathname === expectedUrl.pathname;

	await page.route(callbackRouteMatcher, async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'text/html',
			body: '<!doctype html><title>OIDC callback</title>'
		});
	});

	return callbackRouteMatcher;
}

// ─── PAR (Pushed Authorization Requests - RFC 9126) ──────────────────────────

test.describe('Pushed Authorization Requests (PAR)', () => {
	const client = oidcClients.parClient;

	test('PAR endpoint returns request_uri for valid confidential client', async ({ page }) => {
		const result = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl
		});

		expect(result.request_uri).toMatch(/^urn:ietf:params:oauth:request_uri:/);
		expect(result.expires_in).toBe(90);
		expect(result.error).toBeUndefined();
	});

	test('PAR full flow: push then authorize then exchange tokens', async ({ page }) => {
		// Step 1: Push authorization parameters
		const parResult = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl,
			nonce: 'par-nonce-123'
		});
		expect(parResult.request_uri).toBeDefined();
		expect(parResult.error).toBeUndefined();

		// Step 2: Navigate to /authorize using the request_uri
		const urlParams = new URLSearchParams({
			client_id: client.id,
			request_uri: parResult.request_uri!
		});

		const callbackUrl = await expectCallbackRedirect(page, client.callbackUrl, () =>
			page.goto(`/authorize?${urlParams.toString()}`)
		);
		const code = callbackUrl.searchParams.get('code');
		expect(code).toBeTruthy();

		// Step 3: Exchange the authorization code for tokens
		const tokenResult = await oidcUtil.exchangeCode(page, {
			grant_type: 'authorization_code',
			code: code!,
			client_id: client.id,
			client_secret: client.secret,
			redirect_uri: client.callbackUrl
		});
		expect(tokenResult.access_token).toBeTruthy();
		expect(tokenResult.token_type).toBe('Bearer');
		expect(tokenResult.error).toBeUndefined();
	});

	test('par-request-info resolves the stored request parameters', async ({ page }) => {
		const state = 'par-info-state-9f3a';
		const parResult = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl,
			scope: 'openid profile',
			state,
			responseMode: 'form_post'
		});
		expect(parResult.request_uri).toBeDefined();

		const res = await page.request.get('/api/oidc/par-request-info', {
			params: { client_id: client.id, request_uri: parResult.request_uri! }
		});
		expect(res.ok()).toBe(true);

		const body = await res.json();
		expect(body.scope).toBe('openid profile');
		expect(body.redirectURI).toBe(client.callbackUrl);
		expect(body.state).toBe(state);
		expect(body.responseMode).toBe('form_post');
	});

	test('PAR full flow carries the resolved state into the callback redirect', async ({ page }) => {
		const state = 'par-flow-state-7b21';
		const parResult = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl,
			state,
			responseMode: 'form_post'
		});
		expect(parResult.request_uri).toBeDefined();

		const urlParams = new URLSearchParams({
			client_id: client.id,
			request_uri: parResult.request_uri!
		});

		const formPostRequestPromise = waitForFormPostRequest(page, client.callbackUrl);
		await page.goto(`/authorize?${urlParams.toString()}`);

		const request = await formPostRequestPromise;
		const formData = new URLSearchParams(request.postData() ?? '');
		expect(formData.get('code')).toBeTruthy();
		expect(formData.get('state')).toBe(state);
	});

	test('PAR full flow shows consent screen when authorization is required', async ({ page }) => {
		// The parClient is pre-authorized for "openid profile email"; pushing a different
		// scope means consent is required and the consent screen must be shown rather than
		// silently authorizing.
		const parResult = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl,
			scope: 'openid profile'
		});
		expect(parResult.request_uri).toBeDefined();

		const urlParams = new URLSearchParams({
			client_id: client.id,
			request_uri: parResult.request_uri!
		});
		await page.goto(`/authorize?${urlParams.toString()}`);

		// Consent screen with the requested scope (resolved from the PAR) must be shown
		await expect(
			page.getByTestId('scopes').getByRole('heading', { name: 'Profile' })
		).toBeVisible();

		// Confirming proceeds with the authorization
		await expectCallbackRedirect(page, client.callbackUrl, () =>
			page.getByRole('button', { name: 'Sign in' }).click()
		);
	});

	test('PAR request_uri is single-use', async ({ page }) => {
		// Push two requests — use the first via the browser, then try to reuse it
		const parResult = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl
		});
		expect(parResult.request_uri).toBeDefined();

		// First use — navigate to /authorize (must succeed and consume the request_uri)
		const urlParams = new URLSearchParams({
			client_id: client.id,
			request_uri: parResult.request_uri!
		});
		const firstCallbackUrl = await expectCallbackRedirect(page, client.callbackUrl, () =>
			page.goto(`/authorize?${urlParams.toString()}`)
		);
		expect(firstCallbackUrl.searchParams.get('code')).toBeTruthy();

		// Second use of the same request_uri should fail
		// Use the authorize API directly (requires auth cookie which we have)
		const response = await page.request.post('/api/oidc/authorize', {
			headers: { 'Content-Type': 'application/json' },
			data: {
				clientID: client.id,
				requestURI: parResult.request_uri
			}
		});
		expect(response.status()).toBe(400);
	});

	test('PAR endpoint rejects request without client credentials', async ({ page }) => {
		const result = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			// no clientSecret
			redirectUri: client.callbackUrl
		});

		expect(result.error).toBeDefined();
		expect(result.request_uri).toBeUndefined();
	});

	test('PAR endpoint rejects public client', async ({ page }) => {
		// The parClient is confidential — test by setting isPublic via admin API first
		await page.request.put(`/api/oidc/clients/${client.id}`, {
			headers: { 'Content-Type': 'application/json' },
			data: {
				name: client.name,
				callbackURLs: [client.callbackUrl],
				logoutCallbackURLs: [],
				isPublic: true,
				pkceEnabled: true,
				requiresReauthentication: false,
				requiresPushedAuthorizationRequests: false,
				credentials: { federatedIdentities: [] },
				isGroupRestricted: false
			}
		});

		const result = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: client.callbackUrl
		});

		expect(result.error).toBe('Pushed authorization requests are not supported for public clients');
		expect(result.request_uri).toBeUndefined();
	});

	test('PAR endpoint rejects invalid redirect_uri at push time', async ({ page }) => {
		const result = await oidcUtil.pushAuthorizationRequest(page, {
			clientId: client.id,
			clientSecret: client.secret,
			redirectUri: 'http://evil.example.com/steal'
		});

		expect(result.error).toBeDefined();
		expect(result.request_uri).toBeUndefined();
	});

	test('Client with requiresPushedAuthorizationRequests rejects direct /authorize', async ({
		page,
		request
	}) => {
		// Enable the PAR requirement on the client
		await request.put(`/api/oidc/clients/${client.id}`, {
			headers: { 'Content-Type': 'application/json' },
			data: {
				name: client.name,
				callbackURLs: [client.callbackUrl],
				logoutCallbackURLs: [],
				isPublic: false,
				pkceEnabled: false,
				requiresReauthentication: false,
				requiresPushedAuthorizationRequests: true,
				credentials: { federatedIdentities: [] },
				isGroupRestricted: false
			}
		});

		// Attempt a normal authorization (without request_uri)
		const response = await page.request.post('/api/oidc/authorize', {
			headers: { 'Content-Type': 'application/json' },
			data: {
				clientID: client.id,
				scope: 'openid profile',
				callbackURL: client.callbackUrl
			}
		});
		expect(response.status()).toBe(400);
	});

	test('Admin UI: PAR toggle persists after save', async ({ page }) => {
		await page.goto(`/settings/admin/oidc-clients/${client.id}`);

		await page.getByRole('button', { name: 'Show Advanced Options' }).click();

		// Enable the PAR toggle
		const parToggle = page.getByRole('switch', { name: 'Requires Pushed Authorization' });
		if (!(await parToggle.isChecked())) {
			await parToggle.click();
		}

		await page.getByRole('button', { name: /save/i }).click();
		await page.reload();

		await page.getByRole('button', { name: 'Show Advanced Options' }).click();
		const savedToggle = page.getByRole('switch', { name: 'Requires Pushed Authorization' });
		await expect(savedToggle).toBeChecked();
	});
});
