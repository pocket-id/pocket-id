import test, { expect } from '@playwright/test';
import { userGroups, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(() => cleanupBackend());

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
	await expect(page.locator('[data-type="success"]')).toHaveText('User created successfully');
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

	await expect(page.locator('[data-type="error"]')).toHaveText('Email is already in use');
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

	await expect(page.locator('[data-type="error"]')).toHaveText('Username is already in use');
});

test('Create one time access token', async ({ page, context }) => {
	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', {
			name: `${users.craig.firstname} ${users.craig.lastname}`
		})
		.getByRole('button')
		.click();

	await page.getByRole('menuitem', { name: 'Login Code' }).click();

	await page.getByLabel('Expiration').click();
	await page.getByRole('option', { name: '12 hours' }).click();
	await page.getByRole('button', { name: 'Show Code' }).click();

	const link = await page.getByTestId('login-code-link').textContent();
	await context.clearCookies();

	await page.goto(link!);
	await page.waitForURL('/settings/account');
});

test('Delete user', async ({ page }) => {
	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', {
			name: `${users.craig.firstname} ${users.craig.lastname}`
		})
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('User deleted successfully');
	await expect(
		page.getByRole('row', {
			name: `${users.craig.firstname} ${users.craig.lastname}`
		})
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

	await expect(page.locator('[data-type="success"]')).toHaveText('User updated successfully');
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

	await expect(page.locator('[data-type="error"]')).toHaveText('Email is already in use');
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

	await expect(page.locator('[data-type="error"]')).toHaveText('Username is already in use');
});

test('Update user custom claims', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);

	await page.getByRole('button', { name: 'Expand card' }).nth(1).click();

	// Add two custom claims
	await page.getByRole('button', { name: 'Add custom claim' }).click();

	await page.getByPlaceholder('Key').fill('customClaim1');
	await page.getByPlaceholder('Value').fill('customClaim1_value');

	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Key').nth(1).fill('customClaim2');
	await page.getByPlaceholder('Value').nth(1).fill('customClaim2_value');

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Custom claims updated successfully'
	);

	await page.reload();

	// Check if custom claims are saved
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim1');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim1_value');
	await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('customClaim2_value');

	// Remove one custom claim
	await page.getByLabel('Remove custom claim').first().click();
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Custom claims updated successfully'
	);

	await page.reload();

	// Check if custom claim is removed
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim2_value');
});

test('Update user group assignments', async ({ page }) => {
	const user = users.craig;
	await page.goto(`/settings/admin/users/${user.id}`);

	page.getByRole('button', { name: 'Expand card' }).first().click();

	await page.getByRole('row', { name: userGroups.developers.name }).getByRole('checkbox').click();
	await page.getByRole('row', { name: userGroups.designers.name }).getByRole('checkbox').click();

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'User groups updated successfully'
	);

	await page.reload();

	await expect(
		page.getByRole('row', { name: userGroups.designers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked');
	await expect(
		page.getByRole('row', { name: userGroups.developers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'unchecked');
});
