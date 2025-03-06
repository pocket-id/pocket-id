import test, { expect } from '@playwright/test';
import { users, userGroups } from './data';
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

	await page.getByLabel('One Time Link').getByRole('combobox').click();
	await page.getByRole('option', { name: '12 hours' }).click();
	await page.getByRole('button', { name: 'Generate Link' }).click();

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

	await page.getByRole('button', { name: 'Expand card' }).nth(1).click();

	// Add two custom claims
	await page.getByRole('button', { name: 'Add custom claim' }).click();

	await page.getByPlaceholder('Key').fill('customClaim1');
	await page.getByPlaceholder('Value').fill('customClaim1_value');

	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Key').nth(1).fill('customClaim2');
	await page.getByPlaceholder('Value').nth(1).fill('customClaim2_value');

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.getByRole('status')).toHaveText('Custom claims updated successfully');

	await page.reload();

	// Check if custom claims are saved
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim1');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim1_value');
	await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('customClaim2_value');

	// Remove one custom claim
	await page.getByLabel('Remove custom claim').first().click();
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.getByRole('status')).toHaveText('Custom claims updated successfully');

	await page.reload();

	// Check if custom claim is removed
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim2_value');
});

test('Update user group assignments', async ({ page }) => {
	const user = users.craig;
	await page.goto(`/settings/admin/users/${user.id}`);

	// Increase the test timeout since this test is complex
	test.setTimeout(30000);

	// Expand the user groups section if it's collapsed
	const expandButton = page.getByRole('button', { name: 'Expand card' }).first();
	if (await expandButton.isVisible()) {
		await expandButton.click();
	}

	// Wait for the user groups table to load
	await page.waitForSelector('table');

	// First, ensure we start with a clean state - uncheck any checked boxes
	const developersCheckbox = page
		.getByRole('row', { name: userGroups.developers.name })
		.getByRole('checkbox');
	const designersCheckbox = page
		.getByRole('row', { name: userGroups.designers.name })
		.getByRole('checkbox');

	// Force click if needed to overcome element interception issues
	if ((await developersCheckbox.getAttribute('data-state')) === 'checked') {
		await developersCheckbox.click({ force: true });
	}

	if ((await designersCheckbox.getAttribute('data-state')) === 'checked') {
		await designersCheckbox.click({ force: true });
	}

	// Save the changes to reset state if needed
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	// Wait for toast message to appear and disappear
	await expect(page.getByRole('status')).toHaveText('User groups updated successfully');
	await page.waitForTimeout(1000); // Wait for any animations or state changes

	// Now add both groups (using force: true to avoid interception problems)
	await developersCheckbox.click({ force: true });
	await designersCheckbox.click({ force: true });

	// Save the changes
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	// Verify success message
	await expect(page.getByRole('status')).toHaveText('User groups updated successfully');

	await page.reload();

	await expect(
		page.getByRole('row', { name: userGroups.developers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked', { timeout: 10000 });
	await expect(
		page.getByRole('row', { name: userGroups.designers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked', { timeout: 10000 });
});
