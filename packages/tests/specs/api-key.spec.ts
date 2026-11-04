// frontend/tests/api-key.spec.ts
import { expect, type Page, test } from '@playwright/test';
import { apiKeys } from '../data';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';

test.describe('API Key Management', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await page.goto('/settings/account/api-keys');
	});

	test('Create new API key', async ({ page }) => {
		await page.getByRole('button', { name: 'Create API key' }).click();

		// Fill out the API key form
		const name = 'New Test API Key';
		await page.getByLabel('Name').fill(name);
		await page.getByLabel('Description').fill('Created by automated test');

		const expectedDate = getDefaultApiKeyExpirationDate();
		await expect(page.getByRole('button', { name: 'Select a date' })).toHaveText(
			expectedDate.toLocaleDateString('en-US', {
				year: 'numeric',
				month: 'long',
				day: 'numeric'
			})
		);

		await page.getByRole('dialog').getByRole('button', { name: 'Create API key' }).click();

		// The sheet turns into the one-time reveal of the new key
		const sheet = page.getByRole('dialog', { name: 'API key created' });
		await expect(sheet).toBeVisible();
		await expect(sheet.getByText(name)).toBeVisible();
		const token = await sheet.getByRole('textbox', { name: 'API key' }).textContent();
		expect(token?.trim().length).toBe(32);

		await sheet.getByRole('button', { name: 'Done' }).click();

		await page.reload();

		await expect(page.getByRole('row', { name })).toContainText('Expires in 30 days');
		await expect(page.getByRole('row', { name })).toContainText('Full access');
	});

	test('Scoped API key only reaches its scopes', async ({ page, playwright, baseURL }) => {
		await page.getByRole('button', { name: 'Create API key' }).click();
		await page.getByLabel('Name').fill('Read users');
		await page.getByRole('tab', { name: 'Custom' }).click();
		const users = page.getByRole('radiogroup', { name: 'Users', exact: true });

		// Writing users lets a key issue a sign-in code for its owner, which undoes every other limit
		await users.getByRole('radio', { name: 'Write' }).click();
		await expect(page.getByText('This key can gain full access')).toBeVisible();
		await users.getByRole('radio', { name: 'Read' }).click();
		await expect(page.getByText('This key can gain full access')).toHaveCount(0);

		await page.getByRole('dialog').getByRole('button', { name: 'Create API key' }).click();

		const sheet = page.getByRole('dialog', { name: 'API key created' });
		await expect(sheet).toContainText('Custom (1)');
		const token = (await sheet.getByRole('textbox', { name: 'API key' }).textContent())!.trim();

		// A context without the session cookie, so the key is the only credential
		const api = await playwright.request.newContext({
			baseURL,
			storageState: { cookies: [], origins: [] },
			extraHTTPHeaders: { 'X-API-Key': token }
		});
		expect((await api.get('/api/users')).status()).toBe(200);
		expect((await api.get('/api/oidc/clients')).status()).toBe(403);
		await api.dispose();
	});

	test('Edit API key permissions', async ({ page }) => {
		const apiKey = apiKeys[0];
		const row = page.getByRole('row', { name: apiKey.name });
		await expect(row).toContainText('Full access');

		await row.getByRole('button', { name: /^Actions for / }).click();
		await page.getByRole('menuitem', { name: 'Edit' }).click();

		const sheet = page.getByRole('dialog', { name: 'Edit API key' });
		await sheet.getByRole('tab', { name: 'Read only' }).click();
		await sheet.getByRole('button', { name: 'Save' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText('API key updated');
		await expect(row).toContainText('Read only');
	});

	test('Non-admins can only grant account permissions', async ({ page }) => {
		await authUtil.changeUser(page, 'craig');
		await page.goto('/settings/account#developers');
		await page.getByRole('link', { name: 'Manage' }).click();

		await page.getByRole('button', { name: 'Create API key' }).click();
		const sheet = page.getByRole('dialog', { name: 'Create API key' });
		await sheet.getByRole('tab', { name: 'Custom' }).click();

		await expect(sheet.getByRole('heading', { name: 'Your account' })).toBeVisible();
		await expect(sheet.getByRole('radiogroup', { name: 'Passkeys' })).toBeVisible();
		await expect(sheet.getByRole('heading', { name: 'Administration' })).toHaveCount(0);
		await expect(sheet.getByRole('radiogroup', { name: 'Users', exact: true })).toHaveCount(0);
	});

	test('Renew API key', async ({ page }) => {
		const apiKey = apiKeys[1];

		await page
			.getByRole('row', { name: apiKey.name })
			.getByRole('button', { name: /^Actions for / })
			.click();

		await page.getByRole('menuitem', { name: 'Renew' }).click();

		const expectedDate = getDefaultApiKeyExpirationDate();
		await selectApiKeyExpirationDate(page, expectedDate);
		await expect(page.getByRole('button', { name: 'Select a date' })).toHaveText(
			expectedDate.toLocaleDateString('en-US', {
				year: 'numeric',
				month: 'long',
				day: 'numeric'
			})
		);

		await page.getByRole('button', { name: 'Renew key' }).click();

		await expect(page.getByRole('heading', { name: 'API key renewed' })).toBeVisible();
		await page.getByRole('button', { name: 'Done' }).click();

		await expect(page.getByRole('row', { name: apiKey.name })).toContainText('Expires in 30 days');
	});

	test('Rotate API key with a grace period', async ({ page, playwright, baseURL }) => {
		await page.getByRole('button', { name: 'Create API key' }).click();
		await page.getByLabel('Name').fill('Rotating key');
		await page.getByRole('dialog').getByRole('button', { name: 'Create API key' }).click();
		const createdSheet = page.getByRole('dialog', { name: 'API key created' });
		const oldToken = (await createdSheet
			.getByRole('textbox', { name: 'API key' })
			.textContent())!.trim();
		await createdSheet.getByRole('button', { name: 'Done' }).click();

		await page
			.getByRole('row', { name: 'Rotating key' })
			.getByRole('button', { name: /^Actions for / })
			.click();
		await page.getByRole('menuitem', { name: 'Rotate' }).click();
		await expect(page.getByLabel('Grace period')).toHaveText('1 day');
		await page.getByRole('button', { name: 'Rotate key' }).click();

		const dialog = page.getByRole('dialog', { name: 'API key rotated' });
		await expect(dialog).toContainText('Old key works until');
		const newToken = (await dialog.getByRole('textbox', { name: 'API key' }).textContent())!.trim();
		expect(newToken).not.toBe(oldToken);
		await dialog.getByRole('button', { name: 'Done' }).click();

		await expect(page.getByRole('row', { name: 'Rotating key' })).toContainText(
			'Old key works until'
		);

		// Both keys work during the grace period
		for (const token of [oldToken, newToken]) {
			const api = await playwright.request.newContext({
				baseURL,
				storageState: { cookies: [], origins: [] },
				extraHTTPHeaders: { 'X-API-Key': token }
			});
			expect((await api.get('/api/users/me')).status()).toBe(200);
			await api.dispose();
		}
	});

	test('Revoke API key', async ({ page }) => {
		const apiKey = apiKeys[0];

		await page
			.getByRole('row', { name: apiKey.name })
			.getByRole('button', { name: /^Actions for / })
			.click();

		await page.getByRole('menuitem', { name: 'Revoke' }).click();

		await page.getByRole('alertdialog').getByRole('button', { name: 'Revoke' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText(`${apiKey.name} revoked`);

		// Verify key is no longer in the list
		await expect(page.getByRole('row', { name: apiKey.name })).not.toBeVisible();
	});
});

function getDefaultApiKeyExpirationDate() {
	const date = new Date();
	date.setDate(date.getDate() + 30);
	date.setHours(0, 0, 0, 0);
	return date;
}

async function selectApiKeyExpirationDate(page: Page, date: Date) {
	await page.getByRole('button', { name: 'Select a date' }).click();

	const targetDay = page.getByRole('button', { name: getCalendarDayName(date) });
	const nextButton = page.getByRole('button', { name: 'Next', exact: true });

	// The calendar opens on the month of the field's current value, which can be an
	// arbitrary number of months away from the target date (e.g. when renewing an
	// already-expired key, whose proposed date is in the past). Advance one month at
	// a time until the target day is rendered, rather than assuming it is exactly one
	// month ahead — that assumption is date-dependent and fails on most days.
	await expect(async () => {
		if (!(await targetDay.isVisible())) {
			await nextButton.click();
		}
		await expect(targetDay).toBeVisible({ timeout: 500 });
	}).toPass({ timeout: 8000 });

	await targetDay.click();
}

function getCalendarDayName(date: Date) {
	return new Intl.DateTimeFormat('en-US', {
		weekday: 'long',
		month: 'long',
		day: 'numeric',
		year: 'numeric'
	}).format(date);
}
