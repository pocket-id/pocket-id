import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';
import { unsavedChangesBar } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

test('main navigation has an accessible name', async ({ page }) => {
	await page.goto('/settings/account');

	const nav = page.getByRole('navigation', { name: 'Main' }).first();
	await expect(nav).toBeVisible();
});

test('keyboard focus stays on sidebar link after navigating', async ({ page }) => {
	await page.goto('/settings/account');

	const activity = page.getByRole('link', { name: 'Activity' }).first();
	await activity.focus();
	await page.keyboard.press('Enter');

	await page.waitForURL('**/settings/audit-log');
	await expect(activity).toBeFocused();
});

test('unsaved changes block navigation until they are discarded', async ({ page }) => {
	await page.goto('/settings/account');

	const displayName = page.getByLabel('Display Name');
	const originalDisplayName = await displayName.inputValue();
	await displayName.fill('Pending navigation');
	await expect(unsavedChangesBar(page)).toContainText(/\d+ unsaved changes?/);

	const myApps = page.getByRole('link', { name: 'My Apps' }).first();
	await myApps.click();
	await expect(page).toHaveURL(/\/settings\/account$/);

	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(displayName).toHaveValue(originalDisplayName);
	await expect(unsavedChangesBar(page)).toHaveCount(0);

	await myApps.click();
	await expect(page).toHaveURL(/\/settings\/apps$/);
	await expect(unsavedChangesBar(page)).toHaveCount(0);
});
