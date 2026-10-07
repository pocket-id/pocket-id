import test, { expect } from '@playwright/test';
import authUtil from 'utils/auth.util';
import { oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('Dashboard shows only clients with launch URLs in the correct order', async ({ page }) => {
	const client1 = oidcClients.nextcloud;
	const client2 = oidcClients.immich;

	await page.goto('/settings/apps');

	const appCards = page.getByRole('article');
	await expect(appCards).toHaveCount(2);

	// Should be first
	const card1 = appCards.first();

	await expect(card1.getByRole('heading')).toHaveText(client1.name);
	await expect(card1.getByText(new URL(client1.launchURL).hostname)).toBeVisible();

	const card2 = page.getByRole('article', { name: client2.name });
	await expect(card2.getByRole('heading', { name: client2.name })).toBeVisible();
	await expect(card2.getByText(new URL(client2.launchURL).hostname)).toBeVisible();

	await expect(page.getByRole('article', { name: oidcClients.tailscale.name })).toHaveCount(0);
});

test.describe('Dashboard shows only clients where user has access', () => {
	test("User can't see a restricted launchable client", async ({ page }) => {
		await authUtil.changeUser(page, 'craig');
		await page.goto('/settings/apps');

		await expect(page.getByRole('article')).toHaveCount(1);
		await expect(page.getByRole('article', { name: oidcClients.nextcloud.name })).toBeVisible();
		await expect(page.getByRole('article', { name: oidcClients.immich.name })).toHaveCount(0);
	});

	test('User can see every accessible launchable client', async ({ page }) => {
		await page.goto('/settings/apps');

		await expect(page.getByRole('article')).toHaveCount(2);
		await expect(page.getByRole('article', { name: oidcClients.nextcloud.name })).toBeVisible();
		await expect(page.getByRole('article', { name: oidcClients.immich.name })).toBeVisible();
	});
});

test('Show and revoke a hidden authorized client in the app grid', async ({ page }) => {
	const client = oidcClients.tailscale;

	await page.goto('/settings/apps');

	const appCards = page.getByRole('article');
	const clientCard = page.getByRole('article', { name: client.name });
	await expect(clientCard).toHaveCount(0);

	await page.getByRole('button', { name: /Show all apps/ }).click();

	await expect(appCards).toHaveCount(4);
	await expect(clientCard).toBeVisible();
	await expect(page.getByRole('main').getByRole('separator')).toBeVisible();
	await expect(clientCard.getByRole('link', { name: 'Launch' })).toHaveCount(0);
	await expect(clientCard.getByRole('button', { name: 'Revoke' })).toHaveCount(0);
	await clientCard.getByRole('button', { name: 'Toggle menu' }).click();
	await page.getByRole('menuitem', { name: 'Revoke' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Revoke' }).click();

	await expect(
		page.getByText(`The access to ${client.name} has been successfully revoked.`, { exact: true })
	).toBeVisible();
	await expect(clientCard).toHaveCount(0);
});

test('Launch authorized client', async ({ page }) => {
	const client = oidcClients.nextcloud;

	await page.goto('/settings/apps');

	const appCards = page.getByRole('article');
	await expect(appCards.getByRole('link', { name: 'Launch' })).toHaveCount(2);

	const card = page.getByRole('article', { name: client.name });
	await expect(card.getByRole('link', { name: 'Launch' })).toHaveAttribute(
		'href',
		client.launchURL
	);
});

test('Sort apps by name and remember the choice', async ({ page }) => {
	await page.goto('/settings/apps');

	const appCards = page.getByRole('article');
	const sortButton = page.getByRole('button', { name: 'Sort by' });
	await expect(appCards.first().getByRole('heading')).toHaveText(oidcClients.nextcloud.name);

	await sortButton.click();
	await page.getByRole('menuitemradio', { name: 'Name (A-Z)' }).click();
	await expect(appCards.first().getByRole('heading')).toHaveText(oidcClients.immich.name);

	await page.reload();
	await expect(appCards.first().getByRole('heading')).toHaveText(oidcClients.immich.name);

	await sortButton.click();
	await expect(page.getByRole('menuitemradio', { name: 'Name (A-Z)' })).toHaveAttribute(
		'aria-checked',
		'true'
	);
	await page.getByRole('menuitemradio', { name: 'Name (Z-A)' }).click();
	await expect(appCards.first().getByRole('heading')).toHaveText(oidcClients.nextcloud.name);
});

test('Search apps by name', async ({ page }) => {
	await page.goto('/settings/apps');

	const appCards = page.getByRole('article');
	const searchInput = page.getByRole('searchbox', { name: 'Search apps' });

	await searchInput.fill('NEXT');
	await expect(appCards).toHaveCount(1);
	await expect(page.getByRole('article', { name: oidcClients.nextcloud.name })).toBeVisible();

	await searchInput.fill('does-not-exist');
	await expect(page.getByText('No apps match your search')).toBeVisible();
	await expect(appCards).toHaveCount(0);

	await searchInput.fill('');
	await expect(appCards).toHaveCount(2);
});

test('Search also filters hidden apps', async ({ page }) => {
	await page.goto('/settings/apps');

	await page.getByRole('button', { name: /Show all apps/ }).click();
	await expect(page.getByRole('article')).toHaveCount(4);

	await page.getByRole('searchbox', { name: 'Search apps' }).fill(oidcClients.tailscale.name);
	await expect(page.getByRole('article')).toHaveCount(1);
	await expect(page.getByRole('article', { name: oidcClients.tailscale.name })).toBeVisible();
});
