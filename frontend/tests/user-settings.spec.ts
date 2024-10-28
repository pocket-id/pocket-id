import test, { expect } from '@playwright/test';
import { users } from './data';
import { cleanupBackend } from './utils/cleanup.util';

test.beforeEach(cleanupBackend);

test('Create user', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add User' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(user.username);
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.getByRole('row', { name: `${user.firstname} ${user.lastname}` })).toBeVisible();
	await expect(page.getByRole('status')).toHaveText('User created successfully');
});

test('Create user fails with already taken email', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add User' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(users.tim.email);
	await page.getByLabel('Username').fill(user.username);
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.getByRole('status')).toHaveText('Email is already in use');
});

test('Create user fails with already taken username', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add User' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(users.tim.username);
	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.getByRole('status')).toHaveText('Username is already in use');
});

test('Create one time access token', async ({ page }) => {
	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${users.craig.firstname} ${users.craig.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'One-time link' }).click();

	await expect(page.getByRole('textbox', { name: 'One Time Link' })).toHaveValue(
		/http:\/\/localhost\/login\/.*/
	);
});

test('Delete user', async ({ page }) => {
	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${users.craig.firstname} ${users.craig.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.getByRole('status')).toHaveText('User deleted successfully');
	await expect(
		page.getByRole('row', { name: `${users.craig.firstname} ${users.craig.lastname}` })
	).not.toBeVisible();
});

test('Update user', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Edit' }).click();

	await page.getByLabel('First name').fill('Crack');
	await page.getByLabel('Last name').fill('Apple');
	await page.getByLabel('Email').fill('crack.apple@test.com');
	await page.getByLabel('Username').fill('crack');
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('User updated successfully');
});

test('Update user fails with already taken email', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Edit' }).click();

	await page.getByLabel('Email').fill(users.tim.email);
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('Email is already in use');
});

test('Update user fails with already taken username', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Edit' }).click();

	await page.getByLabel('Username').fill(users.tim.username);
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText('Username is already in use');
});

test('Update user custom claims', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);

	// Add two custom claims
	await page.getByRole('button', { name: 'Add custom claim' }).click();

	await page.getByPlaceholder('Key').fill('custom_claim_1');
	await page.getByPlaceholder('Value').fill('custom_claim_1_value');

	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Key').nth(1).fill('custom_claim_2');
	await page.getByPlaceholder('Value').nth(1).fill('custom_claim_2_value');

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.getByRole('status')).toHaveText('Custom claims updated successfully');

	await page.reload();

	// Check if custom claims are saved
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('custom_claim_1');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('custom_claim_1_value');
	await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('custom_claim_2');
	await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('custom_claim_2_value');

	// Remove one custom claim
	await page.getByLabel('Remove custom claim').first().click();
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await page.reload();

	// Check if custom claim is removed
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('custom_claim_2');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('custom_claim_2_value');
});
