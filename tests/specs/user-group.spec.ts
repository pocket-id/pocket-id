import test, { expect } from '@playwright/test';
import { customFields, oidcClients, userGroups, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('Create user group', async ({ page }) => {
	await page.goto('/settings/admin/user-groups');
	const group = userGroups.humanResources;

	await page.getByRole('button', { name: 'Add Group' }).click();
	await page.getByLabel('Friendly Name').fill(group.friendlyName);

	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('User group created successfully');

	await page.waitForURL('/settings/admin/user-groups/*');

	await expect(page.getByLabel('Friendly Name')).toHaveValue(group.friendlyName);
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue(group.name);
});

test('Edit user group', async ({ page }) => {
	await page.goto('/settings/admin/user-groups');
	const group = userGroups.developers;

	await page.getByRole('row', { name: group.name }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Edit' }).click();

	await page.getByLabel('Friendly Name').fill('Developers updated');

	await expect(page.getByLabel('Name', { exact: true })).toHaveValue(group.name);

	await page.getByLabel('Name', { exact: true }).fill('developers_updated');

	await page.getByRole('button', { name: 'Save' }).nth(0).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('User group updated successfully');
	await expect(page.getByLabel('Friendly Name')).toHaveValue('Developers updated');
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('developers_updated');
});

test('Update user group users', async ({ page }) => {
	const group = userGroups.designers;
	await page.goto(`/settings/admin/user-groups/${group.id}`);

	await page.getByRole('row', { name: users.tim.username }).getByRole('checkbox').click();
	await page.getByRole('row', { name: users.craig.username }).getByRole('checkbox').click();

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('Users updated successfully');

	await page.reload();

	await expect(
		page.getByRole('row', { name: users.tim.username }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'unchecked');
	await expect(
		page.getByRole('row', { name: users.craig.username }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked');
});

test('Delete user group', async ({ page }) => {
	const group = userGroups.developers;
	await page.goto('/settings/admin/user-groups');

	await page.getByRole('row', { name: group.name }).getByRole('button').click();
	await page.getByRole('menuitem', { name: 'Delete' }).click();
	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText('User group deleted successfully');
	await expect(page.getByRole('row', { name: group.name })).not.toBeVisible();
});

test('Update user group custom fields', async ({ page }) => {
	await page.goto(`/settings/admin/user-groups/${userGroups.designers.id}`);

	await expect(
		page.getByRole('switch', { name: customFields.elevatedRights.displayName })
	).not.toBeChecked();

	await page.getByRole('switch', { name: customFields.elevatedRights.displayName }).click();
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.locator('[data-type="success"]')).toHaveText('User group updated successfully');

	await page.reload();

	await expect(
		page.getByRole('switch', { name: customFields.elevatedRights.displayName })
	).toBeChecked();
});

test('Update user group allowed user groups', async ({ page }) => {
	await page.goto(`/settings/admin/user-groups/${userGroups.designers.id}`);

	await page.getByRole('button', { name: 'Expand card' }).click();

	// Unrestricted OIDC clients should be checked and disabled
	const nextcloudRow = page
		.getByRole('row', { name: oidcClients.nextcloud.name })
		.getByRole('checkbox');
	await expect(nextcloudRow).toHaveAttribute('data-state', 'checked');
	await expect(nextcloudRow).toBeDisabled();

	await page.getByRole('row', { name: oidcClients.tailscale.name }).getByRole('checkbox').click();
	await page.getByRole('row', { name: oidcClients.immich.name }).getByRole('checkbox').click();

	await page.getByRole('button', { name: 'Save' }).nth(2).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Allowed OIDC clients updated successfully'
	);

	await page.reload();

	await expect(
		page.getByRole('row', { name: oidcClients.tailscale.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked');
	await expect(
		page.getByRole('row', { name: oidcClients.immich.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'unchecked');
});
