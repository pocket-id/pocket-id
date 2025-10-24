import test, { expect, Page } from '@playwright/test';
import { oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test.describe('Create OIDC client', () => {
	async function createClientTest(page: Page, clientId?: string) {
		const oidcClient = oidcClients.pingvinShare;
		await page.goto('/settings/admin/oidc-clients');
		await page.getByRole('button', { name: 'Add OIDC Client' }).click();

		await page.getByLabel('Name').fill(oidcClient.name);
		await page.getByLabel('Client Launch URL').fill(oidcClient.launchURL);

		await page.getByRole('button', { name: 'Add' }).first().click();
		await page.getByTestId('callback-url-1').fill(oidcClient.callbackUrl);

		await page.getByRole('button', { name: 'Add another' }).click();
		await page.getByTestId('callback-url-2').fill(oidcClient.secondCallbackUrl);

		await page.locator('[role="tab"][data-value="light-logo"]').first().click();
		await page.setInputFiles('#oidc-client-logo-light', 'assets/pingvin-share-logo.png');
		await page.locator('[role="tab"][data-value="dark-logo"]').first().click();
		await page.setInputFiles('#oidc-client-logo-dark', 'assets/pingvin-share-logo.png');

		if (clientId) {
			await page.getByRole('button', { name: 'Show Advanced Options' }).click();
			await page.getByLabel('Client ID').fill(clientId);
		}

		await page.getByRole('button', { name: 'Save' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText(
			'OIDC client created successfully'
		);

		const resolvedClientId = (await page.getByTestId('client-id').innerText()).trim();
		const clientSecret = (await page.getByTestId('client-secret').innerText()).trim();

		if (clientId) {
			expect(resolvedClientId).toBe(clientId);
		} else {
			expect(resolvedClientId).toMatch(/^[\w-]{36}$/);
		}

		expect(clientSecret).toMatch(/^\w{32}$/);

		await expect(page.getByLabel('Name')).toHaveValue(oidcClient.name);
		await expect(page.getByTestId('callback-url-1')).toHaveValue(oidcClient.callbackUrl);
		await expect(page.getByTestId('callback-url-2')).toHaveValue(oidcClient.secondCallbackUrl);
		await expect(page.getByRole('img', { name: `${oidcClient.name} logo` }).first()).toBeVisible();

		const res = await page.request.get(`/api/oidc/clients/${resolvedClientId}/logo`);
		expect(res.ok()).toBeTruthy();
	}

	test('with auto-generated client ID', async ({ page }) => {
		await createClientTest(page);
	});

	test('with custom client ID', async ({ page }) => {
		await createClientTest(page, '123e4567-e89b-12d3-a456-426614174000');
	});
});

test('Edit OIDC client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${oidcClient.id}`);

	await page.getByLabel('Name').fill('Nextcloud updated');
	await page.getByTestId('callback-url-1').first().fill('http://nextcloud-updated/auth/callback');
	await page.locator('[role="tab"][data-value="light-logo"]').first().click();
	await page.setInputFiles('#oidc-client-logo-light', 'assets/nextcloud-logo.png');
	await page.locator('[role="tab"][data-value="dark-logo"]').first().click();
	await page.setInputFiles('#oidc-client-logo-dark', 'assets/nextcloud-logo.png');
	await page.getByLabel('Client Launch URL').fill(oidcClient.launchURL);
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client updated successfully'
	);
	await expect(page.getByRole('img', { name: 'Nextcloud updated logo' }).first()).toBeVisible();
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

	await page
		.getByRole('row', { name: oidcClient.name })
		.getByRole('button', { name: 'Toggle menu' })
		.click();

	await page.getByRole('menuitem', { name: 'Delete' }).click();

	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client deleted successfully'
	);
	await expect(page.getByRole('row', { name: oidcClient.name })).not.toBeVisible();
});
