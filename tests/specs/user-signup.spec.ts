import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';
import { users, signupTokens } from 'data';

test.beforeEach(cleanupBackend);

test.describe('User Signup', () => {
  async function setSignupMode(page: any, mode: 'Disabled' | 'Signup with token' | 'Open Signup') {
    await page.goto('/settings/admin/application-configuration');

    await page.getByLabel('Enable user signups').click();
    await page.getByRole('option', { name: mode }).click();

    await page.getByRole('button', { name: 'Save' }).first().click();
    await expect(page.locator('[data-type="success"]')).toHaveText('Application configuration updated successfully');
    await page.waitForLoadState('networkidle');

    await page.context().clearCookies();
    await page.goto('/login');
  }

  test('Signup is disabled - shows error message', async ({ page }) => {
    await setSignupMode(page, 'Disabled');

    await page.goto('/signup');

    await expect(page.getByText('User signups are currently disabled')).toBeVisible();
  });

  test('Signup with token - success flow', async ({ page }) => {
    await setSignupMode(page, 'Signup with token');

    await page.goto(`/st/${signupTokens.valid.token}`);

    await page.getByLabel('First name').fill('John');
    await page.getByLabel('Last name').fill('Doe');
    await page.getByLabel('Username').fill('johndoe');
    await page.getByLabel('Email').fill('john.doe@test.com');

    await page.getByRole('button', { name: 'Sign Up' }).click();

    await page.waitForURL('/signup/add-passkey');
    await expect(page.getByText('Set up your passkey')).toBeVisible();
  });

  test('Signup with token - invalid token shows error', async ({ page }) => {
    await setSignupMode(page, 'Signup with token');

    await page.goto('/st/invalid-token-123');
    await page.getByLabel('First name').fill('Complete');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('completeuser');
    await page.getByLabel('Email').fill('complete.user@test.com');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Token is invalid or expired.')).toBeVisible();
  });

  test('Signup with token - no token in URL shows error', async ({ page }) => {
    await setSignupMode(page, 'Signup with token');

    await page.goto('/signup');

    await expect(page.getByText('A valid signup token is required to create an account.')).toBeVisible();
  });

  test('Open signup - success flow', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');

    await expect(page.getByText('Create your account to get started')).toBeVisible();

    await page.getByLabel('First name').fill('Jane');
    await page.getByLabel('Last name').fill('Smith');
    await page.getByLabel('Username').fill('janesmith');
    await page.getByLabel('Email').fill('jane.smith@test.com');

    await page.getByRole('button', { name: 'Sign Up' }).click();

    await page.waitForURL('/signup/add-passkey');
    await expect(page.getByText('Set up your passkey')).toBeVisible();
  });

  test('Open signup - validation errors', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');

    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Invalid input').first()).toBeVisible();
  });

  test('Open signup - duplicate email shows error', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');

    await page.getByLabel('First name').fill('Test');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('testuser123');
    await page.getByLabel('Email').fill(users.tim.email);

    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Email is already in use.')).toBeVisible();
  });

  test('Open signup - duplicate username shows error', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');

    await page.getByLabel('First name').fill('Test');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill(users.tim.username);
    await page.getByLabel('Email').fill('newuser@test.com');

    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Username is already in use.')).toBeVisible();
  });

  test('Complete signup flow with passkey creation', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');
    await page.getByLabel('First name').fill('Complete');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('completeuser');
    await page.getByLabel('Email').fill('complete.user@test.com');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await page.waitForURL('/signup/add-passkey');

    await (await passkeyUtil.init(page)).addPasskey('timNew');
    await page.getByRole('button', { name: 'Add Passkey' }).click();

    await page.waitForURL('/settings/account');
    await expect(page.getByText('Single Passkey Configured')).toBeVisible();
  });

  test('Skip passkey creation during signup', async ({ page }) => {
    await setSignupMode(page, 'Open Signup');

    await page.goto('/signup');
    await page.getByLabel('First name').fill('Skip');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('skipuser');
    await page.getByLabel('Email').fill('skip.user@test.com');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await page.waitForURL('/signup/add-passkey');

    await page.getByRole('button', { name: 'Skip for now' }).click();

    await page.waitForURL('/settings/account');
    await expect(page.getByText('Passkey missing')).toBeVisible();
  });

  test('Token usage limit is enforced', async ({ page }) => {
    await setSignupMode(page, 'Signup with token');

    await page.goto(`/st/${signupTokens.fullyUsed.token}`);
    await page.getByLabel('First name').fill('Complete');
    await page.getByLabel('Last name').fill('User');
    await page.getByLabel('Username').fill('completeuser');
    await page.getByLabel('Email').fill('complete.user@test.com');
    await page.getByRole('button', { name: 'Sign Up' }).click();

    await expect(page.getByText('Token is invalid or expired.')).toBeVisible();
  });
});
