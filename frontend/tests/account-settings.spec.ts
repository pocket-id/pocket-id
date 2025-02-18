import test, { expect } from '@playwright/test';
import { users } from './data';
import { cleanupBackend } from './utils/cleanup.util';
import passkeyUtil from './utils/passkey.util';

test.beforeEach(cleanupBackend);

test('Update account details', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('First name').fill('Timothy');
	await page.getByLabel('Last name').fill('Apple');
	await page.getByLabel('Email').fill('timothy.apple@test.com');
	await page.getByLabel('Username').fill('timothy');
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('Account details updated successfully');
});

test('Update account details fails with already taken email', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Email').fill(users.craig.email);

	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('Email is already in use');
});

test('Update account details fails with already taken username', async ({ page }) => {
	await page.goto('/settings/account');

	await page.getByLabel('Username').fill(users.craig.username);

	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('Username is already in use');
});

test('Add passkey to an account', async ({ page }) => {
	await page.goto('/settings/account');

	await (await passkeyUtil.init(page)).addPasskey('timNew');

	await page.click('button:text("Add Passkey")');

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

	await expect(page.getByRole('status')).toHaveText('Passkey deleted successfully');
});
