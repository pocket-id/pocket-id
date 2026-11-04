import test, { expect, type Page } from '@playwright/test';
import authUtil from 'utils/auth.util';
import { oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

function apps(page: Page) {
	return page.getByRole('list', { name: 'All apps' }).getByRole('listitem');
}

function app(page: Page, name: string) {
	return page.getByRole('listitem').filter({ has: page.getByText(name, { exact: true }) });
}

test('Dashboard shows only clients with launch URLs sorted by name', async ({ page }) => {
	await page.goto('/settings/apps');

	await expect(apps(page)).toHaveCount(2);
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.immich.name);
	await expect(apps(page).last().getByRole('link')).toHaveText(oidcClients.nextcloud.name);

	await expect(apps(page).filter({ hasText: oidcClients.tailscale.name })).toHaveCount(0);
});

test.describe('Dashboard shows only clients where user has access', () => {
	test("User can't see a restricted launchable client", async ({ page }) => {
		await authUtil.changeUser(page, 'craig');
		await page.goto('/settings/apps');

		await expect(apps(page)).toHaveCount(1);
		await expect(app(page, oidcClients.nextcloud.name)).toBeVisible();
		await expect(app(page, oidcClients.immich.name)).toHaveCount(0);
	});

	test('User can see every accessible launchable client', async ({ page }) => {
		await page.goto('/settings/apps');

		await expect(apps(page)).toHaveCount(2);
		await expect(app(page, oidcClients.nextcloud.name)).toBeVisible();
		await expect(app(page, oidcClients.immich.name)).toBeVisible();
	});
});

test('Show and revoke an app without a launch link', async ({ page }) => {
	const client = oidcClients.tailscale;

	await page.goto('/settings/apps');

	const otherApps = page.getByRole('list', { name: 'Others' }).getByRole('listitem');
	await expect(otherApps).toHaveCount(0);

	await page.getByRole('button', { name: /^Others/ }).click();
	await expect(otherApps).toHaveCount(2);

	const clientRow = app(page, client.name);
	await expect(clientRow).toBeVisible();
	await expect(clientRow.getByRole('link')).toHaveCount(0);
	await clientRow.getByRole('button', { name: `Actions for ${client.name}` }).click();
	await page.getByRole('menuitem', { name: 'Revoke access…' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Revoke' }).click();

	await expect(page.getByText(`${client.name}’s access revoked`, { exact: true })).toBeVisible();
	await expect(clientRow).toHaveCount(0);
});

test('Launch authorized client', async ({ page }) => {
	const client = oidcClients.nextcloud;

	await page.goto('/settings/apps');

	await expect(app(page, client.name).getByRole('link', { name: client.name })).toHaveAttribute(
		'href',
		client.launchURL
	);
});

test('Pin an app and keep it pinned after a reload', async ({ page }) => {
	const client = oidcClients.nextcloud;
	const pinned = page.getByRole('region', { name: 'Pinned' }).getByRole('listitem');

	await page.goto('/settings/apps');
	await expect(pinned).toHaveCount(0);

	await apps(page)
		.filter({ hasText: client.name })
		.getByRole('button', { name: `Actions for ${client.name}` })
		.click();
	await page.getByRole('menuitem', { name: 'Pin', exact: true }).click();
	await expect(pinned).toHaveCount(1);
	await expect(pinned.getByRole('link')).toHaveText(client.name);
	await expect(apps(page)).toHaveCount(1);
	await expect(apps(page).filter({ hasText: client.name })).toHaveCount(0);

	await page.reload();
	await expect(pinned).toHaveCount(1);

	await pinned.getByRole('button', { name: `Actions for ${client.name}` }).click();
	await page.getByRole('menuitem', { name: 'Unpin' }).click();
	await expect(pinned).toHaveCount(0);
	await expect(apps(page)).toHaveCount(2);
});

test('Sort apps and remember the choice', async ({ page }) => {
	await page.goto('/settings/apps');

	const sortButton = page.getByRole('button', { name: 'Sort by' });
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.immich.name);

	await sortButton.click();
	await page.getByRole('menuitemradio', { name: 'Name Z–A' }).click();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.nextcloud.name);

	await page.reload();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.nextcloud.name);

	await sortButton.click();
	await expect(page.getByRole('menuitemradio', { name: 'Name Z–A' })).toHaveAttribute(
		'aria-checked',
		'true'
	);
	await page.getByRole('menuitemradio', { name: 'Name A–Z' }).click();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.immich.name);
});

test('Search apps by name', async ({ page }) => {
	await page.goto('/settings/apps');

	const searchInput = page.getByRole('searchbox', { name: 'Search apps' });

	await searchInput.fill('NEXT');
	await expect(apps(page)).toHaveCount(1);
	await expect(app(page, oidcClients.nextcloud.name)).toBeVisible();

	await searchInput.fill('does-not-exist');
	await expect(page.getByText('No apps match “does-not-exist”')).toBeVisible();
	await expect(apps(page)).toHaveCount(0);

	await page.getByRole('button', { name: 'Clear search' }).click();
	await expect(apps(page)).toHaveCount(2);
});

test('Search also finds apps without a launch link', async ({ page }) => {
	await page.goto('/settings/apps');

	await page.getByRole('searchbox', { name: 'Search apps' }).fill(oidcClients.tailscale.name);
	await expect(apps(page)).toHaveCount(0);
	await expect(app(page, oidcClients.tailscale.name)).toBeVisible();
});
