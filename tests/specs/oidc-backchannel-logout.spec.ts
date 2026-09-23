import base, { expect, type APIRequestContext } from '@playwright/test';
import { createLocalJWKSet, jwtVerify } from 'jose';
import { createServer } from 'node:http';
import { oidcClients, userGroups, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
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
	receiver: async ({}, use) => {
		const receiverURL = new URL(oidcClients.nextcloud.backchannelLogoutURL);
		const receiver: LogoutReceiver = {
			url: receiverURL.origin,
			deliveries: [],
			respond: () => 204
		};
		const server = createServer(async (request, response) => {
			const chunks: Buffer[] = [];
			for await (const chunk of request) chunks.push(Buffer.from(chunk));
			const path = request.url!;
			receiver.deliveries.push({
				path,
				method: request.method!,
				contentType: request.headers['content-type'] ?? '',
				body: new URLSearchParams(Buffer.concat(chunks).toString())
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

async function verifyLogoutToken(
	request: APIRequestContext,
	delivery: Delivery,
	clientId: string,
	userId: string
) {
	expect(delivery.method).toBe('POST');
	expect(delivery.contentType).toBe('application/x-www-form-urlencoded');
	expect(delivery.body.getAll('logout_token')).toHaveLength(1);
	const discoveryResponse = await request.get('/.well-known/openid-configuration');
	expect(discoveryResponse.ok()).toBeTruthy();
	const discovery = await discoveryResponse.json();
	expect(discovery.backchannel_logout_supported).toBe(true);
	expect(discovery.backchannel_logout_session_supported).toBe(false);
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
	expect(payload).not.toHaveProperty('sid');
	return payload;
}

test('Saving a logout URL and revoking an app sends a verifiable logout token', async ({
	page,
	receiver
}) => {
	const client = oidcClients.nextcloud;
	const logoutURL = `${receiver.url}/logout?tenant=test`;
	await page.goto(`/settings/admin/oidc-clients/${client.id}`);
	await page.getByRole('button', { name: 'Show Advanced Options' }).click();
	await page.getByLabel('Back-Channel Logout URL', { exact: true }).fill(logoutURL);
	await saveUnsavedChanges(page);
	await page.reload();
	await page.getByRole('button', { name: 'Show Advanced Options' }).click();
	await expect(page.getByLabel('Back-Channel Logout URL', { exact: true })).toHaveValue(logoutURL);

	await page.goto('/settings/apps');
	await page
		.getByRole('article', { name: client.name })
		.getByRole('button', { name: 'Toggle menu' })
		.click();
	await page.getByRole('menuitem', { name: 'Revoke' }).click();
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
