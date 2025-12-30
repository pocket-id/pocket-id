import test, { expect, type Page } from '@playwright/test';
import { cleanupBackend, cleanupScimServiceProvider } from 'utils/cleanup.util';
import { oidcClients, userGroups, users } from '../data';

async function configureOidcClient(page: Page) {
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);

	await page.getByRole('button', { name: 'Expand card' }).nth(1).click();

	await page
		.getByLabel('SCIM Endpoint')
		.fill(process.env.SCIM_SERVICE_PROVIDER_URL_INTERNAL || 'http://scim.provider/api');

	await page.getByRole('button', { name: 'Enable' }).click();
}

async function syncScimServiceProvider(page: Page) {
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);

	await page.getByRole('button', { name: 'Sync' }).click();
	await page.waitForSelector('[data-type="success"]');
}

test.beforeEach(async () => {
	await cleanupBackend({ skipLdapSetup: true });
});

test.describe('SCIM Configuration', () => {
	test('Enable SCIM for OIDC client', async ({ page }) => {
		await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);

		await page.getByRole('button', { name: 'Expand card' }).nth(1).click();

		await page.getByLabel('SCIM Endpoint').fill('http://scim.provider/api');
		await page.getByLabel('SCIM Token').fill('supersecrettoken');

		await page.getByRole('button', { name: 'Enable' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText('SCIM enabled successfully.');

		await page.reload();

		await expect(page.getByLabel('SCIM Endpoint')).toHaveValue('http://scim.provider/api');
		await expect(page.getByLabel('SCIM Token')).toHaveValue('supersecrettoken');
	});

	test('Update SCIM of OIDC client', async ({ page }) => {
		await configureOidcClient(page);

		await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);

		await page.getByLabel('SCIM Endpoint').fill('http://new.scim.provider/api');
		await page.getByLabel('SCIM Token').fill('evenmoresecrettoken');

		await page.getByRole('button', { name: 'Save' }).nth(1).click();

		await expect(page.locator('[data-type="success"]')).toHaveText(
			'SCIM configuration updated successfully.'
		);

		await page.reload();

		await expect(page.getByLabel('SCIM Endpoint')).toHaveValue('http://new.scim.provider/api');
		await expect(page.getByLabel('SCIM Token')).toHaveValue('evenmoresecrettoken');
	});

	test('Disable SCIM of OIDC client', async ({ page }) => {
		await configureOidcClient(page);

		await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);

		await page.getByRole('button', { name: 'Disable' }).click();
		await page.getByRole('button', { name: 'Disable' }).nth(1).click();

		await expect(page.locator('[data-type="success"]')).toHaveText('SCIM disabled successfully.');

		await page.reload();

		await expect(page.getByRole('button', { name: 'Enable' })).toBeVisible();
		await expect(page.getByLabel('SCIM Endpoint')).toHaveValue('');
		await expect(page.getByLabel('SCIM Token')).toHaveValue('');
	});
});

test.describe('SCIM Sync', () => {
	test.skip(
		!process.env.SCIM_SERVICE_PROVIDER_URL || !process.env.SCIM_SERVICE_PROVIDER_URL_INTERNAL,
		'Skipping SCIM Sync tests because SCIM_SERVICE_PROVIDER_URL or SCIM_SERVICE_PROVIDER_URL_INTERNAL is not set'
	);

	test.beforeEach(async ({ page }) => {
		await Promise.all([configureOidcClient(page), cleanupScimServiceProvider()]);
	});

	test('Sync client', async ({ page }) => {
		await syncScimServiceProvider(page);

		const scimUsers = await getScimResources('Users');
		await expect(scimUsers.length).toBe(2);

		const groups = await getScimResources('Groups');
		await expect(groups.length).toBe(2);

		const timUser = scimUsers.find((u: any) => u.userName === 'tim');
		await expect(timUser).toBeDefined();
		await expect(timUser).toMatchObject({
			externalId: users.tim.id,
			emails: [
				{
					value: users.tim.email,
					primary: true
				}
			],
			name: {
				givenName: users.tim.firstname,
				familyName: users.tim.lastname
			},
			displayName: users.tim.displayName,
			active: true
		});
	});

	test('Remove allowed group and sync', async ({ page }) => {
		await syncScimServiceProvider(page);

		await page.getByRole('button', { name: 'Expand card' }).first().click();

		await page
			.getByRole('row', { name: userGroups.developers.name })
			.getByRole('cell')
			.first()
			.click();

		await page.getByRole('button', { name: 'Save' }).nth(1).click();

		await syncScimServiceProvider(page);

		const scimUsers = await getScimResources('Users');
		await expect(scimUsers.length).toBe(1);
		await expect(scimUsers.find((u: any) => u.userName === users.tim)).toBeUndefined();

		const scimGroups = await getScimResources('Groups');
		await expect(scimGroups.length).toBe(1);
		await expect(
			scimGroups.find((g: any) => g.displayName === userGroups.developers.name)
		).toBeUndefined();
	});

	test('Remove group restrictions and sync', async ({ page }) => {
		await syncScimServiceProvider(page);

		await page.getByRole('button', { name: 'Expand card' }).first().click();

		await page.getByRole('button', { name: 'Unrestrict' }).click();
		await page.getByRole('button', { name: 'Unrestrict' }).nth(1).click();

		await syncScimServiceProvider(page);

		const scimUsers = await getScimResources('Users');
		await expect(scimUsers.length).toBe(3);

		const scimGroups = await getScimResources('Groups');
		await expect(scimGroups.length).toBe(2);
	});
});

async function getScimResources(resourceType: 'Users' | 'Groups') {
	const response = await fetch(`${process.env.SCIM_SERVICE_PROVIDER_URL}/${resourceType}`).then(
		(res) => res.json()
	);
	return response['Resources'];
}
