import test, { expect } from '@playwright/test';
import * as jose from 'jose';
import { apis, oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import * as oidcUtil from '../utils/oidc.util';

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
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API created successfully');
	await page.waitForURL('/settings/admin/apis/*');

	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Billing API');
	await expect(page.getByLabel('Resource')).toHaveValue('https://api.billing.test');
});

test('Cannot create an API with the issuer as resource', async ({ page }) => {
	const { issuer } = await page.request
		.get('/.well-known/openid-configuration')
		.then((r) => r.json());

	await page.goto('/settings/admin/apis');
	await page.getByRole('button', { name: 'Add API' }).click();
	await page.getByLabel('Name', { exact: true }).fill('Reserved API');
	await page.getByLabel('Resource').fill(issuer);
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="error"]')).toContainText('reserved');
});

test('Edit the name of an API', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	await page.getByLabel('Name', { exact: true }).fill('Orders API renamed');
	await page.getByRole('button', { name: 'Save' }).nth(0).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API updated successfully');

	await page.reload();
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Orders API renamed');
});

test('Add a permission to an API', async ({ page }) => {
	await page.goto(`/settings/admin/apis/${apis.orders.id}`);

	// The seeded API already has permissions, so the button reads "Add another"
	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Permission', { exact: true }).last().fill('ship:orders');
	await page.getByPlaceholder('Name', { exact: true }).last().fill('Ship orders');
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Permissions updated successfully'
	);

	await page.reload();
	// The two seeded permissions plus the newly added one
	await expect(page.getByPlaceholder('Permission', { exact: true })).toHaveCount(3);
});

test('Delete an API', async ({ page }) => {
	await page.goto('/settings/admin/apis');

	await page.getByRole('row', { name: apis.orders.name }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API deleted successfully');
	await expect(page.getByRole('row', { name: apis.orders.name })).not.toBeVisible();
});

test('Grant a client access to an API permission', async ({ page }) => {
	// Nextcloud has no API access granted by default
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);

	// Expand the API access card, then edit the Orders API row
	await page.getByText('API access', { exact: true }).click();
	await page
		.getByRole('row', { name: apis.orders.name })
		.getByRole('button', { name: 'Edit' })
		.click();

	const dialog = page.getByRole('dialog');
	await dialog
		.getByRole('row', { name: apis.orders.permissions.readOrders.name })
		.getByRole('checkbox')
		.click();
	await dialog.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('API access updated successfully');
	await expect(page.getByRole('row', { name: apis.orders.name })).toContainText('1 / 2');
});

// ---------------------------------------------------------------------------
// Authorization flow with the RFC 8707 resource parameter
// ---------------------------------------------------------------------------

test('Authorization with a resource parameter issues a token audienced to that API', async ({
	page
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
			await page.getByRole('button', { name: 'Sign in' }).click();
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
	expect(tokenScopes(claims)).toContain(api.permissions.readOrders.key);
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
