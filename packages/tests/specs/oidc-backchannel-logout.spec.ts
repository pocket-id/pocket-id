import base, { expect, type APIRequestContext, type BrowserContext } from '@playwright/test';
import { createLocalJWKSet, decodeJwt, jwtVerify } from 'jose';
import { createServer } from 'node:http';
import { oidcClients, userGroups, users } from '../data';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { exchangeCode, interceptCallbackRedirect } from '../utils/oidc.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

type Delivery = {
	path: string;
	method: string;
	contentType: string;
	body: URLSearchParams;
};

type LogoutReceiver = {
	url: string;
	deliveries: Delivery[];
	respond: (path: string, attempt: number) => number;
};

const test = base.extend<{ receiver: LogoutReceiver }>({
	receiver: async ({ baseURL }, use) => {
		const receiverURL = new URL(oidcClients.nextcloud.backchannelLogoutURL);
		const receiver: LogoutReceiver = {
			url: receiverURL.origin,
			deliveries: [],
			respond: () => 204
		};
		const server = createServer(async (request, response) => {
			const chunks: Buffer[] = [];
			for await (const chunk of request) chunks.push(Buffer.from(chunk));
			const body = new URLSearchParams(Buffer.concat(chunks).toString());
			// In CI the Pocket ID instances of the other shards send their logouts to this same host port
			const logoutToken = body.get('logout_token');
			if (logoutToken && decodeJwt(logoutToken).iss !== baseURL) {
				response.writeHead(204).end();
				return;
			}
			const path = request.url!;
			receiver.deliveries.push({
				path,
				method: request.method!,
				contentType: request.headers['content-type'] ?? '',
				body
			});
			const attempt = receiver.deliveries.filter((delivery) => delivery.path === path).length;
			response.writeHead(receiver.respond(path, attempt)).end();
		});

		// Docker reaches this real RP endpoint through the host gateway on both local machines and CI
		await new Promise<void>((resolve, reject) => {
			server.once('error', reject);
			server.listen(Number(receiverURL.port), '0.0.0.0', resolve);
		});
		try {
			await use(receiver);
		} finally {
			server.closeAllConnections();
			await new Promise<void>((resolve, reject) => {
				server.close((error) => (error ? reject(error) : resolve()));
			});
		}
	}
});

test.beforeEach(async () => cleanupBackend({ skipLdapSetup: true }));

async function revoke(request: APIRequestContext, clientId: string) {
	const response = await request.delete(`/api/oidc/users/me/authorized-clients/${clientId}`);
	expect(response.ok()).toBeTruthy();
}

type App = { id: string; callbackUrl: string; secret: string };

// Signs in to the app and returns its tokens, consenting on the way for scopes the user hasn't granted yet
// Each sign-in gets its own page, because the redirect to the unreachable app host ends in a late error navigation
async function signInToApp(context: BrowserContext, app: App, scope: string, consent = false) {
	const page = await context.newPage();
	const params = new URLSearchParams({
		client_id: app.id,
		response_type: 'code',
		scope,
		redirect_uri: app.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});
	const authorize = () => page.goto(`/authorize?${params.toString()}`);
	const continueButton = page.getByRole('button', { name: /^Continue to / });
	if (consent) {
		await authorize();
		await expect(continueButton).toBeVisible();
	}
	const callbackUrl = await interceptCallbackRedirect(page, '/auth/callback', () =>
		consent ? continueButton.click() : authorize()
	);
	const tokens = await exchangeCode(page, {
		grant_type: 'authorization_code',
		code: callbackUrl.searchParams.get('code')!,
		redirect_uri: app.callbackUrl,
		client_id: app.id,
		client_secret: app.secret
	});
	await page.close();
	return tokens as { id_token: string; refresh_token: string };
}

async function refresh(request: APIRequestContext, app: App, refreshToken: string) {
	return request.post('/api/oidc/token', {
		form: {
			grant_type: 'refresh_token',
			client_id: app.id,
			client_secret: app.secret,
			refresh_token: refreshToken
		}
	});
}

function deliveryFor(receiver: LogoutReceiver, app: { backchannelLogoutURL: string }) {
	return receiver.deliveries.find(
		(delivery) => delivery.path === new URL(app.backchannelLogoutURL).pathname
	)!;
}

async function verifyLogoutToken(
	request: APIRequestContext,
	delivery: Delivery,
	clientId: string,
	userId: string,
	sessionId?: string
) {
	expect(delivery.method).toBe('POST');
	expect(delivery.contentType).toBe('application/x-www-form-urlencoded');
	expect(delivery.body.getAll('logout_token')).toHaveLength(1);
	const discoveryResponse = await request.get('/.well-known/openid-configuration');
	expect(discoveryResponse.ok()).toBeTruthy();
	const discovery = await discoveryResponse.json();
	expect(discovery.backchannel_logout_supported).toBe(true);
	expect(discovery.backchannel_logout_session_supported).toBe(true);
	const jwksResponse = await request.get(discovery.jwks_uri);
	expect(jwksResponse.ok()).toBeTruthy();
	const { payload, protectedHeader } = await jwtVerify(
		delivery.body.get('logout_token')!,
		createLocalJWKSet(await jwksResponse.json()),
		{
			issuer: discovery.issuer,
			audience: clientId,
			algorithms: discovery.id_token_signing_alg_values_supported,
			typ: 'logout+jwt',
			maxTokenAge: '2m',
			requiredClaims: ['iss', 'sub', 'aud', 'iat', 'exp', 'jti', 'events']
		}
	);
	expect(protectedHeader.kid).toBeTruthy();
	expect(payload.sub).toBe(userId);
	expect(payload.aud).toEqual([clientId]);
	expect(payload.events).toEqual({ 'http://schemas.openid.net/event/backchannel-logout': {} });
	expect(payload.exp! - payload.iat!).toBe(120);
	expect(payload.jti).toBeTruthy();
	expect(payload).not.toHaveProperty('nonce');
	if (sessionId) expect(payload.sid).toBe(sessionId);
	else expect(payload).not.toHaveProperty('sid');
	return payload;
}

test('Signing out of Pocket ID logs the apps of that session out', async ({
	browser,
	request,
	receiver
}) => {
	const other = await authUtil.signInElsewhere(browser, 'tim');
	const nextcloud = await signInToApp(other.context, oidcClients.nextcloud, 'openid profile email');
	const tailscale = await signInToApp(
		other.context,
		oidcClients.tailscale,
		'openid profile email offline_access',
		true
	);
	const sessionId = decodeJwt(nextcloud.id_token).sid as string;
	expect(sessionId).toBeTruthy();
	expect(decodeJwt(tailscale.id_token).sid).toBe(sessionId);

	expect((await other.context.request.post('/api/webauthn/logout')).status()).toBe(204);

	await expect.poll(() => receiver.deliveries.length).toBe(2);
	for (const app of [oidcClients.nextcloud, oidcClients.tailscale]) {
		await verifyLogoutToken(request, deliveryFor(receiver, app), app.id, users.tim.id, sessionId);
	}

	// Only the grant that requested offline_access outlives the session
	expect((await refresh(request, oidcClients.nextcloud, nextcloud.refresh_token)).ok()).toBe(false);
	expect((await refresh(request, oidcClients.tailscale, tailscale.refresh_token)).ok()).toBe(true);
	await other.context.close();
});

test('An app’s logout from another browser ends the session only after the user confirms', async ({
	browser,
	request,
	receiver
}) => {
	const other = await authUtil.signInElsewhere(browser, 'tim');
	const { id_token } = await signInToApp(
		other.context,
		oidcClients.nextcloud,
		'openid profile email'
	);

	const cookieless = await browser.newContext({ storageState: { cookies: [], origins: [] } });
	const page = await cookieless.newPage();
	await page.goto(`/api/oidc/end-session?id_token_hint=${id_token}`);
	await expect(page.getByText(`${oidcClients.nextcloud.name} asked`)).toBeVisible();
	expect((await other.context.request.get('/api/users/me')).status()).toBe(200);

	await interceptCallbackRedirect(page, '/auth/logout/callback', () =>
		page.getByRole('button', { name: 'Sign out' }).click()
	);
	await cookieless.close();

	expect((await other.context.request.get('/api/users/me')).status()).toBe(401);
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	await verifyLogoutToken(
		request,
		receiver.deliveries[0],
		oidcClients.nextcloud.id,
		users.tim.id,
		decodeJwt(id_token).sid as string
	);
	await other.context.close();
});

test('An app’s cross-site logout POST is repeated as a GET that carries the session cookie', async ({
	browser,
	baseURL,
	request,
	receiver
}) => {
	const other = await authUtil.signInElsewhere(browser, 'tim');
	const { id_token } = await signInToApp(
		other.context,
		oidcClients.nextcloud,
		'openid profile email'
	);

	// The app's page lives on another site, so the browser drops the SameSite=Lax cookie from its form post
	const page = await other.context.newPage();
	await page.route('http://rp.test/**', (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: `<form method="post" action="${baseURL}/api/oidc/end-session"><input name="id_token_hint" value="${id_token}"></form><script>document.forms[0].submit()</script>`
		})
	);
	await interceptCallbackRedirect(page, '/auth/logout/callback', () =>
		page.goto('http://rp.test/logout')
	);

	expect((await other.context.request.get('/api/users/me')).status()).toBe(401);
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	await verifyLogoutToken(
		request,
		receiver.deliveries[0],
		oidcClients.nextcloud.id,
		users.tim.id,
		decodeJwt(id_token).sid as string
	);
	await other.context.close();
});

test('An admin signing a user out everywhere also ends their offline_access grants', async ({
	browser,
	request
}) => {
	const craig = await authUtil.signInElsewhere(browser, 'craig');
	const { refresh_token } = await signInToApp(
		craig.context,
		oidcClients.nextcloud,
		'openid profile email offline_access',
		true
	);
	await craig.context.close();

	expect((await request.delete(`/api/users/${users.craig.id}/sessions`)).status()).toBe(204);

	expect((await refresh(request, oidcClients.nextcloud, refresh_token)).ok()).toBe(false);
});

test('Saving a logout URL and revoking an app sends a verifiable logout token', async ({
	page,
	receiver
}) => {
	const client = oidcClients.nextcloud;
	const logoutURL = `${receiver.url}/logout?tenant=test`;
	await page.goto(`/settings/admin/oidc-clients/${client.id}`);
	// The seeded client already has changed advanced settings, so Advanced starts open
	await expect(page.getByRole('button', { name: /^Advanced/ })).toHaveAttribute(
		'aria-expanded',
		'true'
	);
	await page.getByLabel('Back-channel logout URL Optional', { exact: true }).fill(logoutURL);
	await saveUnsavedChanges(page);
	// A changed setting keeps Advanced open after the reload
	await page.reload();
	await expect(page.getByLabel('Back-channel logout URL Optional', { exact: true })).toHaveValue(
		logoutURL
	);

	await page.goto('/settings/apps');
	await page.getByRole('button', { name: `Actions for ${client.name}` }).click();
	await page.getByRole('menuitem', { name: 'Revoke access…' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Revoke' }).click();
	// Other specs can leave retries for the seeded URLs, so observe this client's updated endpoint
	const deliveries = () =>
		receiver.deliveries.filter((delivery) => delivery.path === '/logout?tenant=test');
	await expect.poll(() => deliveries().length).toBe(1);
	await verifyLogoutToken(page.request, deliveries()[0], client.id, users.tim.id);
});

test('Disabling a user delivers logout to their authorized clients', async ({
	request,
	receiver
}) => {
	const clients = [oidcClients.nextcloud, oidcClients.tailscale];
	const userResponse = await request.get(`/api/users/${users.tim.id}`);
	expect(userResponse.ok()).toBeTruthy();
	const disabled = await request.put(`/api/users/${users.tim.id}`, {
		data: { ...(await userResponse.json()), disabled: true }
	});
	expect(disabled.ok()).toBeTruthy();
	await expect.poll(() => receiver.deliveries.length).toBe(clients.length);
	expect(receiver.deliveries.map((delivery) => delivery.path).sort()).toEqual(
		clients.map((client) => new URL(client.backchannelLogoutURL).pathname).sort()
	);
	const tokens = await Promise.all(
		clients.map((client) =>
			verifyLogoutToken(
				request,
				receiver.deliveries.find(
					(delivery) => delivery.path === new URL(client.backchannelLogoutURL).pathname
				)!,
				client.id,
				users.tim.id
			)
		)
	);
	expect(new Set(tokens.map((token) => token.jti)).size).toBe(clients.length);
});

test('Disabling a user also names each session they are still signed in with', async ({
	browser,
	request,
	receiver
}) => {
	const other = await authUtil.signInElsewhere(browser, 'tim');
	const { id_token } = await signInToApp(
		other.context,
		oidcClients.nextcloud,
		'openid profile email'
	);
	await other.context.close();

	const userResponse = await request.get(`/api/users/${users.tim.id}`);
	expect(userResponse.ok()).toBeTruthy();
	const disabled = await request.put(`/api/users/${users.tim.id}`, {
		data: { ...(await userResponse.json()), disabled: true }
	});
	expect(disabled.ok()).toBeTruthy();

	// Every app gets a token that ends all of the user's sessions, and Nextcloud one more for the session it was signed in through
	await expect.poll(() => receiver.deliveries.length).toBe(3);
	const nextcloudPath = new URL(oidcClients.nextcloud.backchannelLogoutURL).pathname;
	const nextcloudTokens = await Promise.all(
		receiver.deliveries
			.filter((delivery) => delivery.path === nextcloudPath)
			.map((delivery) => decodeJwt(delivery.body.get('logout_token')!))
	);
	expect(nextcloudTokens.map((token) => token.sid ?? null).sort()).toEqual(
		[decodeJwt(id_token).sid, null].sort()
	);
	await verifyLogoutToken(
		request,
		deliveryFor(receiver, oidcClients.tailscale),
		oidcClients.tailscale.id,
		users.tim.id
	);
});

test('Deleting a client still notifies its users after authorizations are deleted', async ({
	request,
	receiver
}) => {
	const client = oidcClients.nextcloud;
	expect((await request.delete(`/api/oidc/clients/${client.id}`)).ok()).toBeTruthy();
	expect((await request.get(`/api/oidc/clients/${client.id}`)).status()).toBe(404);
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	await verifyLogoutToken(request, receiver.deliveries[0], client.id, users.tim.id);
});

test('Disabling a client delivers logout to its users', async ({ request, receiver }) => {
	const client = oidcClients.nextcloud;
	const clientResponse = await request.get(`/api/oidc/clients/${client.id}`);
	expect(clientResponse.ok()).toBeTruthy();
	const disabled = await request.put(`/api/oidc/clients/${client.id}`, {
		data: { ...(await clientResponse.json()), disabled: true }
	});
	expect(disabled.ok()).toBeTruthy();
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	await verifyLogoutToken(request, receiver.deliveries[0], client.id, users.tim.id);
});

test('Group access is retained through another allowed group and revoked after the last one', async ({
	request,
	receiver
}) => {
	const client = oidcClients.tailscale;
	expect(
		(
			await request.put(`/api/oidc/clients/${client.id}/allowed-user-groups`, {
				data: { userGroupIds: [userGroups.developers.id, userGroups.designers.id] }
			})
		).ok()
	).toBeTruthy();
	expect(
		(
			await request.put(`/api/users/${users.tim.id}/user-groups`, {
				data: { userGroupIds: [userGroups.designers.id] }
			})
		).ok()
	).toBeTruthy();

	// Observe the asynchronous delivery window before removing the user's remaining access
	await new Promise((resolve) => setTimeout(resolve, 1500));
	expect(receiver.deliveries).toHaveLength(0);
	expect(
		(
			await request.put(`/api/users/${users.tim.id}/user-groups`, {
				data: { userGroupIds: [] }
			})
		).ok()
	).toBeTruthy();
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	await verifyLogoutToken(request, receiver.deliveries[0], client.id, users.tim.id);
});

test('Deleting a user still notifies their clients after authorizations are deleted', async ({
	request,
	receiver
}) => {
	const response = await request.delete(`/api/users/${users.tim.id}`);
	expect(response.status()).toBe(204);
	await expect.poll(() => receiver.deliveries.length).toBe(2);
	for (const client of [oidcClients.nextcloud, oidcClients.tailscale]) {
		const delivery = receiver.deliveries.find(
			(delivery) => delivery.path === new URL(client.backchannelLogoutURL).pathname
		);
		expect(delivery).toBeDefined();
		await verifyLogoutToken(request, delivery!, client.id, users.tim.id);
	}
});

test('Changing a client’s allowed groups logs out users only after their last allowed group is removed', async ({
	request,
	receiver
}) => {
	const client = oidcClients.tailscale;
	const endpoint = `/api/oidc/clients/${client.id}/allowed-user-groups`;
	expect(
		(
			await request.put(endpoint, {
				data: { userGroupIds: [userGroups.developers.id, userGroups.designers.id] }
			})
		).ok()
	).toBeTruthy();
	expect(
		(
			await request.put(endpoint, {
				data: { userGroupIds: [userGroups.designers.id] }
			})
		).ok()
	).toBeTruthy();

	// Retaining an allowed group must not schedule a logout for its members
	await new Promise((resolve) => setTimeout(resolve, 1500));
	expect(receiver.deliveries).toHaveLength(0);
	expect((await request.put(endpoint, { data: { userGroupIds: [] } })).ok()).toBeTruthy();
	await expect.poll(() => receiver.deliveries.length).toBe(1);
	expect(receiver.deliveries[0].path).toBe('/tailscale');
	await verifyLogoutToken(request, receiver.deliveries[0], client.id, users.tim.id);
});
