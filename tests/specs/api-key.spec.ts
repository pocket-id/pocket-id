// frontend/tests/api-key.spec.ts
import { expect, Page, test } from '@playwright/test';
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

		// Choose the date
		const currentDate = new Date();
		await selectDate(page, currentDate.getFullYear() + 1, currentDate.getMonth(), 1);

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
	});

	test('Renew API key', async ({ page }) => {
		const apiKey = apiKeys[1];

		await page
			.getByRole('row', { name: apiKey.name })
			.getByRole('button', { name: 'Toggle menu' })
			.click();

		await page.getByRole('menuitem', { name: 'Renew' }).click();

		// Choose the date
		const currentDate = new Date();
		await selectDate(page, currentDate.getFullYear() + 1, currentDate.getMonth(), 1);

		await page.getByRole('button', { name: 'Renew' }).click();

		await expect(page.getByRole('heading', { name: 'API key renewed' })).toBeVisible();

		// Verify the new expiration date is shown
		const row = page.getByRole('row', { name: apiKey.name });
		const expectedDate = new Date(currentDate.getFullYear() + 1, currentDate.getMonth(), 1);
		await expect(row.getByRole('cell', { name: expectedDate.toLocaleString() })).toBeVisible();
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

async function selectDate(page: Page, year: number, month: number, day: number) {
	// Open the date picker
	await page.getByRole('button', { name: 'Select a date' }).click();
	// Select the year
	await page.getByLabel('Select year').click();
	await page.getByRole('option', { name: year.toString() }).click();
	// Select the month and day
	const monthNames = [
		'January',
		'February',
		'March',
		'April',
		'May',
		'June',
		'July',
		'August',
		'September',
		'October',
		'November',
		'December'
	];
	const monthName = monthNames[month];
	await page.getByRole('button', { name: 'Select month' }).click();
	await page.getByRole('option', { name: monthName }).click();

	await page
		.getByRole('button', { name: new RegExp(`/([A-Z][a-z]+), ([A-Z][a-z]+) ${day}, (\\d{4})/`) }).first()
		.click();
}
