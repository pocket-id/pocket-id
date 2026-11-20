import test, { expect } from '@playwright/test';
import { userGroups, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { saveUnsavedChanges, unsavedChangesBar } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

test('Create user', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add user' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(user.username);

	await expect(page.getByLabel('Display name')).toHaveValue(`${user.firstname} ${user.lastname}`);

	await page.getByRole('dialog').getByRole('button', { name: 'Add user' }).click();

	// A new person lands on their own page, ready for a login code
	await expect(page.locator('[data-type="success"]')).toHaveText(
		`${user.firstname} ${user.lastname} added`
	);
	await expect(
		page.getByRole('heading', { level: 1, name: `${user.firstname} ${user.lastname}` })
	).toBeVisible();
});

test('Create user fails with already taken email', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add user' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(users.tim.email);
	await page.getByLabel('Username').fill(user.username);
	await page.getByRole('dialog').getByRole('button', { name: 'Add user' }).click();

	await expect(page.locator('[data-type="error"]')).toHaveText('Email is already in use');
});

test('Create user fails with already taken username', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add user' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(users.tim.username);
	await page.getByRole('dialog').getByRole('button', { name: 'Add user' }).click();

	await expect(page.locator('[data-type="error"]')).toHaveText('Username is already in use');
});

test('Create user fails with already taken username in different casing', async ({ page }) => {
	const user = users.steve;

	await page.goto('/settings/admin/users');

	await page.getByRole('button', { name: 'Add user' }).click();
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(users.tim.username.toUpperCase());
	await page.getByRole('dialog').getByRole('button', { name: 'Add user' }).click();

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

	await page.getByRole('menuitem', { name: 'Login code' }).click();

	await page.getByLabel('Expires after').click();
	await page.getByRole('option', { name: '12 hours' }).click();
	await page.getByRole('button', { name: 'Create code' }).click();

	const code = await page.getByTestId('login-code').textContent();
	await context.clearCookies();

	await page.goto(`/lc/${code!.trim()}`);
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
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		`${users.craig.firstname} ${users.craig.lastname} deleted`
	);
	await expect(
		page.getByRole('row', {
			name: `${users.craig.firstname} ${users.craig.lastname}`
		})
	).not.toBeVisible();
});

test('Delete user from the details page', async ({ page }) => {
	await page.goto('/settings/admin/users');
	await page.getByRole('row', { name: `${users.craig.firstname} ${users.craig.lastname}` }).click();
	await page.waitForURL(`/settings/admin/users/${users.craig.id}`);

	// A pending edit must not block leaving the page once the user is gone
	await page.getByLabel('First name').fill('Changed');

	await page.getByRole('button', { name: 'More actions' }).click();
	await page.getByRole('menuitem', { name: 'Delete user' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		`${users.craig.firstname} ${users.craig.lastname} deleted`
	);
	await page.waitForURL((url) => url.pathname === '/settings/admin/users');
	await expect(
		page.getByRole('row', {
			name: `${users.craig.firstname} ${users.craig.lastname}`
		})
	).not.toBeVisible();
});

test('Create one time access token from the details page', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);

	await page.getByRole('button', { name: 'Login code' }).click();
	await page.getByRole('button', { name: 'Create code' }).click();

	await expect(page.getByTestId('login-code')).toBeVisible();
});

test('Update user', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Open' }).click();

	await page.getByLabel('First name').fill('Crack');
	await page.getByLabel('Last name').fill('Apple');
	await page.getByLabel('Display name').fill('Crack Apple');
	await page.getByLabel('Email').fill('crack.apple@test.com');
	await page.getByLabel('Username').fill('crack');
	await saveUnsavedChanges(page);
});

// The form has no language field, so saving it must not reset the language the user picked
test('Saving a user keeps their language', async ({ page }) => {
	const user = users.craig;
	await page.request.patch(`/api/admin/users/${user.id}`, { data: { locale: 'en' } });
	await page.goto(`/settings/admin/users/${user.id}`);

	await page.getByLabel('First name').fill('Crack');
	await saveUnsavedChanges(page);

	const saved = await (await page.request.get(`/api/admin/users/${user.id}`)).json();
	expect(saved.locale).toBe('en');
});

test('Update user fails with already taken email', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Open' }).click();

	await page.getByLabel('Email').fill(users.tim.email);
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(
		unsavedChangesBar(page).getByText('Email is already in use', { exact: true })
	).toBeVisible();
});

test('Update user fails with already taken username', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Open' }).click();

	await page.getByLabel('Username').fill(users.tim.username);
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(
		unsavedChangesBar(page).getByText('Username is already in use', { exact: true })
	).toBeVisible();
});

test('Update user fails with already taken username in different casing', async ({ page }) => {
	const user = users.craig;

	await page.goto('/settings/admin/users');

	await page
		.getByRole('row', { name: `${user.firstname} ${user.lastname}` })
		.getByRole('button')
		.click();
	await page.getByRole('menuitem', { name: 'Open' }).click();

	await page.getByLabel('Username').fill(users.tim.username.toUpperCase());
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(
		unsavedChangesBar(page).getByText('Username is already in use', { exact: true })
	).toBeVisible();
});

test('Update user custom claims', async ({ page }) => {
	await setAppConfigValues(page.request, { enabledFeatures: '["customClaims"]' });
	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await page.getByRole('tab', { name: /^Custom claims/ }).click();

	// Add two custom claims
	await page.getByRole('button', { name: 'Add custom claim' }).click();

	await page.getByPlaceholder('Key').fill('customClaim1');
	await page.getByPlaceholder('Value').fill('customClaim1_value');

	await page.getByRole('button', { name: 'Add another' }).click();
	await page.getByPlaceholder('Key').nth(1).fill('customClaim2');
	await page.getByPlaceholder('Value').nth(1).fill('customClaim2_value');

	await saveUnsavedChanges(page);

	await page.reload();

	// Check if custom claims are saved
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim1');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim1_value');
	await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('customClaim2_value');

	// Remove one custom claim
	await page.getByLabel('Remove custom claim').first().click();
	await saveUnsavedChanges(page);

	await page.reload();

	// Check if custom claim is removed
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim2_value');
});

test('Update user group assignments', async ({ page }) => {
	const user = users.craig;
	await page.goto(`/settings/admin/users/${user.id}`);

	await page.getByRole('tab', { name: 'Groups' }).click();

	// Memberships save at once: Designers is added through the picker and Developers is removed from the list
	await page.getByRole('button', { name: 'Add to groups…' }).click();
	const picker = page.getByRole('dialog');
	await picker.getByRole('checkbox', { name: userGroups.designers.friendlyName }).click();
	await picker.getByRole('button', { name: 'Add to 1 group' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Group memberships updated');

	await page
		.getByRole('button', {
			name: `Remove ${user.firstname} from ${userGroups.developers.friendlyName}`
		})
		.click();

	await expect(
		page.getByText(`${user.firstname} removed from ${userGroups.developers.friendlyName}`, {
			exact: true
		})
	).toBeVisible();
	await page.reload();

	const groups = page.getByRole('tabpanel');
	await expect(groups.getByRole('link', { name: userGroups.designers.friendlyName })).toBeVisible();
	await expect(groups.getByRole('link', { name: userGroups.developers.friendlyName })).toHaveCount(
		0
	);
});

test('Admin can view another user passkeys', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await page.getByRole('tab', { name: 'Passkeys' }).click();

	await expect(page.getByText('Passkey 2', { exact: true })).toBeVisible();
	await expect(page.getByText(/Added on/)).toBeVisible();
});

test('Admin can delete another user passkey and audit log is created', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await page.getByRole('tab', { name: 'Passkeys' }).click();

	await page.getByRole('button', { name: 'Delete Passkey 2' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Passkey deleted successfully');
	await expect(page.getByText('Passkey 2', { exact: true })).not.toBeVisible();

	const auditLogResponse = await page.request.get('/api/admin/audit-logs');
	expect(auditLogResponse.ok()).toBeTruthy();

	const auditLogs = await auditLogResponse.json();
	expect(
		auditLogs.data.some(
			(log: { event: string; userID: string; data?: { passkeyName?: string } }) =>
				log.event === 'PASSKEY_REMOVED' &&
				log.userID === users.craig.id &&
				log.data?.passkeyName === 'Passkey 2'
		)
	).toBeTruthy();
});

test('Non-admin cannot use admin passkey endpoints', async ({ page }) => {
	await authUtil.changeUser(page, 'craig');

	const listResponse = await page.request.get(
		`/api/admin/users/${users.tim.id}/webauthn-credentials`
	);
	expect(listResponse.status()).toBe(403);

	const deleteResponse = await page.request.delete(
		`/api/admin/users/${users.tim.id}/webauthn-credentials/test-passkey-id`
	);
	expect(deleteResponse.status()).toBe(403);
});

test('Reorder table columns in the column picker', async ({ page }) => {
	await page.goto('/settings/admin/users');

	const headerNames = async () =>
		(await page.getByRole('columnheader').allInnerTexts()).map((t) => t.trim()).filter(Boolean);
	await expect.poll(headerNames).toEqual(['User', 'Username', 'Role', 'Status']);

	await page.getByRole('button', { name: 'Columns' }).click();
	const grip = (name: string) => page.getByRole('button', { name: `Reorder ${name}` });

	await grip('Status').press('ArrowUp');
	await grip('Status').press('ArrowUp');
	await expect.poll(headerNames).toEqual(['User', 'Status', 'Username', 'Role']);

	await grip('Role').dragTo(grip('Status'));
	await expect.poll(headerNames).toEqual(['User', 'Role', 'Status', 'Username']);

	await page.reload();
	await expect.poll(headerNames).toEqual(['User', 'Role', 'Status', 'Username']);

	await page.getByRole('button', { name: 'Columns' }).click();
	await page.getByRole('button', { name: 'Reset' }).click();
	await expect.poll(headerNames).toEqual(['User', 'Username', 'Role', 'Status']);
});

test('Create user keeps default groups and sends the selection atomically', async ({ page }) => {
	await setAppConfigValues(page.request, {
		signupDefaultUserGroupIDs: JSON.stringify([userGroups.developers.id])
	});
	const user = users.steve;
	await page.goto('/settings/admin/users');
	await page.getByRole('button', { name: 'Add user', exact: true }).click();
	const dialog = page.getByRole('dialog');
	const groupPicker = dialog.getByLabel('Groups', { exact: true });
	await expect(groupPicker).toContainText(userGroups.developers.friendlyName);
	await page.getByLabel('First name').fill(user.firstname);
	await page.getByLabel('Last name').fill(user.lastname);
	await page.getByLabel('Email').fill(user.email);
	await page.getByLabel('Username').fill(user.username);
	await groupPicker.click();
	await page.getByRole('option', { name: userGroups.designers.friendlyName, exact: true }).click();
	await groupPicker.click();
	const created = page.waitForResponse(
		(r) => r.url().endsWith('/api/users') && r.request().method() === 'POST'
	);
	await dialog.getByRole('button', { name: 'Add user', exact: true }).click();
	const response = await created;
	expect(response.request().postDataJSON().userGroupIds.sort()).toEqual(
		[userGroups.developers.id, userGroups.designers.id].sort()
	);
	const saved = await response.json();
	expect(saved.userGroups.map((g: { id: string }) => g.id).sort()).toEqual(
		[userGroups.developers.id, userGroups.designers.id].sort()
	);
	await expect(dialog).not.toBeVisible();
	await page.goto('/settings/admin/users');
	await page.getByRole('button', { name: 'Add user', exact: true }).click();
	await expect(groupPicker).toContainText(userGroups.developers.friendlyName);
	await expect(groupPicker).not.toContainText(userGroups.designers.friendlyName);
});
