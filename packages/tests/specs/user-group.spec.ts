import test, { expect } from '@playwright/test';
import { oidcClients, userGroups, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import { setAppConfigValues } from '../utils/app-config.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

test('Create user group', async ({ page }) => {
	await page.goto('/settings/admin/user-groups');
	const group = userGroups.humanResources;

	await page.getByRole('button', { name: 'Add group' }).click();
	await page.getByLabel('Friendly name').fill(group.friendlyName);

	await page.getByRole('dialog').getByRole('button', { name: 'Add group' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(`${group.friendlyName} added`);

	await page.waitForURL('/settings/admin/user-groups/*');

	await expect(page.getByLabel('Friendly Name')).toHaveValue(group.friendlyName);
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue(group.name);
});

test('Edit user group', async ({ page }) => {
	await page.goto('/settings/admin/user-groups');
	const group = userGroups.developers;

	await page.getByRole('row', { name: group.friendlyName }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Open' }).click();

	await page.getByLabel('Friendly Name').fill('Developers updated');

	await expect(page.getByLabel('Name', { exact: true })).toHaveValue(group.name);

	await page.getByLabel('Name', { exact: true }).fill('developers_updated');

	await saveUnsavedChanges(page);
	await expect(page.getByLabel('Friendly Name')).toHaveValue('Developers updated');
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('developers_updated');
});

test('Update user group members', async ({ page }) => {
	const group = userGroups.designers;
	await page.goto(`/settings/admin/user-groups/${group.id}`);
	await page.getByRole('tab', { name: 'Members' }).click();

	// Members save at once: Craig is added through the picker and Tim is removed from the list
	await page.getByRole('button', { name: 'Add members…' }).click();
	const picker = page.getByRole('dialog');
	await picker
		.getByRole('checkbox', { name: `${users.craig.firstname} ${users.craig.lastname}` })
		.click();
	await picker.getByRole('button', { name: 'Add 1 member' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Members added');

	await page
		.getByRole('button', {
			name: `Remove ${users.tim.firstname} ${users.tim.lastname} from ${group.friendlyName}`
		})
		.click();

	await page.reload();

	const members = page.getByRole('tabpanel');
	await expect(members.getByText(`${users.craig.firstname} ${users.craig.lastname}`)).toBeVisible();
	await expect(members.getByText(`${users.tim.firstname} ${users.tim.lastname}`)).toHaveCount(0);
});

test('Delete user group', async ({ page }) => {
	const group = userGroups.developers;
	await page.goto('/settings/admin/user-groups');

	await page.getByRole('row', { name: group.friendlyName }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(`${group.friendlyName} deleted`);
	await expect(page.getByRole('row', { name: group.friendlyName })).not.toBeVisible();
});

test('Delete user group from the details page', async ({ page }) => {
	const group = userGroups.developers;
	await page.goto(`/settings/admin/user-groups/${group.id}`);

	await page.getByRole('button', { name: 'More actions' }).click();
	await page.getByRole('menuitem', { name: 'Delete group' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(`${group.friendlyName} deleted`);
	await page.waitForURL((url) => url.pathname === '/settings/admin/user-groups');
	await expect(page.getByRole('row', { name: group.friendlyName })).not.toBeVisible();
});

test('Update user group custom claims', async ({ page }) => {
	await setAppConfigValues(page.request, { enabledFeatures: '["customClaims"]' });
	await page.goto(`/settings/admin/user-groups/${userGroups.designers.id}`);
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
	await page.waitForLoadState('networkidle');

	// Check if custom claims are saved
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim1');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim1_value');
	await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('customClaim2_value');

	// Remove one custom claim
	await page.getByLabel('Remove custom claim').first().click();
	await saveUnsavedChanges(page);

	await page.reload();
	await page.waitForLoadState('networkidle');

	// Check if custom claim is removed
	await expect(page.getByPlaceholder('Key').first()).toHaveValue('customClaim2');
	await expect(page.getByPlaceholder('Value').first()).toHaveValue('customClaim2_value');
});

test('Update user group allowed clients', async ({ page }) => {
	await page.goto(`/settings/admin/user-groups/${userGroups.designers.id}`);

	await page.getByRole('tab', { name: 'Allowed clients' }).click();

	// Clients open to everyone are named in a note instead of being listed
	await expect(page.getByText(/are open to all users|is open to all users/)).toContainText(
		oidcClients.nextcloud.name
	);

	await page.getByRole('button', { name: 'Allow clients…' }).click();
	const picker = page.getByRole('dialog');
	await picker.getByRole('checkbox', { name: oidcClients.tailscale.name }).click();
	await picker.getByRole('button', { name: 'Allow 1 client' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Allowed clients updated');

	await page.reload();

	await expect(page.getByRole('link', { name: oidcClients.tailscale.name })).toBeVisible();
});
