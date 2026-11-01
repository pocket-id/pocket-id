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

test('Dashboard shows only clients with launch URLs in the correct order', async ({ page }) => {
	const client1 = oidcClients.nextcloud;
	const client2 = oidcClients.immich;

	await page.goto('/settings/apps');

	await expect(apps(page)).toHaveCount(2);

	// Should be first
	const tile1 = apps(page).first();
	await expect(tile1.getByRole('link')).toHaveText(client1.name);
	await expect(tile1.getByText(new URL(client1.launchURL).hostname)).toBeVisible();

	const tile2 = app(page, client2.name);
	await expect(tile2.getByRole('link', { name: client2.name })).toBeVisible();
	await expect(tile2.getByText(new URL(client2.launchURL).hostname)).toBeVisible();

	await expect(app(page, oidcClients.tailscale.name)).toHaveCount(0);
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

	const clientRow = app(page, client.name);
	await expect(clientRow).toHaveCount(0);

	await page.getByRole('button', { name: /Apps without a launch link/ }).click();

	await expect(
		page.getByRole('list', { name: /Apps without a launch link/ }).getByRole('listitem')
	).toHaveCount(2);
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

test('Sort apps by name and remember the choice', async ({ page }) => {
	await page.goto('/settings/apps');

	const sortButton = page.getByRole('button', { name: 'Sort by' });
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.nextcloud.name);

	await sortButton.click();
	await page.getByRole('menuitemradio', { name: 'Name A–Z' }).click();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.immich.name);

	await page.reload();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.immich.name);

	await sortButton.click();
	await expect(page.getByRole('menuitemradio', { name: 'Name A–Z' })).toHaveAttribute(
		'aria-checked',
		'true'
	);
	await page.getByRole('menuitemradio', { name: 'Name Z–A' }).click();
	await expect(apps(page).first().getByRole('link')).toHaveText(oidcClients.nextcloud.name);
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
