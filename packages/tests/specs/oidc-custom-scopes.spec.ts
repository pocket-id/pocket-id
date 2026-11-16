import test, { expect, type Page } from '@playwright/test';
import * as jose from 'jose';
import { oidcClients, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { cleanupBackend } from '../utils/cleanup.util';
import * as oidcUtil from '../utils/oidc.util';

test.beforeEach(async ({ page }) => {
	await cleanupBackend();
	await setAppConfigValues(page.request, { enabledFeatures: '["claimMappings"]' });
	await page.request.put(`/api/admin/custom-claims/user/${users.tim.id}`, {
		data: [
			{ key: 'k8s_role', value: 'admin' },
			{ key: 'caddyfile', value: 'tim.caddy' }
		]
	});
});

async function authorize(
	page: Page,
	client: { id: string; callbackUrl: string; secret: string },
	scope: string,
	consent: () => Promise<void>
) {
	const params = new URLSearchParams({
		client_id: client.id,
		response_type: 'code',
		scope,
		redirect_uri: client.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});
	const callbackUrl = await oidcUtil.interceptCallbackRedirect(page, '/auth/callback', async () => {
		await page.goto(`/authorize?${params.toString()}`);
		await consent();
	});

	const token = (await oidcUtil.exchangeCode(page, {
		grant_type: 'authorization_code',
		code: callbackUrl.searchParams.get('code')!,
		redirect_uri: client.callbackUrl,
		client_id: client.id,
		client_secret: client.secret
	})) as { id_token: string; scope: string };
	return { scope: token.scope.split(' '), idToken: jose.decodeJwt(token.id_token) };
}

// Assigns Immich a policy that releases k8s_role with the kuber scope, which also lets Immich request kuber
async function assignKubernetesPolicy(page: Page) {
	const customClaim = (key: string, scope: string) => ({
		claimName: key,
		sourceType: 'custom_claim',
		sourceValue: key,
		scope: [scope],
		idToken: true,
		accessToken: false,
		userInfo: true
	});
	const res = await page.request.post('/api/admin/oidc/claim-mapping-policies', {
		data: {
			name: 'Kubernetes',
			isDefault: false,
			claimMappings: [customClaim('k8s_role', 'kuber'), customClaim('caddyfile', 'profile')]
		}
	});
	const policy = await res.json();
	await page.request.post(
		`/api/admin/oidc/claim-mapping-policies/${policy.id}/clients/${oidcClients.immich.id}`
	);
}

test('Create a custom scope that shows on the consent screen', async ({ page }) => {
	await page.goto('/settings/admin/claim-mappings');
	await page.getByRole('tab', { name: 'Scopes' }).click();
	await page.getByRole('button', { name: 'Add scope' }).click();

	const sheet = page.getByRole('dialog');
	await sheet.getByLabel('Name').fill('Kubernetes role');
	await sheet.getByLabel('Scope', { exact: true }).fill('kuber');
	await sheet.getByLabel('Description').fill('Share your Kubernetes role with this app');
	await sheet.getByRole('button', { name: 'Add scope' }).click();

	await expect(page.getByRole('cell', { name: 'Kubernetes role kuber' })).toBeVisible();
	await expect(page.getByText('Not used by any policy')).toBeVisible();

	await assignKubernetesPolicy(page);

	const params = new URLSearchParams({
		client_id: oidcClients.immich.id,
		response_type: 'code',
		scope: 'openid profile kuber',
		redirect_uri: oidcClients.immich.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa'
	});
	await page.goto(`/authorize?${params.toString()}`);
	await expect(
		page.getByRole('checkbox', { name: 'Kubernetes role Share your Kubernetes role with this app' })
	).toBeChecked();
});

test('A claim released with a custom scope reaches only clients whose policy uses the scope', async ({
	page
}) => {
	await page.request.post('/api/admin/oidc/custom-scopes', {
		data: {
			key: 'kuber',
			name: 'Kubernetes role',
			description: 'Share your Kubernetes role with this app'
		}
	});
	await assignKubernetesPolicy(page);

	const allowed = await authorize(page, oidcClients.immich, 'openid profile kuber', () =>
		page.getByRole('button', { name: /^Continue to / }).click()
	);
	expect(allowed.scope).toContain('kuber');
	expect(allowed.idToken.k8s_role).toBe('admin');
	expect(allowed.idToken.caddyfile).toBe('tim.caddy');

	// Nextcloud is on the built-in policy, which doesn't use the scope, so it can't request it
	// It leaves out profile, because the built-in policy releases every custom claim with profile
	const other = await authorize(page, oidcClients.nextcloud, 'openid kuber', async () => {});
	expect(other.scope).not.toContain('kuber');
	expect(other.idToken.k8s_role).toBeUndefined();
});
