import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(cleanupBackend);

test.describe('LDAP Integration', () => {
  test.skip(process.env.SKIP_LDAP_TESTS === 'true', 'Skipping LDAP tests due to SKIP_LDAP_TESTS environment variable');

  test('LDAP configuration is working properly', async ({ page }) => {
    await page.goto('/settings/admin/application-configuration');

    await page.getByRole('button', { name: 'Expand card' }).nth(2).click();

    await expect(page.getByRole('button', { name: 'Disable', exact: true })).toBeVisible();
    await expect(page.getByLabel('LDAP URL')).toHaveValue(/ldap:\/\/.*/);
    await expect(page.getByLabel('LDAP Base DN')).not.toBeEmpty();

    await expect(page.getByLabel('User Unique Identifier Attribute')).not.toBeEmpty();
    await expect(page.getByLabel('Username Attribute')).not.toBeEmpty();
    await expect(page.getByLabel('User Mail Attribute')).not.toBeEmpty();
    await expect(page.getByLabel('Group Name Attribute')).not.toBeEmpty();

    const syncButton = page.getByRole('button', { name: 'Sync now' });
    await syncButton.click();
    await expect(page.locator('[data-type="success"]')).toHaveText('LDAP sync finished');
  });

  test('LDAP users are synced into PocketID', async ({ page }) => {
    // Navigate to user management
    await page.goto('/settings/admin/users');

    // Verify the LDAP users exist
    await expect(page.getByText('testuser1@pocket-id.org')).toBeVisible();
    await expect(page.getByText('testuser2@pocket-id.org')).toBeVisible();

    // Check LDAP user details
    await page.getByRole('row', { name: 'testuser1' }).getByRole('button').click();
    await page.getByRole('menuitem', { name: 'Edit' }).click();

    // Verify user source is LDAP
    await expect(page.getByText('LDAP').first()).toBeVisible();

    // Verify essential fields are filled
    await expect(page.getByLabel('Username')).not.toBeEmpty();
    await expect(page.getByLabel('Email')).not.toBeEmpty();
  });

  test('LDAP groups are synced into PocketID', async ({ page }) => {
    // Navigate to user groups
    await page.goto('/settings/admin/user-groups');

    // Verify LDAP groups exist
    await expect(page.getByRole('cell', { name: 'test_group' }).first()).toBeVisible();
    await expect(page.getByRole('cell', { name: 'admin_group' }).first()).toBeVisible();

    await page.getByRole('row', { name: 'test_group' }).getByRole('button', { name: 'Toggle menu' }).click();
    await page.getByRole('menuitem', { name: 'Edit' }).click();

    // Verify group source is LDAP
    await expect(page.getByText('LDAP').first()).toBeVisible();
  });

  test('LDAP users cannot be modified in PocketID', async ({ page }) => {
    // Navigate to LDAP user details
    await page.goto('/settings/admin/users');
    await page.waitForLoadState('networkidle');

    await page.getByRole('row', { name: 'testuser1' }).getByRole('button').click();
    await page.getByRole('menuitem', { name: 'Edit' }).click();

    // Verify key fields are disabled
    const usernameInput = page.getByLabel('Username');
    await expect(usernameInput).toBeDisabled();
  });

  test('LDAP groups cannot be modified in PocketID', async ({ page }) => {
    // Navigate to LDAP group details
    await page.goto('/settings/admin/user-groups');
    await page.waitForLoadState('networkidle');

    await page.getByRole('row', { name: 'test_group' }).getByRole('button', { name: 'Toggle menu' }).click();
    await page.getByRole('menuitem', { name: 'Edit' }).click();

    // Verify key fields are disabled
    const nameInput = page.getByLabel('Name', { exact: true });
    await expect(nameInput).toBeDisabled();
  });
});
