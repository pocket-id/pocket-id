import test, { expect } from '@playwright/test';
import { oidcClients } from './data';
import { cleanupBackend } from './utils/cleanup.util';

test.beforeEach(cleanupBackend);

test('Create OIDC client', async ({ page }) => {
	await page.goto('/settings/admin/oidc-clients');
	const oidcClient = oidcClients.pingvinShare;

	await page.getByRole('button', { name: 'Add OIDC Client' }).click();
	await page.getByLabel('Name').fill(oidcClient.name);

	await page.getByTestId('callback-url-1').fill(oidcClient.callbackUrl);
	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByTestId('callback-url-2').fill(oidcClient.secondCallbackUrl!);

	await page.getByLabel('logo').setInputFiles('tests/assets/pingvin-share-logo.png');
	await page.getByRole('button', { name: 'Save' }).click();

	const clientId = await page.getByTestId('client-id').textContent();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client created successfully'
	);
	expect(clientId?.length).toBe(36);
	expect((await page.getByTestId('client-secret').textContent())?.length).toBe(32);
	await expect(page.getByLabel('Name')).toHaveValue(oidcClient.name);
	await expect(page.getByTestId('callback-url-1')).toHaveValue(oidcClient.callbackUrl);
	await expect(page.getByTestId('callback-url-2')).toHaveValue(oidcClient.secondCallbackUrl!);
	await expect(page.getByRole('img', { name: `${oidcClient.name} logo` })).toBeVisible();
	await page.request
		.get(`/api/oidc/clients/${clientId}/logo`)
		.then((res) => expect.soft(res.status()).toBe(200));
});

test('Edit OIDC client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${oidcClient.id}`);

	await page.getByLabel('Name').fill('Nextcloud updated');
	await page.getByTestId('callback-url-1').first().fill('http://nextcloud-updated/auth/callback');
	await page.getByLabel('logo').setInputFiles('tests/assets/nextcloud-logo.png');
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client updated successfully'
	);
	await expect(page.getByRole('img', { name: 'Nextcloud updated logo' })).toBeVisible();
	await page.request
		.get(`/api/oidc/clients/${oidcClient.id}/logo`)
		.then((res) => expect.soft(res.status()).toBe(200));
});

test('Create new OIDC client secret', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${oidcClient.id}`);

	await page.getByLabel('Create new client secret').click();
	await page.getByRole('button', { name: 'Generate' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'New client secret created successfully'
	);
	expect((await page.getByTestId('client-secret').textContent())?.length).toBe(32);
});

test('Delete OIDC client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto('/settings/admin/oidc-clients');

	await page.getByRole('row', { name: oidcClient.name }).getByLabel('Delete').click();
	await page.getByText('Delete', { exact: true }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client deleted successfully'
	);
	await expect(page.getByRole('row', { name: oidcClient.name })).not.toBeVisible();
});
