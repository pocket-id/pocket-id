import test, { expect } from '@playwright/test';
import { emailVerificationTokens, users } from '../data';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

test('Update account details', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Display Name').fill('Tim Apple');
	await page.getByLabel('First name').fill('Timothy');
	await page.getByLabel('Last name').fill('Apple');
	await page.getByLabel('Display Name').fill('Timothy Apple');
	await page.getByLabel('Email').fill('timothy.apple@test.com');
	await page.getByLabel('Username').fill('timothy');
	await saveUnsavedChanges(page);
});

test('Failed account update remains dirty and can be retried', async ({ page }) => {
	await page.goto('/settings/account');

	let failedUpdates = 0;
	await page.route('**/api/users/me', async (route) => {
		if (route.request().method() !== 'PUT' || failedUpdates > 0) {
			await route.fallback();
			return;
		}

		failedUpdates++;
		await route.fulfill({
			status: 500,
			contentType: 'application/json',
			body: JSON.stringify({ error: 'Temporary account update failure' })
		});
	});

	const displayName = page.getByLabel('Display Name');
	await displayName.fill('Retryable Account');
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(page.getByText('Temporary account update failure', { exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeVisible();
	expect(failedUpdates).toBe(1);

	await saveUnsavedChanges(page);
	await page.reload();
	await expect(page.getByLabel('Display Name')).toHaveValue('Retryable Account');
});

test('Update account details fails with already taken email', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Email').fill(users.craig.email);

	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(page.getByText('Email is already in use', { exact: true })).toBeVisible();
});

test('Update account details fails with already taken username', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Username').fill(users.craig.username);

	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(page.getByText('Username is already in use', { exact: true })).toBeVisible();
});

test('Update account details fails with already taken username in different casing', async ({
	page
}) => {
	await page.goto('/settings/account');

	await page.getByLabel('Username').fill(users.craig.username.toUpperCase());

	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(page.getByText('Username is already in use', { exact: true })).toBeVisible();
});

test('Change Locale', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Select Locale').click();
	await page.getByRole('option', { name: 'Nederlands' }).click();

	// Check if th language heading now says 'Taal' instead of 'Language'
	await expect(page.getByText('Taal', { exact: true })).toBeVisible();

	// Check if the validation messages are translated because they are provided by Zod
	await page.getByRole('textbox', { name: 'Gebruikersnaam' }).fill('');
	await page.getByRole('button', { name: 'Opslaan' }).click();
	await expect(page.getByText('Te kort: verwacht dat string >=1 tekens heeft')).toBeVisible();

	// Clear all cookies and sign in again to check if the language is still set to Dutch
	await page.context().clearCookies();
	await authUtil.authenticate(page);

	await expect(page.getByText('Taal', { exact: true })).toBeVisible();

	await page.getByRole('textbox', { name: 'Gebruikersnaam' }).fill('');
	await page.getByRole('button', { name: 'Opslaan' }).click();
	await expect(page.getByText('Te kort: verwacht dat string >=1 tekens heeft')).toBeVisible();
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
	await page.getByLabel('Delete Passkey').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Passkey deleted successfully');
});

test('Email verification succeeds', async ({ page, context }) => {
	await context.clearCookies();

	const token = emailVerificationTokens.find((t) => !t.expired)!.token;
	await page.goto(`/verify-email?token=${token}`);
	await (await passkeyUtil.init(page)).addPasskey('craig');

	await page.getByRole('button', { name: 'Authenticate' }).click();
	await page.waitForURL('/settings/account?emailVerificationState=success');

	await expect(page.getByText('Email Verified Successfully')).toBeVisible();
});

test('Email verification fails with expired token', async ({ page, context }) => {
	await context.clearCookies();

	const token = emailVerificationTokens.find((t) => t.expired)!.token;
	await page.goto(`/verify-email?token=${token}`);
	await (await passkeyUtil.init(page)).addPasskey('craig');

	await page.getByRole('button', { name: 'Authenticate' }).click();
	await page.waitForURL(
		'/settings/account?emailVerificationState=Email+verification+token+is+invalid'
	);

	await expect(page.getByText('Email verification token is invalid')).toBeVisible();
});
