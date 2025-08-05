import { expect, test } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.describe('Signup Defaults Configuration', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await page.goto('/settings/admin/application-configuration');
		await page.getByRole('button', { name: 'Expand card' }).nth(1).click();
	});

	test('should save default user groups for new signups', async ({ page }) => {
		const signupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup Defaults")');

		await signupDefaultsCard.locator('[data-slot="dropdown-menu-trigger"]').click();
		await page.getByRole('menuitemcheckbox', { name: 'Developers' }).click();
		await page.getByRole('menuitemcheckbox', { name: 'Designers' }).click();
		await page.keyboard.press('Escape');

		await signupDefaultsCard.locator('form').getByRole('button', { name: 'Save' }).click();

  	await expect(page.locator('[data-type="success"]')).toHaveText(
      'Signup defaults updated successfully'
    );

		await page.reload();

		const updatedSignupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup Defaults")');
		await expect(updatedSignupDefaultsCard.locator('[data-slot="dropdown-menu-trigger"]')).toContainText('Developers');
		await expect(updatedSignupDefaultsCard.locator('[data-slot="dropdown-menu-trigger"]')).toContainText('Designers');
	});

	test('should save default custom claims for new signups', async ({ page }) => {
		const signupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup Defaults")');
		
		await signupDefaultsCard.getByRole('button', { name: 'Add custom claim' }).click();
		await signupDefaultsCard.getByPlaceholder('Key').fill('test-claim');
		await signupDefaultsCard.getByPlaceholder('Value').fill('test-value');
		await signupDefaultsCard.getByRole('button', { name: 'Add another' }).click();
		await signupDefaultsCard.getByPlaceholder('Key').nth(1).fill('another-claim');
		await signupDefaultsCard.getByPlaceholder('Value').nth(1).fill('another-value');

		await signupDefaultsCard.locator('form').getByRole('button', { name: 'Save' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText(
			'Signup defaults updated successfully'
		);

		await page.reload();

		const updatedSignupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup Defaults")');
		await expect(updatedSignupDefaultsCard.getByPlaceholder('Key').first()).toHaveValue('test-claim');
		await expect(updatedSignupDefaultsCard.getByPlaceholder('Value').first()).toHaveValue('test-value');
		await expect(updatedSignupDefaultsCard.getByPlaceholder('Key').nth(1)).toHaveValue('another-claim');
		await expect(updatedSignupDefaultsCard.getByPlaceholder('Value').nth(1)).toHaveValue('another-value');
	});
});
