import { expect, test } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async ({ page }) => {
	await cleanupBackend();
	await page.goto('/settings/admin/application-configuration');
});

test('Update general configuration', async ({ page }) => {
	await page.getByLabel('Application Name', { exact: true }).fill('Updated Name');
	await page.getByLabel('Session Duration').fill('30');
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Application configuration updated successfully'
	);
	await expect(page.getByTestId('application-name')).toHaveText('Updated Name');

	await page.reload();

	await expect(page.getByLabel('Application Name', { exact: true })).toHaveValue('Updated Name');
	await expect(page.getByLabel('Session Duration')).toHaveValue('30');
});

test.describe('Update user creation configuration', () => {
	test.beforeEach(async ({ page }) => {
		await page.getByRole('button', { name: 'Expand card' }).nth(1).click();
	});

	test('should save sign up mode', async ({ page }) => {
		await page.getByRole('button', { name: 'Enable User Signups' }).click();
		await page.getByRole('option', { name: 'Open Signup' }).click();

		await page.getByRole('button', { name: 'Save' }).nth(1).click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'User creation settings updated successfully.'
		);

		await page.reload();

		await expect(page.getByRole('button', { name: 'Enable User Signups' })).toBeVisible();
	});

	test('should save default user groups for new signups', async ({ page }) => {
		await page.getByRole('combobox', { name: 'User Groups' }).click();
		await page.getByRole('option', { name: 'Developers' }).click();
		await page.getByRole('option', { name: 'Designers' }).click();

		await page.getByRole('button', { name: 'Save' }).nth(1).click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'User creation settings updated successfully.'
		);

		await page.reload();

		await page.getByRole('combobox', { name: 'User Groups' }).click();

		await expect(page.getByRole('option', { name: 'Developers' })).toBeChecked();
		await expect(page.getByRole('option', { name: 'Designers' })).toBeChecked();
	});

	test('should save default custom claims for new signups', async ({ page }) => {
		await page.getByRole('button', { name: 'Add custom claim' }).click();
		await page.getByPlaceholder('Key').fill('test-claim');
		await page.getByPlaceholder('Value').fill('test-value');
		await page.getByRole('button', { name: 'Add another' }).click();
		await page.getByPlaceholder('Key').nth(1).fill('another-claim');
		await page.getByPlaceholder('Value').nth(1).fill('another-value');

		await page.getByRole('button', { name: 'Save' }).nth(1).click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'User creation settings updated successfully.'
		);

		await page.reload();

		await expect(page.getByPlaceholder('Key').first()).toHaveValue('test-claim');
		await expect(page.getByPlaceholder('Value').first()).toHaveValue('test-value');
		await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('another-claim');
		await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('another-value');
	});
});

test('Update email configuration', async ({ page }) => {
	await page.getByRole('button', { name: 'Expand card' }).nth(2).click();

	await page.getByLabel('SMTP Host').fill('smtp.gmail.com');
	await page.getByLabel('SMTP Port').fill('587');
	await page.getByLabel('SMTP User').fill('test@gmail.com');
	await page.getByLabel('SMTP Password').fill('password');
	await page.getByLabel('SMTP From').fill('test@gmail.com');
	await page.getByLabel('Email Login Notification').click();
	await page.getByLabel('Email Login Code Requested by User').click();
	await page.getByLabel('Email Login Code from Admin').click();
	await page.getByLabel('API Key Expiration').click();

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Email configuration updated successfully'
	);

	await page.reload();

	await expect(page.getByLabel('SMTP Host')).toHaveValue('smtp.gmail.com');
	await expect(page.getByLabel('SMTP Port')).toHaveValue('587');
	await expect(page.getByLabel('SMTP User')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('SMTP Password')).toHaveValue('password');
	await expect(page.getByLabel('SMTP From')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('Email Login Notification')).toBeChecked();
	await expect(page.getByLabel('Email Login Code Requested by User')).toBeChecked();
	await expect(page.getByLabel('Email Login Code from Admin')).toBeChecked();
	await expect(page.getByLabel('API Key Expiration')).toBeChecked();
});

test('Update application images', async ({ page }) => {
	await page.getByRole('button', { name: 'Expand card' }).nth(4).click();

	await page.getByLabel('Favicon').setInputFiles('assets/w3-schools-favicon.ico');
	await page.getByLabel('Light Mode Logo').setInputFiles('assets/pingvin-share-logo.png');
	await page.getByLabel('Dark Mode Logo').setInputFiles('assets/nextcloud-logo.png');
	await page.getByLabel('Background Image').setInputFiles('assets/clouds.jpg');
	await page.getByRole('button', { name: 'Save' }).last().click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Images updated successfully');

	await page.request
		.get('/api/application-configuration/favicon')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/logo?light=true')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/logo?light=false')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/background-image')
		.then((res) => expect.soft(res.status()).toBe(200));
});

test.describe('Allow Uppercase Usernames toggle', () => {
	test('rejects uppercase usernames when disabled (admin create user)', async ({ page }) => {
		// Ensure toggle is OFF
		const toggle = page.getByLabel('Allow Uppercase Usernames');
		if (await toggle.isChecked()) {
			await toggle.click();
		}

		// Save general configuration
		await page.getByRole('button', { name: 'Save' }).first().click();
		await expect(page.locator('[data-type="success"]')).toHaveText('Application configuration updated successfully');

		// Go to Users page and open Add User form
		await page.goto('/settings/admin/users');
		await page.getByRole('button', { name: 'Add User' }).click();

		// Try to create a user with uppercase in username
		await page.getByLabel('First name').fill('Upper');
		await page.getByLabel('Last name').fill('Case');
		await page.getByLabel('Email').fill('upper.case@test.com');
		await page.getByLabel('Username').fill('UpperUser');
		await page.getByRole('button', { name: 'Save' }).click();

		// Expect client-side validation error message
		await expect(
			page.getByText("Username can only contain letters, numbers, underscores, dots, hyphens, and '@' symbols")
		).toBeVisible();
	});

	test('allows uppercase usernames when enabled and persists setting', async ({ page }) => {
		// Enable toggle
		const toggle = page.getByLabel('Allow Uppercase Usernames');
		if (!(await toggle.isChecked())) {
			await toggle.click();
		}

		// Save general configuration
		await page.getByRole('button', { name: 'Save' }).first().click();
		await expect(page.locator('[data-type="success"]')).toHaveText('Application configuration updated successfully');

		// Verify persistence after reload
		await page.reload();
		await expect(page.getByLabel('Allow Uppercase Usernames')).toBeChecked();

		// Go to Users page and open Add User form
		await page.goto('/settings/admin/users');
		await page.getByRole('button', { name: 'Add User' }).click();

		// Create a user with uppercase in username
		await page.getByLabel('First name').fill('Upper');
		await page.getByLabel('Last name').fill('Allowed');
		await page.getByLabel('Email').fill('upper.allowed@test.com');
		await page.getByLabel('Username').fill('UpperUser');
		await page.getByRole('button', { name: 'Save' }).click();

		// Expect success toast and that the user is listed with uppercase preserved
		await expect(page.getByText('User created successfully')).toBeVisible();
		await expect(page.getByRole('cell', { name: 'UpperUser' })).toBeVisible();
	});
});
