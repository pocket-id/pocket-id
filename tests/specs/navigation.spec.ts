import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('settings sidebar has an accessible name', async ({ page }) => {
	await page.goto('/settings/account');

	const nav = page.getByRole('navigation', { name: 'Settings' });
	await expect(nav).toBeVisible();
});

test('keyboard focus stays on sidebar link after navigating', async ({ page }) => {
	await page.goto('/settings/account');

	const auditLog = page.getByRole('link', { name: 'Audit Log' });
	await auditLog.focus();
	await page.keyboard.press('Enter');

	await page.waitForURL('**/settings/audit-log');
	await expect(auditLog).toBeFocused();
});

test('unsaved changes block navigation until they are discarded', async ({ page }) => {
	await page.goto('/settings/account');

	const displayName = page.getByLabel('Display Name');
	const originalDisplayName = await displayName.inputValue();
	await displayName.fill('Pending navigation');
	await expect(page.getByText('You have unsaved changes', { exact: true })).toBeVisible();

	const myApps = page.getByRole('link', { name: 'My Apps' });
	await myApps.click();
	await expect(page).toHaveURL(/\/settings\/account$/);

	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(displayName).toHaveValue(originalDisplayName);
	await expect(page.getByText('You have unsaved changes', { exact: true })).toHaveCount(0);

	await myApps.click();
	await expect(page).toHaveURL(/\/settings\/apps$/);
	await expect(page.getByText('You have unsaved changes', { exact: true })).toHaveCount(0);
});
