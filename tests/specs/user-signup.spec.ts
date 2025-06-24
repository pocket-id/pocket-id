import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(cleanupBackend);

// Disable authentication for these tests since we're testing signup
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('User Signup', () => {
  async function setSignupMode(page: any, mode: 'disabled' | 'withtoken' | 'open') {
    // Login as admin first
    await page.goto('/login');
    await (await passkeyUtil.init(page)).addPasskey();
    await page.getByRole('button', { name: 'Authenticate' }).click();
    await page.waitForURL('/settings/account');

    await page.goto('/settings/admin/application-configuration');

    await page.getByLabel('Enable user signups').click();
    await page.getByRole('option', { name: mode }).click();

    await page.getByRole('button', { name: 'Save' }).first().click();
    await expect(page.locator('[data-type="success"]')).toHaveText('Application configuration updated successfully');

    await page.context().clearCookies();
  }

  async function createSignupToken(page: any): Promise<string> {
    // Login as admin
    await page.goto('/login');
    await (await passkeyUtil.init(page)).addPasskey();
    await page.getByRole('button', { name: 'Authenticate' }).click();
    await page.waitForURL('/settings/account');

    // Navigate to users page
    await page.goto('/settings/admin/users');

    // Create signup token
    await page.getByRole('button', { name: 'Create Signup Token' }).or(page.getByText('Create Token')).click();

    // Set token details
    await page.getByLabel('Expiration').selectOption('One day');
    await page.getByLabel('Usage Limit').fill('1');
    await page.getByRole('button', { name: 'Create' }).click();

    // Get the token from the modal/response
    const tokenElement = page.locator('[data-testid="signup-token"]').or(page.locator('.font-mono'));
    const token = await tokenElement.textContent();

    // Close modal
    await page.getByRole('button', { name: 'Close' }).click();

    // Clear cookies
    await page.context().clearCookies();

    return token!;
  }

  test('Signup is disabled - shows error message', async ({ page }) => {
    await setSignupMode(page, 'disabled');

    await page.goto('/signup');

    await expect(page.getByText('User signups are currently disabled')).toBeVisible();
  });

  test('Signup with token - success flow', async ({ page }) => {
    await setSignupMode(page, 'withtoken');
    const token = await createSignupToken(page);

    // Navigate to signup with token
    await page.goto(`/st/${token}`);

    // Fill out signup form
    await page.getByLabel('First name').fill('John');
    await page.getByLabel('Last name').fill('Doe');
    await page.getByLabel('Username').fill('johndoe');
    await page.getByLabel('Email').fill('john.doe@test.com');

    // Submit form
    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should redirect to add passkey page
    await page.waitForURL('/signup/add-passkey');
    await expect(page.getByText('Setup your passkey')).toBeVisible();
  });

  test('Signup with token - invalid token shows error', async ({ page }) => {
    await setSignupMode(page, 'withtoken');

    // Try to access signup with invalid token
    await page.goto('/st/invalid-token-123');

    // Should show error message
    await expect(page.getByText('Signup requires valid token')).or(page.getByText('Invalid or expired token')).toBeVisible();

    // Should show login button
    await expect(page.getByRole('button', { name: 'Go to Login' })).toBeVisible();
  });

  test('Signup with token - no token in URL shows error', async ({ page }) => {
    await setSignupMode(page, 'withtoken');

    // Try to access signup page without token
    await page.goto('/signup');

    // Should show error message
    await expect(page.getByText('Signup requires valid token')).toBeVisible();

    // Should show login button
    await expect(page.getByRole('button', { name: 'Go to Login' })).toBeVisible();
  });

  test('Open signup - success flow', async ({ page }) => {
    await setSignupMode(page, 'open');

    // Navigate to signup page
    await page.goto('/signup');

    // Verify signup form is visible
    await expect(page.getByText('Create your account to get started')).toBeVisible();

    // Fill out signup form
    await page.getByLabel('First name').fill('Jane');
    await page.getByLabel('Last name').fill('Smith');
    await page.getByLabel('Username').fill('janesmith');
    await page.getByLabel('Email').fill('jane.smith@test.com');

    // Submit form
    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should redirect to add passkey page
    await page.waitForURL('/signup/add-passkey');
    await expect(page.getByText('Setup your passkey')).toBeVisible();
  });

  test('Open signup - validation errors', async ({ page }) => {
    await setSignupMode(page, 'open');

    await page.goto('/signup');

    // Try to submit empty form
    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should show validation errors
    await expect(page.getByText('Required')).toBeVisible();
  });

  test('Open signup - duplicate email shows error', async ({ page }) => {
    await setSignupMode(page, 'open');

    await page.goto('/signup');

    // Fill form with existing user's email (from test data)
    await page.getByLabel('First name').fill('Test');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('testuser123');
    await page.getByLabel('Email').fill('tim@pocket-id.org'); // Existing user email

    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should show error
    await expect(page.locator('[data-type="error"]')).toHaveText('Email is already in use');
  });

  test('Open signup - duplicate username shows error', async ({ page }) => {
    await setSignupMode(page, 'open');

    await page.goto('/signup');

    // Fill form with existing user's username
    await page.getByLabel('First name').fill('Test');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('tim'); // Existing username
    await page.getByLabel('Email').fill('newuser@test.com');

    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should show error
    await expect(page.locator('[data-type="error"]')).toHaveText('Username is already in use');
  });

  test('Login page shows signup button only when open signup enabled', async ({ page }) => {
    // Test with open signup
    await setSignupMode(page, 'open');
    await page.goto('/login');
    await expect(page.getByRole('link', { name: 'Sign up' })).toBeVisible();

    // Test with token signup
    await setSignupMode(page, 'withtoken');
    await page.goto('/login');
    await expect(page.getByRole('link', { name: 'Sign up' })).not.toBeVisible();

    // Test with disabled signup
    await setSignupMode(page, 'disabled');
    await page.goto('/login');
    await expect(page.getByRole('link', { name: 'Sign up' })).not.toBeVisible();
  });

  test('Complete signup flow with passkey creation', async ({ page }) => {
    await setSignupMode(page, 'open');

    // Complete signup
    await page.goto('/signup');
    await page.getByLabel('First name').fill('Complete');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('completeuser');
    await page.getByLabel('Email').fill('complete.user@test.com');
    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should be on add passkey page
    await page.waitForURL('/signup/add-passkey');

    // Add passkey
    await (await passkeyUtil.init(page)).addPasskey('completeuser');
    await page.getByRole('button', { name: 'Add Passkey' }).click();

    // Should redirect to settings
    await page.waitForURL('/settings/account');
    await expect(page.getByText('Complete User')).toBeVisible();
  });

  test('Skip passkey creation during signup', async ({ page }) => {
    await setSignupMode(page, 'open');

    // Complete signup
    await page.goto('/signup');
    await page.getByLabel('First name').fill('Skip');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('skipuser');
    await page.getByLabel('Email').fill('skip.user@test.com');
    await page.getByRole('button', { name: 'Sign up' }).click();

    // Should be on add passkey page
    await page.waitForURL('/signup/add-passkey');

    // Skip passkey creation
    await page.getByRole('button', { name: 'Skip for now' }).click();

    // Should redirect to settings
    await page.waitForURL('/settings/account');
    await expect(page.getByText('Skip User')).toBeVisible();
  });

  test('Token usage limit is enforced', async ({ page }) => {
    await setSignupMode(page, 'withtoken');
    const token = await createSignupToken(page);

    // First signup - should succeed
    await page.goto(`/st/${token}`);
    await page.getByLabel('First name').fill('First');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('firstuser');
    await page.getByLabel('Email').fill('first.user@test.com');
    await page.getByRole('button', { name: 'Sign up' }).click();
    await page.waitForURL('/signup/add-passkey');

    // Clear session and try again with same token
    await page.context().clearCookies();
    await page.goto(`/st/${token}`);

    // Should show error that token is used up
    await expect(page.getByText('Invalid or expired token')).or(page.getByText('Token has been used')).toBeVisible();
  });
});
