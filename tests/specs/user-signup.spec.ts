import test, { expect, type Page } from '@playwright/test';
import { signupTokens, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

async function setSignupMode(page: Page, mode: 'Disabled' | 'Signup with token' | 'Open Signup') {
	await page.goto('/settings/admin/application-configuration');

	const signupCard = page.locator('div[data-slot="card"]:has-text("Signup")');
	await signupCard.getByRole('button', { name: 'Expand card' }).click();

	await signupCard.getByLabel('Enable User Signups').click();
	await page.getByRole('option', { name: mode }).click();

	await signupCard.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]').last()).toHaveText(
		'Signup settings updated successfully.'
	);

	await page.waitForLoadState('networkidle');

	await page.context().clearCookies();
	await page.goto('/login');
}

test.describe('Initial User Signup', () => {
	test.beforeEach(async ({ page }) => {
		await page.context().clearCookies();
	});

	test('Initial Signup - success flow', async ({ page }) => {
		await cleanupBackend(true);
		await page.goto('/setup');
		await page.getByLabel('First name').fill('Jane');
		await page.getByLabel('Last name').fill('Smith');
		await page.getByLabel('Username').fill('janesmith');
		await page.getByLabel('Email').fill('jane.smith@test.com');
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await page.waitForURL('/signup/add-passkey');
		await expect(page.getByText('Set up your passkey')).toBeVisible();
	});

	test('Initial Signup - setup already completed', async ({ page }) => {
		await cleanupBackend();
		await page.goto('/setup');
		await page.getByLabel('First name').fill('Test');
		await page.getByLabel('Last name').fill('User');
		await page.getByLabel('Username').fill('testuser123');
		await page.getByLabel('Email').fill(users.tim.email);
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await expect(page.getByText('Setup already completed')).toBeVisible();
	});
});

test.describe('User Signup and Configuration', () => {
	test.beforeEach(() => {
		cleanupBackend();
	});

	test.describe('Signup Defaults Configuration', () => {
		test.beforeEach(async ({ page }) => {
			await page.goto('/settings/admin/application-configuration');
			const signupCard = page.locator('div[data-slot="card"]:has-text("Signup")');
			await signupCard.getByRole('button', { name: 'Expand card' }).click();
			await signupCard.locator('button[data-select-trigger]').click();
			await page.getByRole('option', { name: 'Open Signup' }).click();
			await signupCard.getByRole('button', { name: 'Save' }).click();
			await expect(page.locator('[data-type="success"]').last()).toHaveText(
				'Signup settings updated successfully.'
			);
		});

		test('should save default user groups for new signups', async ({ page }) => {
			const signupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup")');

			await signupDefaultsCard.locator('[data-slot="dropdown-menu-trigger"]').click();
			await page.getByRole('menuitemcheckbox', { name: 'Developers' }).click();
			await page.getByRole('menuitemcheckbox', { name: 'Designers' }).click();
			await page.keyboard.press('Escape');

			await signupDefaultsCard.locator('form').getByRole('button', { name: 'Save' }).click();

			await expect(page.locator('[data-type="success"]').last()).toHaveText(
				'Signup settings updated successfully.'
			);

			await page.reload();
			await page.waitForLoadState('networkidle');

			const updatedSignupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup")');
			await updatedSignupDefaultsCard.locator('[data-slot="dropdown-menu-trigger"]').click();

			await expect(page.getByRole('menuitemcheckbox', { name: 'Developers' })).toBeChecked();
			await expect(page.getByRole('menuitemcheckbox', { name: 'Designers' })).toBeChecked();
		});

		test('should save default custom claims for new signups', async ({ page }) => {
			const signupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup")');

			await signupDefaultsCard.getByRole('button', { name: 'Add custom claim' }).click();
			await signupDefaultsCard.getByPlaceholder('Key').fill('test-claim');
			await signupDefaultsCard.getByPlaceholder('Value').fill('test-value');
			await signupDefaultsCard.getByRole('button', { name: 'Add another' }).click();
			await signupDefaultsCard.getByPlaceholder('Key').nth(1).fill('another-claim');
			await signupDefaultsCard.getByPlaceholder('Value').nth(1).fill('another-value');

			await signupDefaultsCard.locator('form').getByRole('button', { name: 'Save' }).click();

			await expect(page.locator('[data-type="success"]').last()).toHaveText(
				'Signup settings updated successfully.'
			);

			await page.reload();
			await page.waitForLoadState('networkidle');

			const updatedSignupDefaultsCard = page.locator('div[data-slot="card"]:has-text("Signup")');

			await expect(updatedSignupDefaultsCard.getByPlaceholder('Key').first()).toHaveValue('test-claim');
			await expect(updatedSignupDefaultsCard.getByPlaceholder('Value').first()).toHaveValue('test-value');
			await expect(updatedSignupDefaultsCard.getByPlaceholder('Key').nth(1)).toHaveValue('another-claim');
			await expect(updatedSignupDefaultsCard.getByPlaceholder('Value').nth(1)).toHaveValue('another-value');
		});
	});

	test.describe('Signup Flows', () => {
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

			await expect(
				page.getByText('A valid signup token is required to create an account.')
			).toBeVisible();
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

			await expect(page.getByText('Skip Passkey Setup')).toBeVisible();
			await page.getByRole('button', { name: 'Skip for now' }).nth(1).click();

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
});
