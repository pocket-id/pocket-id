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
