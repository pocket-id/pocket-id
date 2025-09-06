import test, { expect } from '@playwright/test';
import { users } from '../data';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => await cleanupBackend());

test('Update account details', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Display Name').fill('Tim Apple');
	await page.getByLabel('First name').fill('Timothy');
	await page.getByLabel('Last name').fill('Apple');
	await page.getByLabel('Email').fill('timothy.apple@test.com');
	await page.getByLabel('Username').fill('timothy');
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Account details updated successfully'
	);
});

test('Update account details fails with already taken email', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Email').fill(users.craig.email);

	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="error"]')).toHaveText('Email is already in use');
});

test('Update account details fails with already taken username', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Username').fill(users.craig.username);

	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="error"]')).toHaveText('Username is already in use');
});

test('Change Locale', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Select Locale').click();
	await page.getByRole('option', { name: 'Nederlands' }).click();

	// Check if th language heading now says 'Taal' instead of 'Language'
	await expect(page.getByText('Taal', { exact: true })).toBeVisible();

	// Check if the validation messages are translated because they are provided by Zod
	await page.getByRole('textbox', { name: 'Voornaam' }).fill('');
	await page.getByRole('button', { name: 'Opslaan' }).click();
	await expect(page.getByText('Te kort: verwacht dat string')).toBeVisible();

	// Clear all cookies and sign in again to check if the language is still set to Dutch
	await page.context().clearCookies();
	await authUtil.authenticate(page);

	await expect(page.getByText('Taal', { exact: true })).toBeVisible();

	await page.getByRole('textbox', { name: 'Voornaam' }).fill('');
	await page.getByRole('button', { name: 'Opslaan' }).click();
	await expect(page.getByText('Te kort: verwacht dat string')).toBeVisible();
});

test('Add passkey to an account', async ({ page }) => {
	await page.goto('/settings/account');

	await (await passkeyUtil.init(page)).addPasskey('timNew');

	await page.getByRole('button', { name: 'Add Passkey' }).click();

	await page.getByLabel('Name', { exact: true }).fill('Test Passkey');
	await page.getByLabel('Name Passkey').getByRole('button', { name: 'Save' }).click();

	await expect(page.getByText('Test Passkey')).toBeVisible();
});

test('Rename passkey', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Rename').first().click();

	await page.getByLabel('Name', { exact: true }).fill('Renamed Passkey');
	await page.getByLabel('Name Passkey').getByRole('button', { name: 'Save' }).click();

	await expect(page.getByText('Renamed Passkey')).toBeVisible();
});

test('Delete passkey from account', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Delete').first().click();
	await page.getByText('Delete', { exact: true }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Passkey deleted successfully');
});

test('Generate own one time access token as non admin', async ({ page, context }) => {
	await context.clearCookies();
	await page.goto('/login');
	await (await passkeyUtil.init(page)).addPasskey('craig');

	await page.getByRole('button', { name: 'Authenticate' }).click();
	await page.waitForURL('/settings/account');

	await page.getByRole('button', { name: 'Create' }).click();
	const link = await page.getByTestId('login-code-link').textContent();

	await context.clearCookies();

	await page.goto(link!);
	await page.waitForURL('/settings/account');
});
