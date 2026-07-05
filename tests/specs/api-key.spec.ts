// frontend/tests/api-key.spec.ts
import { expect, type Page, test } from '@playwright/test';
import { apiKeys } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.describe('API Key Management', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await page.goto('/settings/admin/api-keys');
	});

	test('Create new API key', async ({ page }) => {
		await page.getByRole('button', { name: 'Add API Key' }).click();

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

		// Submit the form
		await page.getByRole('button', { name: 'Save' }).click();

		// Verify the success dialog appears
		await expect(page.getByRole('heading', { name: 'API Key Created' })).toBeVisible();

		// Verify the key details are shown
		await expect(page.getByRole('cell', { name })).toBeVisible();

		// Verify the token is displayed (should be 32 characters)
		const token = await page.locator('.font-mono').textContent();
		expect(token?.length).toBe(32);

		// Close the dialog
		await page.getByRole('button', { name: 'Close', exact: true }).nth(1).click();

		await page.reload();

		// Verify the key appears in the list
		await expect(page.getByRole('cell', { name }).first()).toContainText(name);
		await expect(
			page.getByRole('row', { name }).getByRole('cell', { name: expectedDate.toLocaleString() })
		).toBeVisible();
	});

	test('Renew API key', async ({ page }) => {
		const apiKey = apiKeys[1];

		await page
			.getByRole('row', { name: apiKey.name })
			.getByRole('button', { name: 'Toggle menu' })
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

		await page.getByRole('button', { name: 'Renew' }).click();

		await expect(page.getByRole('heading', { name: 'API key renewed' })).toBeVisible();

		// Verify the new expiration date is shown
		const row = page.getByRole('row', { name: apiKey.name });
		await expect(
			row.getByRole('cell', { name: new RegExp(escapeRegExp(expectedDate.toLocaleDateString())) })
		).toBeVisible();
	});

	test('Revoke API key', async ({ page }) => {
		const apiKey = apiKeys[0];

		await page
			.getByRole('row', { name: apiKey.name })
			.getByRole('button', { name: 'Toggle menu' })
			.click();

		await page.getByRole('menuitem', { name: 'Revoke' }).click();

		await page.getByRole('button', { name: 'Revoke' }).click();

		// Verify success message
		await expect(page.locator('[data-type="success"]')).toHaveText('API key revoked successfully');

		// Verify key is no longer in the list
		await expect(page.getByRole('cell', { name: apiKey.name })).not.toBeVisible();
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

function escapeRegExp(value: string) {
	return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
