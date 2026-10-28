import test, { expect } from '@playwright/test';
import * as jose from 'jose';
import { apis, oidcClients } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { cleanupBackend } from '../utils/cleanup.util';
import * as oidcUtil from '../utils/oidc.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

function tokenScopes(claims: jose.JWTPayload): string[] {
	if (Array.isArray((claims as Record<string, unknown>).scp)) {
		return (claims as Record<string, unknown>).scp as string[];
	}
	if (typeof claims.scope === 'string') {
		return claims.scope.split(' ');
	}
	return [];
}

function tokenAudiences(claims: jose.JWTPayload): string[] {
	if (Array.isArray(claims.aud)) return claims.aud;
	if (typeof claims.aud === 'string') return [claims.aud];
	return [];
}

// ---------------------------------------------------------------------------
// Admin UI
// ---------------------------------------------------------------------------

test('Lists the preseeded API', async ({ page }) => {
	await page.goto('/settings/admin/apis');

	const row = page.getByRole('row', { name: apis.orders.name });
	await expect(row).toBeVisible();
	await expect(row).toContainText(apis.orders.resource);
});

test('Create API', async ({ page }) => {
	await page.goto('/settings/admin/apis');

	await page.getByRole('button', { name: 'Add API' }).click();
	await page.getByLabel('Name', { exact: true }).fill('Billing API');
	await page.getByLabel('Resource').fill('https://api.billing.test');
	await page.getByRole('dialog').getByRole('button', { name: 'Add API' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Billing API added');
	await page.waitForURL('/settings/admin/apis/*');

	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Billing API');
	// The resource can't change after creation, so it is shown as an identifier
	await expect(page.getByRole('definition').first()).toContainText('https://api.billing.test');
});

test('Cannot create an API with the issuer as resource', async ({ page }) => {
	const { issuer } = await page.request
		.get('/.well-known/openid-configuration')
		.then((r) => r.json());

	await page.goto('/settings/admin/apis');
	await page.getByRole('button', { name: 'Add API' }).click();
	await page.getByLabel('Name', { exact: true }).fill('Reserved API');
	await page.getByLabel('Resource').fill(issuer);
	await page.getByRole('dialog').getByRole('button', { name: 'Add API' }).click();

	await expect(page.locator('[data-type="error"]')).toHaveText(
		'Resource is reserved by Pocket ID and cannot be used for a custom API'
	);
});

test('Edit the name of an API', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	const name = page.getByRole('tabpanel', { name: 'General' }).getByLabel('Name', { exact: true });
	await name.fill('Orders API renamed');
	await saveUnsavedChanges(page);

	await page.reload();
	await expect(name).toHaveValue('Orders API renamed');
});

test('Add a permission to an API', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);
	await page.getByRole('tab', { name: 'Permissions' }).click();

	// The seeded API already has permissions, so the button reads "Add another"
	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Key', { exact: true }).last().fill('ship:orders');
	await page.getByPlaceholder('Name', { exact: true }).last().fill('Ship orders');
	await saveUnsavedChanges(page);

	await page.reload();
	await page.getByRole('tab', { name: 'Permissions' }).click();
	// The two seeded permissions plus the newly added one
	await expect(page.getByPlaceholder('Key', { exact: true })).toHaveCount(3);
});

test('Delete an API', async ({ page }) => {
	await page.goto('/settings/admin/apis');

	await page.getByRole('row', { name: apis.orders.name }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(`${apis.orders.name} deleted`);
	await expect(page.getByRole('row', { name: apis.orders.name })).not.toBeVisible();
});

test('Delete an API from the details page', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	await page.getByRole('button', { name: 'More actions' }).click();
	await page.getByRole('menuitem', { name: 'Delete API' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(`${apis.orders.name} deleted`);
	await page.waitForURL((url) => url.pathname === '/settings/admin/apis');
	await expect(page.getByRole('row', { name: apis.orders.name })).not.toBeVisible();
});

test('Grant a client user-delegated and client access to API permissions', async ({ page }) => {
	// Nextcloud has no API access granted by default
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);

	// Open the Access tab, whose Advanced section lists no API yet, and add the Orders API
	await page.getByRole('tab', { name: 'Access', exact: true }).click();
	await page.getByRole('button', { name: /^Advanced/ }).click();
	await expect(
		page.getByText(`${oidcClients.nextcloud.name} can’t request tokens for any API.`)
	).toBeVisible();
	await page.getByRole('button', { name: 'Add API…' }).first().click();
	await page.getByRole('row', { name: apis.orders.name }).click();

	// Grant read:orders and write:orders on behalf of users, but only write:orders for the client itself
	const dialog = page.getByRole('dialog');
	await dialog
		.getByRole('checkbox', {
			name: `On behalf of people: ${apis.orders.permissions.readOrders.name}`
		})
		.click();
	await dialog
		.getByRole('checkbox', {
			name: `On behalf of people: ${apis.orders.permissions.writeOrders.name}`
		})
		.click();
	await dialog
		.getByRole('checkbox', {
			name: `As itself: ${apis.orders.permissions.writeOrders.name}`
		})
		.click();
	await dialog.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API access updated');
	// The dialogs carry rows with the same names, so the assertions below wait until they are gone
	await expect(page.getByRole('dialog')).toHaveCount(0);
	const row = page.getByRole('listitem').filter({ hasText: apis.orders.name });
	await expect(row).toContainText('On behalf of people: 2 of 2');
	await expect(row).toContainText('As itself: 1 of 2');

	// The API is not offered a second time, because the selection is filtered server-side
	await page.getByRole('button', { name: 'Add API…' }).click();
	await expect(page.getByRole('dialog').getByText('No items found')).toBeVisible();
});

test('Grant a client access from the API details page', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	// Nextcloud has no API access granted by default, so it can be picked from the client selection
	await page.getByRole('tab', { name: 'Clients' }).click();
	await page.getByRole('button', { name: 'Add client…' }).click();
	await page.getByRole('dialog').getByRole('button', { name: oidcClients.nextcloud.name }).click();

	await page
		.getByRole('checkbox', {
			name: `On behalf of people: ${apis.orders.permissions.readOrders.name}`
		})
		.click();
	await page
		.getByRole('checkbox', {
			name: `As itself: ${apis.orders.permissions.writeOrders.name}`
		})
		.click();
	await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API access updated');
	await expect(page.getByRole('dialog')).toHaveCount(0);
	const row = page.getByRole('listitem').filter({ hasText: oidcClients.nextcloud.name });
	await expect(row).toContainText('On behalf of people: 1 of 2');

	// Clients that already have access are filtered out of the selection, the others stay
	await page.getByRole('button', { name: 'Add client…' }).click();
	const picker = page.getByRole('dialog');
	await expect(picker.getByRole('button', { name: oidcClients.tailscale.name })).toBeVisible();
	await expect(picker.getByRole('button', { name: oidcClients.nextcloud.name })).toHaveCount(0);
	await expect(picker.getByRole('button', { name: oidcClients.immich.name })).toHaveCount(0);
	await page.keyboard.press('Escape');

	// The same grant shows up on the client's side of the relation
	// A granted API keeps the Access tab's Advanced section open
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);
	await page.getByRole('tab', { name: 'Access', exact: true }).click();
	await expect(page.getByRole('listitem').filter({ hasText: apis.orders.name })).toContainText(
		'On behalf of people: 1 of 2'
	);
});

test('Grant a client access to an API without any permission', async ({ page, baseURL }) => {
	const client = oidcClients.nextcloud;
	const api = apis.orders;

	// The client is added with user-delegated access and no permission at all
	await page.goto(`/settings/admin/apis/${api.id}#clients`);
	await page.getByRole('button', { name: 'Add client…' }).click();
	await page.getByRole('dialog').getByRole('button', { name: client.name }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Save access' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API access updated');
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(page.getByRole('listitem').filter({ hasText: client.name })).toContainText(
		'On behalf of people: 0 of 2'
	);

	// A resource request without any scope now succeeds, which is what MCP clients send
	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		resource: api.resource,
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});

	const callbackUrl = await oidcUtil.interceptCallbackRedirect(
		page,
		new URL(client.callbackUrl).pathname,
		async () => {
			await page.goto(`/authorize?${params.toString()}`);
			await page.getByRole('button', { name: /^Continue to / }).click();
		}
	);
	const code = callbackUrl.searchParams.get('code');
	expect(code).toBeTruthy();

	const res = await oidcUtil.exchangeCode(page, {
		grant_type: 'authorization_code',
		redirect_uri: client.callbackUrl,
		code: code!,
		client_id: client.id,
		client_secret: client.secret
	});
	expect(res.access_token).toBeTruthy();

	// The token is audienced to the API and carries no scope
	const claims = jose.decodeJwt(res.access_token!);
	expect(tokenAudiences(claims)).toContain(api.resource);
	expect(tokenAudiences(claims)).not.toContain(baseURL);
	expect(tokenScopes(claims)).toEqual([]);
});

test('Revoke a client from the API details page', async ({ page }) => {
	// Immich is seeded with grants on the Orders API
	await page.goto(`/settings/admin/apis/${apis.orders.id}#clients`);

	const row = page.getByRole('listitem').filter({ hasText: oidcClients.immich.name });
	await expect(row).toBeVisible();
	await row.getByRole('button', { name: `Actions for ${oidcClients.immich.name}` }).click();
	await page.getByRole('menuitem', { name: 'Remove access' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Remove access' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API access updated');
	await expect(row).not.toBeVisible();
});

test('Allow all metadata document clients for an API', async ({ page }) => {
	await setAppConfigValues(page.request, { enabledFeatures: '["cimd"]' });
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	await page.getByRole('tab', { name: 'Metadata documents' }).click();
	await page.getByLabel('Allow all metadata document clients').click();
	await page.getByRole('checkbox', { name: apis.orders.permissions.readOrders.name }).click();
	await saveUnsavedChanges(page);

	await page.reload();
	await page.getByRole('tab', { name: 'Metadata documents' }).click();
	await expect(page.getByLabel('Allow all metadata document clients')).toBeChecked();
	await expect(
		page.getByRole('checkbox', { name: apis.orders.permissions.readOrders.name })
	).toBeChecked();
	await expect(
		page.getByRole('checkbox', { name: apis.orders.permissions.writeOrders.name })
	).not.toBeChecked();
});

// ---------------------------------------------------------------------------
// Authorization flow with the RFC 8707 resource parameter
// ---------------------------------------------------------------------------

test('Authorization with a resource parameter issues a token audienced to that API', async ({
	page,
	baseURL
}) => {
	const client = oidcClients.immich;
	const api = apis.orders;

	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		scope: 'openid email read:orders',
		resource: api.resource,
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa',
		nonce: 'P1gN3PtpKHJgKUVcLpLjm'
	});

	const callbackUrl = await oidcUtil.interceptCallbackRedirect(
		page,
		new URL(client.callbackUrl).pathname,
		async () => {
			await page.goto(`/authorize?${params.toString()}`);
			await page.getByRole('button', { name: /^Continue to / }).click();
		}
	);
	const code = callbackUrl.searchParams.get('code');
	expect(code).toBeTruthy();

	const res = await oidcUtil.exchangeCode(page, {
		grant_type: 'authorization_code',
		redirect_uri: client.callbackUrl,
		code: code!,
		client_id: client.id,
		client_secret: client.secret
	});
	expect(res.access_token).toBeTruthy();

	const claims = jose.decodeJwt(res.access_token!);
	expect(tokenAudiences(claims)).toContain(api.resource);
	// Because openid was requested alongside the resource, the token also carries the issuer audience so it can still reach /userinfo
	expect(tokenAudiences(claims)).toContain(baseURL);
	expect(tokenScopes(claims)).toContain(api.permissions.readOrders.key);

	// The same token can be presented at userinfo, by the client's explicit opt-in of requesting openid
	const userinfo = await page.request.get('/api/oidc/userinfo', {
		headers: { Authorization: 'Bearer ' + res.access_token }
	});
	expect(userinfo.status()).toBe(200);
});

test('Consent screen shows the friendly permission name for a resource request', async ({
	page
}) => {
	const client = oidcClients.immich;
	const api = apis.orders;

	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		scope: 'openid read:orders',
		resource: api.resource,
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});
	await page.goto(`/authorize?${params.toString()}`);

	const scopeList = page.getByTestId('scopes');
	await expect(scopeList).toBeVisible();
	// The permission's friendly name is shown, not the raw scope key
	await expect(scopeList.getByText(api.permissions.readOrders.name, { exact: true })).toBeVisible();
});

test('Requesting a custom scope without its resource is rejected with invalid_scope', async ({
	page
}) => {
	const client = oidcClients.immich;

	// The client is allowed read:orders, but it is requested without the resource parameter
	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		scope: 'openid read:orders',
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});

	const callbackUrl = await oidcUtil.interceptCallbackRedirect(
		page,
		new URL(client.callbackUrl).pathname,
		async () => {
			await page.goto(`/authorize?${params.toString()}`);
		}
	);

	expect(callbackUrl.searchParams.get('error')).toBe('invalid_scope');
	expect(callbackUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
});

// ---------------------------------------------------------------------------
// Separation of user-delegated and client (machine-to-machine) access
// ---------------------------------------------------------------------------

test('Client credentials issues a token for a client-granted permission', async ({ page }) => {
	const client = oidcClients.immich;
	const api = apis.orders;

	// write:orders is granted to Immich for client access
	const res = await oidcUtil.exchangeCode(page, {
		grant_type: 'client_credentials',
		client_id: client.id,
		client_secret: client.secret,
		scope: api.permissions.writeOrders.key,
		resource: api.resource
	});
	expect(res.access_token).toBeTruthy();

	const claims = jose.decodeJwt(res.access_token!);
	expect(tokenAudiences(claims)).toContain(api.resource);
	expect(tokenScopes(claims)).toContain(api.permissions.writeOrders.key);
});

test('Client credentials cannot mint a permission that is only user-delegated', async ({
	page
}) => {
	const client = oidcClients.immich;
	const api = apis.orders;

	// read:orders is only granted for user-delegated access
	const res = await oidcUtil.exchangeCode(page, {
		grant_type: 'client_credentials',
		client_id: client.id,
		client_secret: client.secret,
		scope: api.permissions.readOrders.key,
		resource: api.resource
	});

	expect(res.access_token).toBeFalsy();
	expect(res.error).toBe('invalid_scope');
});

test('Authorization on behalf of a user cannot request a client-only permission', async ({
	page
}) => {
	const client = oidcClients.immich;
	const api = apis.orders;

	// write:orders is only granted for client access, so users cannot be asked to delegate it
	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		scope: `openid ${api.permissions.writeOrders.key}`,
		resource: api.resource,
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});

	const callbackUrl = await oidcUtil.interceptCallbackRedirect(
		page,
		new URL(client.callbackUrl).pathname,
		async () => {
			await page.goto(`/authorize?${params.toString()}`);
		}
	);

	// The authorize endpoint collapses every resource-targeted scope/resource failure into a generic invalid_request
	expect(callbackUrl.searchParams.get('error')).toBe('invalid_request');
	expect(callbackUrl.searchParams.get('state')).toBe('nXx-6Qr-owc1SHBa');
});
