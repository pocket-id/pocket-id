// frontend/tests/api-key.spec.ts
import { test, expect } from '@playwright/test';

test.describe('API Key Management', () => {
	test.beforeEach(async ({ page }) => {
		// Login as admin before each test
		// await loginAsAdmin(page);
		// Navigate to API keys page
		await page.goto('/settings/admin/api-keys');
	});

	test('Create new API key', async ({ page }) => {
		// Click the Add API Key button if it's collapsed
		const addButton = page.getByRole('button', { name: 'Add API Key' });
		if (await addButton.isVisible()) {
			await addButton.click();
		}

		// Fill out the API key form
		const name = 'Test API Key';
		await page.getByLabel('Name').fill(name);
		await page.getByLabel('Description').fill('Created by automated test');

		// Set expiration date to 30 days from now
		const futureDate = new Date();
		futureDate.setDate(futureDate.getDate() + 30);
		const formattedDate = futureDate.toISOString().split('T')[0];
		const formattedTime = futureDate.toISOString().split('T')[1].substring(0, 5);
		await page.getByLabel('Expires At').fill(`${formattedDate}T${formattedTime}`);

		// Submit the form
		await page.getByRole('button', { name: 'Generate API Key' }).click();

		// Verify the success dialog appears
		await expect(page.getByRole('heading', { name: 'API Key Created' })).toBeVisible();

		// Verify the key details are shown
		await expect(page.getByText(name)).toBeVisible();

		// Verify the token is displayed (should be 32 characters)
		const tokenElement = page.locator('.font-mono');
		await expect(tokenElement).toBeVisible();
		const token = await tokenElement.textContent();
		expect(token?.length).toBe(32);

		// Close the dialog
		await page
			.getByLabel('API Key Created')
			.locator('div')
			.filter({ hasText: 'Close' })
			.getByRole('button')
			.click();

		await page.reload();

		// Verify the key appears in the list
		await expect(page.getByRole('cell', { name }).first()).toBeVisible();
	});

	test('Revoke API key', async ({ page }) => {
		// First create a key
		// Click the Add API Key button
		const addButton = page.getByRole('button', { name: 'Add API Key' });
		if (await addButton.isVisible()) {
			await addButton.click();
		}

		// Fill out the API key form with a unique name
		const keyName = `Test API Key ${Date.now()}`;
		await page.getByLabel('Name').fill(keyName);

		// Set expiration date
		const futureDate = new Date();
		futureDate.setDate(futureDate.getDate() + 30);
		const formattedDate = futureDate.toISOString().split('T')[0];
		const formattedTime = futureDate.toISOString().split('T')[1].substring(0, 5);
		await page.getByLabel('Expires At').fill(`${formattedDate}T${formattedTime}`);

		// Submit the form and close the dialog
		await page.getByRole('button', { name: 'Generate API Key' }).click();

		// Wait for the dialog to appear
		await expect(page.getByRole('heading', { name: 'API Key Created' })).toBeVisible();

		// Close the dialog
		await page
			.getByLabel('API Key Created')
			.locator('div')
			.filter({ hasText: 'Close' })
			.getByRole('button')
			.click();

		// Wait for dialog to be closed
		await expect(page.getByRole('heading', { name: 'API Key Created' })).not.toBeVisible();
		await page.reload();

		// Wait for the table to load with our key
		await page.waitForSelector('table', { state: 'visible' });
		await expect(page.getByRole('cell', { name: keyName })).toBeVisible();

		// Get the row containing our key name
		const keyCell = page.getByRole('cell', { name: keyName }).first();
		const row = keyCell.locator('xpath=..'); // Go up to the parent row element

		// Click the actions menu button in this row (be more specific with the selector)
		const actionsButton = row.getByTestId('actions-dropdown');
		await actionsButton.click();

		// Click the revoke option
		await page.getByRole('menuitem', { name: 'Revoke' }).click();

		// Confirm revocation
		await page.getByRole('button', { name: 'Revoke' }).click();

		// Verify success message
		await expect(page.getByRole('status')).toHaveText('API key revoked successfully');

		// Wait a moment for the UI to update
		await page.waitForTimeout(500);

		// Verify key is no longer in the list
		await expect(page.getByRole('cell', { name: keyName })).not.toBeVisible();
	});

	test('Validate empty form errors', async ({ page }) => {
		// Click the Add API Key button if it's collapsed
		const addButton = page.getByRole('button', { name: 'Add API Key' });
		if (await addButton.isVisible()) {
			await addButton.click();
		}

		// Submit without filling any fields
		await page.getByRole('button', { name: 'Generate API Key' }).click();

		// Verify validation messages
		await expect(page.getByText('String must contain at least 3 character(s)')).toBeVisible();
	});

	test('API key list displays correctly', async ({ page }) => {
		// Check that the table headers are present
		await expect(page.getByRole('columnheader', { name: 'Name' })).toBeVisible();
		await expect(page.getByRole('columnheader', { name: 'Description' })).toBeVisible();
		await expect(page.getByRole('columnheader', { name: 'Expires At' })).toBeVisible();
		await expect(page.getByRole('columnheader', { name: 'Last Used' })).toBeVisible();
	});
});
