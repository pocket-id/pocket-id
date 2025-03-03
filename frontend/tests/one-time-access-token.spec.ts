import test, { expect } from '@playwright/test';
import { oneTimeAccessTokens } from './data';
import { cleanupBackend } from './utils/cleanup.util';

test.beforeEach(cleanupBackend);

// Disable authentication for these tests
test.use({ storageState: { cookies: [], origins: [] } });

test('Sign in with login code', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => !t.expired)[0];
	await page.goto(`/lc/${token.token}`);

	await page.waitForURL('/settings/account');
});

test('Sign in with login code entered manually', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => !t.expired)[0];
	await page.goto('/lc');

	await page.getByPlaceholder('Code').first().fill(token.token);

	await page.getByText('Submit').first().click();

	await page.waitForURL('/settings/account');
});

test('Sign in with expired login code fails', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => t.expired)[0];
	await page.goto(`/lc/${token.token}`);

	await expect(page.getByRole('paragraph')).toHaveText(
		'Token is invalid or expired. Please try again.'
	);
});

test('Sign in with login code entered manually fails', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => t.expired)[0];
	await page.goto('/lc');

	await page.getByPlaceholder('Code').first().fill(token.token);

	await page.getByText('Submit').first().click();

	await expect(page.getByRole('paragraph')).toHaveText(
		'Token is invalid or expired. Please try again.'
	);
});
