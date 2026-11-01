import test, { expect, type Page } from '@playwright/test';
import { cleanupBackend, cleanupScimServiceProvider } from 'utils/cleanup.util';
import { oidcClients, userGroups, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { saveUnsavedChanges, unsavedChangesBar } from '../utils/unsaved-changes.util';

async function openScimTab(page: Page) {
	await setAppConfigValues(page.request, { enabledFeatures: '["scim"]' });
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.scim.id}`);
	await page.getByRole('tab', { name: 'SCIM', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'SCIM provisioning' })).toBeVisible();
}

async function configureOidcClient(page: Page) {
	await openScimTab(page);

	await page
		.getByLabel('SCIM endpoint')
		.fill(process.env.SCIM_SERVICE_PROVIDER_URL_INTERNAL || 'http://scim.provider/api');

	await page.getByRole('button', { name: 'Enable provisioning' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Provisioning enabled');
}

async function syncScimServiceProvider(page: Page) {
	await openScimTab(page);

	await page.getByRole('button', { name: 'Sync now' }).click();
	await page.waitForSelector('[data-type="success"]');
}

test.beforeEach(async () => {
	await cleanupBackend({ skipLdapSetup: true });
});

test.describe('SCIM Configuration', () => {
	test('Enable SCIM for OIDC client', async ({ page }) => {
		await openScimTab(page);

		await page.getByLabel('SCIM endpoint').fill('http://scim.provider/api');
		await page.getByLabel('SCIM token').fill('supersecrettoken');
		await expect(unsavedChangesBar(page)).toHaveCount(0);

		await page.getByRole('button', { name: 'Enable provisioning' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText('Provisioning enabled');

		await page.getByLabel('SCIM endpoint').fill('http://edited.scim.provider/api');
		await expect(unsavedChangesBar(page)).toContainText(/\d+ unsaved changes?/);
		await page.getByRole('button', { name: 'Discard', exact: true }).click();
		await expect(page.getByLabel('SCIM endpoint')).toHaveValue('http://scim.provider/api');

		await page.reload();

		await expect(page.getByLabel('SCIM endpoint')).toHaveValue('http://scim.provider/api');
		await expect(page.getByLabel('SCIM token')).toHaveValue('supersecrettoken');
	});

	test('Update SCIM of OIDC client', async ({ page }) => {
		await configureOidcClient(page);

		await openScimTab(page);

		await page.getByLabel('SCIM endpoint').fill('http://new.scim.provider/api');
		await page.getByLabel('SCIM token').fill('evenmoresecrettoken');

		await saveUnsavedChanges(page);

		await page.reload();

		await expect(page.getByLabel('SCIM endpoint')).toHaveValue('http://new.scim.provider/api');
		await expect(page.getByLabel('SCIM token')).toHaveValue('evenmoresecrettoken');
	});

	test('Disable SCIM of OIDC client', async ({ page }) => {
		await configureOidcClient(page);

		await openScimTab(page);

		await page.getByRole('button', { name: 'Disable provisioning…' }).click();
		await page
			.getByRole('alertdialog')
			.getByRole('button', { name: 'Disable provisioning' })
			.click();

		await expect(page.locator('[data-type="success"]')).toHaveText('Provisioning disabled');

		await page.reload();

		await expect(page.getByRole('button', { name: 'Enable provisioning' })).toBeVisible();
		await expect(page.getByLabel('SCIM endpoint')).toHaveValue('');
		await expect(page.getByLabel('SCIM token')).toHaveValue('');
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

		await expectScimResourceCount('Users', 2);
		const scimUsers = await getScimResources('Users');

		await expectScimResourceCount('Groups', 2);
		const groups = await getScimResources('Groups');

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

		await page.getByRole('tab', { name: 'Access', exact: true }).click();

		const developersCheckbox = page
			.getByRole('row', { name: userGroups.developers.name })
			.getByRole('checkbox');
		await developersCheckbox.click();
		await expect(developersCheckbox).toHaveAttribute('data-state', 'unchecked');

		await saveUnsavedChanges(page);

		await syncScimServiceProvider(page);

		await expectScimResourceCount('Users', 1);
		const scimUsers = await getScimResources('Users');
		await expect(scimUsers.find((u: any) => u.userName === users.tim.username)).toBeDefined();

		await expectScimResourceCount('Groups', 1);
		const scimGroups = await getScimResources('Groups');
		await expect(
			scimGroups.find((g: any) => g.displayName === userGroups.designers.friendlyName)
		).toBeDefined();
	});

	test('Remove group restrictions and sync', async ({ page }) => {
		await syncScimServiceProvider(page);

		await page.getByRole('tab', { name: 'Access', exact: true }).click();

		await page.getByRole('radio', { name: 'All users' }).click();
		await page.getByRole('alertdialog').getByRole('button', { name: 'Allow all users' }).click();
		await expect(page.getByText('Who can sign in updated', { exact: true })).toBeVisible();

		await syncScimServiceProvider(page);

		await expectScimResourceCount('Users', 3);
		await expectScimResourceCount('Groups', 2);
	});
});

async function expectScimResourceCount(resourceType: 'Users' | 'Groups', count: number) {
	await expect.poll(async () => (await getScimResources(resourceType)).length).toBe(count);
}

async function getScimResources(resourceType: 'Users' | 'Groups') {
	const response = await fetch(`${process.env.SCIM_SERVICE_PROVIDER_URL}/${resourceType}`).then(
		(res) => res.json()
	);
	return response['Resources'];
}
