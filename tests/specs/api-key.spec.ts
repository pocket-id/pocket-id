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
