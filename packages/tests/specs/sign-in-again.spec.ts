import test, { expect, type Page } from '@playwright/test';
import { users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';
import { unsavedChangesBar } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

// Ends the session on the server while the page still thinks it's signed in, like a session that expired in the background
async function endSessionWithUnsavedChanges(page: Page, passkey: 'tim' | 'craig') {
	await page.goto('/settings/account');
	const displayName = page.getByLabel('Display Name');
	await expect(displayName).toBeVisible();
	await (await passkeyUtil.init(page)).addPasskey(passkey);
	await page.context().request.post('/api/webauthn/logout');

	await displayName.fill('Saved after signing in again');
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	const dialog = page.getByRole('alertdialog');
	await expect(dialog.getByText("You've been signed out")).toBeVisible();
	await expect(dialog.getByText(users.tim.email)).toBeVisible();
	return dialog;
}

test('Signing in again saves the changes made after the session ended', async ({ page }) => {
	const dialog = await endSessionWithUnsavedChanges(page, 'tim');
	await dialog.getByRole('button', { name: 'Sign in with passkey' }).click();

	await expect(unsavedChangesBar(page).getByText('Changes saved', { exact: true })).toBeVisible();
	await expect(dialog).toHaveCount(0);

	await page.reload();
	await expect(page.getByLabel('Display Name')).toHaveValue('Saved after signing in again');
});

test('Signing in again as another account starts over without saving', async ({ page }) => {
	const dialog = await endSessionWithUnsavedChanges(page, 'craig');
	await Promise.all([
		page.waitForEvent('load'),
		dialog.getByRole('button', { name: 'Sign in with passkey' }).click()
	]);

	await page.goto('/settings/account');
	await expect(page.getByLabel('Email')).toHaveValue(users.craig.email);
	await expect(page.getByLabel('Display Name')).toHaveValue(users.craig.displayName);
});
