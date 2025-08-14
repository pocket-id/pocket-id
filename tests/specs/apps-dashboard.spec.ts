import test, { expect } from '@playwright/test';
import { oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(() => cleanupBackend());

test('Dashboard shows all authorized clients in the correct order', async ({ page }) => {
	const client1 = oidcClients.tailscale;
	const client2 = oidcClients.nextcloud;

	await page.goto('/settings/apps');

	await expect(page.getByTestId('authorized-oidc-client-card')).toHaveCount(4);

	// Should be first
	const card1 = page.getByTestId('authorized-oidc-client-card').first();

	await expect(card1.getByRole('heading')).toHaveText(client1.name);

	const card2 = page.getByTestId('authorized-oidc-client-card').nth(1);
	await expect(card2.getByRole('heading', { name: client2.name })).toBeVisible();
	await expect(card2.getByText(new URL(client2.launchURL).hostname)).toBeVisible();
});

test('Revoke authorized client', async ({ page }) => {
	const client = oidcClients.tailscale;

	await page.goto('/settings/apps');

	page
		.getByTestId('authorized-oidc-client-card')
		.first()
		.getByRole('button', { name: 'Toggle menu' })
		.click();

	await page.getByRole('menuitem', { name: 'Revoke' }).click();
	await page.getByRole('button', { name: 'Revoke' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		`The access to ${client.name} has been successfully revoked.`
	);

	await expect(page.getByTestId('authorized-oidc-client-card')).toHaveCount(4);
});

test('Launch authorized client', async ({ page }) => {
	const client = oidcClients.nextcloud;

	await page.goto('/settings/apps');

	const card1 = page.getByTestId('authorized-oidc-client-card').first();
	await expect(card1.getByRole('button', { name: 'Launch' })).toBeDisabled();

	const card2 = page.getByTestId('authorized-oidc-client-card').nth(1);
	await expect(card2.getByRole('link', { name: 'Launch' })).toHaveAttribute(
		'href',
		client.launchURL
	);
});
